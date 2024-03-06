package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	lua "github.com/yuin/gopher-lua"
)

// Function to recursively convert Lua table to Go map
func luaValueToInterface(value lua.LValue) interface{} {
	switch value.Type() {
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTNumber:
		return float64(value.(lua.LNumber))
	case lua.LTString:
		return string(value.(lua.LString))
	case lua.LTTable:
		return luaTableToMap(value.(*lua.LTable))
	default:
		return nil
	}
}

func luaTableToMap(table *lua.LTable) interface{} {
	if table.MaxN() > 0 {
		// If the table has sequential integer keys starting from 1, treat it as an array
		arr := make([]interface{}, table.MaxN())
		table.ForEach(func(i lua.LValue, value lua.LValue) {
			idx := int(i.(lua.LNumber))
			arr[idx-1] = luaValueToInterface(value)
		})
		return arr
	}

	// If not, treat it as a map
	result := make(map[string]interface{})

	table.ForEach(func(key, value lua.LValue) {
		result[key.String()] = luaValueToInterface(value)
	})

	return result
}

func findAllLuaAssetModules(prefix string) ([]string, error) {
	var result []string
	for _, name := range AssetNames() {
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".lua") {
			result = append(result, name)
		}
	}
	return result, nil
}

// Load custom Lua modules from embedded resources
func loadLuaAssetModules(modules []string) ([][]byte, error) {
	var result [][]byte
	// Loop through the embedded Lua files
	for _, module := range modules {
		content, err := Asset(module)
		if err != nil {
			return nil, err
		}
		result = append(result, content)
	}
	return result, nil
}

// Define a Go function that you want to expose to Lua
func add(a, b int) int {
	return a + b
}

// Given a filename, return the path to the file
func getPathToFile(filename string) string {
	return filepath.Dir(filename)
}

func generateOutputFilename(inputFilename, outputFormat string) string {
	// Extract the input Lua file name without extension
	fileName := strings.TrimSuffix(inputFilename, filepath.Ext(inputFilename))

	// Define the output YAML file name
	return fileName + "." + outputFormat
}

type initLuaStateInput struct {
	luaModules [][]byte
}

func initLuaState(input initLuaStateInput) (*lua.LState, error) {
	// Create a new Lua state
	luaState := lua.NewState()

	luaState.SetGlobal("testAdd", luaState.NewFunction(func(L *lua.LState) int {
		a := L.ToInt(1)
		b := L.ToInt(2)

		result := add(a, b)

		L.Push(lua.LNumber(result))
		return 1
	}))

	// Load custom Lua modules
	for _, module := range input.luaModules {
		if err := luaState.DoString(string(module)); err != nil {
			return nil, err
		}
	}

	return luaState, nil
}

type runInput struct {
	format   func(v interface{}) ([]byte, error)
	script   []byte
	luaState *lua.LState
	cwd      string
}

func run(input runInput) ([]byte, error) {
	// Set the current working directory
	// There are a bunch of checks here to make sure the path is valid
	// It's not really required in production because we know the script exists
	// However, during testing things can go wrong
	if input.cwd != "" {
		// check if path is valid and exists
		if _, err := os.Stat(input.cwd); os.IsNotExist(err) {
			return nil, fmt.Errorf("cwd path does not exist: %s", input.cwd)
		}
		input.luaState.DoString("package.path = package.path .. ';" + input.cwd + "/?.lua'")
	}

	// Run the user-provided Lua script
	if err := input.luaState.DoString(string(input.script)); err != nil {
		return nil, err
	}

	returnedValue := input.luaState.Get(-1)

	// Get the arguments from Lua

	dataTable, ok := returnedValue.(*lua.LTable)

	if !ok {
		return nil, fmt.Errorf("expected a table, got %T", returnedValue)
	}

	goMap := luaTableToMap(dataTable)

	data, err := input.format(goMap)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func exit(message error) {
	fmt.Println(message)
	syscall.Exit(1)
}

func handleError[T interface{}](data T, err error) (T, error) {
	if err == nil {
		return data, nil
	}

	exit(err)

	// this will never happen at this stage, but we need to return it
	return data, err
}

func main() {

	// Define command-line flags
	outputFormat := flag.String("output", "", "Output format")
	outputUserSuffix := flag.String("suffix", "", "User suffix for output file name")
	flag.Parse()

	// Check if output file is provided
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . -output <yaml|json> <efemel script glob>")
		return
	}

	var filenames []string

	for _, arg := range flag.Args() {
		// Use filepath.Glob to get a slice of filenames that match the glob pattern
		matchedFilenames, _ := handleError(filepath.Glob(arg))
		// append the filenames to the list
		filenames = append(filenames, matchedFilenames...)
	}

	if len(filenames) == 0 {
		fmt.Println("No files found")
		return
	}

	formatter, _ := handleError(getFormatter(*outputFormat, *outputUserSuffix))

	luaModuleNames, _ := handleError(findAllLuaAssetModules("lua/"))

	luaModules, _ := handleError(loadLuaAssetModules(luaModuleNames))

	luaState, _ := handleError(initLuaState(initLuaStateInput{
		luaModules: luaModules,
	}))

	defer luaState.Close()

	var wg sync.WaitGroup

	// loop through the filenames and process each one in a separate goroutine
	for _, filename := range filenames {

		fmt.Println("Processing", filename)
		wg.Add(1)

		go func(filename string) {
			defer wg.Done()

			fmt.Println("Processing goroutine", filename)

			outputFilename := generateOutputFilename(filename, formatter.suffix)
			// Read the Lua script
			userScript, _ := handleError(os.ReadFile(filename))
			luaThread, _ := luaState.NewThread()

			// // Run the Lua script
			data, _ := handleError(run(runInput{
				format:   formatter.Marshal,
				script:   userScript,
				luaState: luaThread,
				cwd:      getPathToFile(filename),
			}))

			// // Write the result to the output file
			err := os.WriteFile(outputFilename, data, 0644)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			fmt.Println("Output written to", outputFilename)

		}(filename)
	}

	// Wait for all goroutines to finish
	wg.Wait()

}

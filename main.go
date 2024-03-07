package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

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

func loadLuaAssetModule(module string) (string, error) {
	content, err := Asset(module)
	if err != nil {
		return "", err
	}

	return string(content), nil
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

type getLuaTableInput struct {
	luaState *lua.LState
	script   string
	cwd      string
}

func getLuaTable(input getLuaTableInput) (*lua.LTable, error) {
	// Set the current working directory
	// There are a bunch of checks here to make sure the path is valid
	// It's not really required in production because we know the script exists
	// However, during testing things can go wrong
	if input.cwd != "" {
		// check if path is valid and exists
		if _, err := os.Stat(input.cwd); os.IsNotExist(err) {
			return nil, fmt.Errorf("cwd path does not exist: %s", input.cwd)
		}

		// escape the path otherwise it will break lua requires
		cwd := strings.ReplaceAll(input.cwd, "\\", "\\\\")

		input.luaState.DoString("package.path = package.path .. ';" + cwd + "/?.lua'")
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

	return dataTable, nil
}

type runInput struct {
	format   func(v interface{}) ([]byte, error)
	script   []byte
	luaState *lua.LState
	cwd      string
}

func run(input runInput) ([]byte, error) {
	dataTable, err := getLuaTable(getLuaTableInput{
		luaState: input.luaState,
		script:   string(input.script),
		cwd:      input.cwd,
	})

	if err != nil {
		return nil, err
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

func handleError[T interface{}](data T, err error) T {
	if err != nil {
		exit(err)
	}

	return data
}

func getAllFilesFromGlobs(globs []string) ([]string, error) {
	var result []string
	for _, glob := range globs {
		files, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		result = append(result, files...)
	}
	return result, nil
}

func main() {

	// Define command-line flags
	outputFormat := flag.String("output", "", "Output format")
	outputFileExtension := flag.String("suffix", "", "Output file extension")
	varsFile := flag.String("varsFile", "", "File with vars to be used in the script")
	flag.Parse()

	// Check if output file is provided
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . -output <yaml|json> <efemel script glob>")
		return
	}

	filenames := handleError(getAllFilesFromGlobs(flag.Args()))

	formatter := handleError(getFormatter(*outputFormat, *outputFileExtension))

	luaModules := handleError(findAllLuaAssetModules("lua/"))

	// Initialize Lua state
	luaState := NewLuaStateBuilder(nil)

	// load all modules
	for _, module := range luaModules {
		loadedModule := handleError(loadLuaAssetModule(module))
		luaState.LoadCustomLuaModule(loadedModule)
	}

	// add testAdd function
	luaState.AddGlobalFunction("testAdd", func(L *lua.LState) int {
		a := L.ToInt(1)
		b := L.ToInt(2)

		result := add(a, b)

		L.Push(lua.LNumber(result))
		return 1
	})

	// set vars file
	luaState.SetGlobalTableFromFile("vars", *varsFile)

	start := time.Now()

	err := luaState.Build()

	if err != nil {
		exit(err)
	}

	fmt.Printf("Lua state initialized in %s\n", time.Since(start))

	defer luaState.Close()

	var wg sync.WaitGroup

	// loop through the filenames and process each one in a separate goroutine
	for _, filename := range filenames {
		wg.Add(1)

		go func(filename string) {
			defer wg.Done()

			start := time.Now()
			outputFilename := generateOutputFilename(filename, formatter.suffix)
			// Read the Lua script
			userScript := handleError(os.ReadFile(filename))
			
			luaState, err := luaState.Clone()

			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			// // Run the Lua script
			data := handleError(run(runInput{
				format:   formatter.Marshal,
				script:   userScript,
				luaState: luaState.state,
				cwd:      getPathToFile(filename),
			}))

			// // Write the result to the output file
			if err := os.WriteFile(outputFilename, data, 0644); err != nil {
				fmt.Println("Error:", err)
				return
			}

			elapsed := time.Since(start)

			fmt.Printf("File %s processed in %s\n", outputFilename, elapsed)

		}(filename)
	}

	// Wait for all goroutines to finish
	wg.Wait()

}

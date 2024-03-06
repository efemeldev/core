package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

type runInput struct {
	format     func(v interface{}) ([]byte, error)
	script     []byte
	luaModules [][]byte
	cwd        string
}

func run(input runInput) ([]byte, error) {
	// Create a new Lua state
	luaState := lua.NewState()
	defer luaState.Close()

	// Register the Go function as a global function in Lua
	luaState.SetGlobal("add", luaState.NewFunction(func(L *lua.LState) int {
		a := luaState.ToInt(1)
		b := luaState.ToInt(2)
		result := add(a, b)
		luaState.Push(lua.LNumber(result))
		return 1 // Number of return values
	}))

	// Load custom Lua modules
	for _, module := range input.luaModules {
		if err := luaState.DoString(string(module)); err != nil {
			return nil, err
		}
	}

	// Set the current working directory
	// There are a bunch of checks here to make sure the path is valid
	// It's not really required in production because we know the script exists
	// However, during testing things can go wrong
	if input.cwd != "" {
		// check if path is valid and exists
		if _, err := os.Stat(input.cwd); os.IsNotExist(err) {
			return nil, fmt.Errorf("cwd path does not exist: %s", input.cwd)
		}
		luaState.DoString("package.path = package.path .. ';" + input.cwd + "/?.lua'")
	}

	// Run the user-provided Lua script
	if err := luaState.DoString(string(input.script)); err != nil {
		return nil, err
	}

	returnedValue := luaState.Get(-1)

	// Get the arguments from Lua

	dataTable, ok := returnedValue.(*lua.LTable);

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
		fmt.Println("Usage: go run . -output <yaml|json> <efemel script>")
		return
	}

	// Get the path to the Lua script file from the command-line argument
	luaScriptFile := flag.Args()[0]

	formatter, _ := handleError(getFormatter(*outputFormat, *outputUserSuffix))

	outputFilename := generateOutputFilename(luaScriptFile, formatter.suffix)

	luaModuleNames, _ := handleError(findAllLuaAssetModules("lua/"))

	luaModules, _ := handleError(loadLuaAssetModules(luaModuleNames))

	// Check if source file exists
	if _, err := os.Stat(luaScriptFile); os.IsNotExist(err) {
		exit(fmt.Errorf("file does not exist: %s", luaScriptFile))
	}

	// Run user-provided Lua script along with the custom module
	userScript, _ := handleError(os.ReadFile(luaScriptFile))

	// Run the Lua script
	data, _ := handleError(run(runInput{
		format:     formatter.Marshal,
		script:     userScript,
		luaModules: luaModules,
		cwd:        getPathToFile(luaScriptFile),
	}))

	// Write the result to the output file
	err := os.WriteFile(outputFilename, data, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

}

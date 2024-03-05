package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

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
	} else {
		// If not, treat it as a map
		result := make(map[string]interface{})
		table.ForEach(func(key, value lua.LValue) {
			result[key.String()] = luaValueToInterface(value)
		})
		return result
	}
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

	// create a channel that will be used to write the result of the Lua script
	resultChannel := make(chan []byte, 1)
	errorChannel := make(chan error, 1)

	// Register the Go function as a global function in Lua
	luaState.SetGlobal("main", luaState.NewFunction(func(L *lua.LState) int {
		// Get the arguments from Lua
		dataTable := luaState.CheckTable(1)

		goMap := luaTableToMap(dataTable)

		// if I were to create custom output formats using lua
		// I would have to take this function and pass it to the lua script
		// I then would have to create custom modules in lua to handle the output
		// and this function would only be responsible for capturing the output into a channel
		data, err := input.format(goMap)

		if err != nil {
			errorChannel <- err
			return 0
		}

		// Write data to channel
		resultChannel <- data

		return 0
	}))

	// Load custom Lua modules
	for _, module := range input.luaModules {
		if err := luaState.DoString(string(module)); err != nil {
			return nil, err
		}
	}

	// Run the user-provided Lua script
	if err := luaState.DoString(string(input.script)); err != nil {
		return nil, err
	}

	// Wait for the result of the Lua script
	select {
	case data := <-resultChannel:
		return data, nil

	case err := <-errorChannel:
		return nil, err
	}
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

	formatter, err := getFormatter(*outputFormat, *outputUserSuffix)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	outputFilename := generateOutputFilename(luaScriptFile, formatter.suffix)

	// Failed to get formatter
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	luaModuleNames, err := findAllLuaAssetModules("lua/")

	luaModules, err := loadLuaAssetModules(luaModuleNames)

	// Run user-provided Lua script along with the custom module
	userScript, err := ioutil.ReadFile(luaScriptFile)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Run the Lua script
	data, err := run(runInput{
		format:     formatter.Marshal,
		script:     userScript,
		luaModules: luaModules,
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Write the result to the output file
	err = ioutil.WriteFile(outputFilename, data, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

}

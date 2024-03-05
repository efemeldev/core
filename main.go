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

// Load custom Lua modules from embedded resources
func loadLuaModules(L *lua.LState) error {
	// Loop through the embedded Lua files
	for _, name := range AssetNames() {
		if strings.HasPrefix(name, "lua/") && strings.HasSuffix(name, ".lua") {
			content, err := Asset(name)
			if err != nil {
				fmt.Println("Error reading asset:", err)
				continue
			}
			if err := L.DoString(string(content)); err != nil {
				fmt.Println("Error loading Lua module:", err)
				continue
			}
		}
	}
	return nil
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
	outputFormat string
	inputFileName string
	outputFilename string
	outputUserSuffix string
}

func run(input runInput) error {
	formatter, err := getFormatter(input.outputFormat, input.outputUserSuffix)
	
	// Failed to get formatter
	if err != nil {
		return err
	}

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

	// Register the Go function as a global function in Lua
	luaState.SetGlobal("main", luaState.NewFunction(func(L *lua.LState) int {
		// Get the arguments from Lua
		dataTable := luaState.CheckTable(1)
		
		goMap := luaTableToMap(dataTable)
		data, err := formatter.Marshal(goMap)

		if err != nil {
			luaState.Push(lua.LString(fmt.Sprintf("Error: %s", err)))
			return 1
		}


		// Write YAML data to file
		err = ioutil.WriteFile(input.outputFilename, []byte(data), 0644)
		if err != nil {
			luaState.Push(lua.LString(fmt.Sprintf("Error writing to file: %s", err)))
			return 1
		}

		return 0
	}))

	// Load custom Lua modules from the "lua" folder
	err = loadLuaModules(luaState)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	// Run user-provided Lua script along with the custom module
	userScript, err := ioutil.ReadFile(input.inputFileName)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	if err := luaState.DoString(string(userScript)); err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	return nil
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

	// Run the Lua script
	err = run(runInput{
		outputFormat: *outputFormat,
		inputFileName: luaScriptFile,
		outputFilename: outputFilename,
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}
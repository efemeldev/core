package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
	"gopkg.in/yaml.v2"
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

// Function to generate YAML or JSON data from a Lua table based on the output format
func generateData(data *lua.LTable, format string) ([]byte, error) {
	// Convert Lua table to Go map
	goMap := luaTableToMap(data)

	// Convert Go map to YAML or JSON based on the format
	if format == "yaml" {
		yamlData, err := yaml.Marshal(goMap)
		if err != nil {
			return nil, err
		}
		return yamlData, nil
	} else if format == "json" {
		jsonData, err := json.Marshal(goMap)
		if err != nil {
			return nil, err
		}
		return jsonData, nil
	} else {
		return nil, fmt.Errorf("unsupported output format: %s", format)
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


func main() {
	// Define command-line flags
	outputFormat := flag.String("output", "yaml", "Output format: yaml or json")
	flag.Parse()

	// Check if a command-line argument is provided
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . -output <yaml|json> <lua_script>")
		return
	}

	// Get the path to the Lua script file from the command-line argument
	luaScriptFile := flag.Args()[0]

	// Create a new Lua state
	L := lua.NewState()
	defer L.Close()

    // Register the Go function as a global function in Lua
    L.SetGlobal("add", L.NewFunction(func(L *lua.LState) int {
        a := L.ToInt(1)
        b := L.ToInt(2)
        result := add(a, b)
        L.Push(lua.LNumber(result))
        return 1 // Number of return values
    }))

	// Register the Go function as a global function in Lua
	L.SetGlobal("main", L.NewFunction(func(L *lua.LState) int {
		// Get the arguments from Lua
		dataTable := L.CheckTable(1)

		// Generate YAML file
		yamlData, err := generateData(dataTable, *outputFormat)
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("Error: %s", err)))
			return 1
		}

		// Extract the input Lua file name without extension
		fileName := strings.TrimSuffix(luaScriptFile, filepath.Ext(luaScriptFile))

		// Define the output YAML file name
		outputFileName := fileName + "." + *outputFormat

		// Write YAML data to file
		err = ioutil.WriteFile(outputFileName, []byte(yamlData), 0644)
		if err != nil {
			L.Push(lua.LString(fmt.Sprintf("Error writing to file: %s", err)))
			return 1
		}

		return 0
	}))

	// Load custom Lua modules from the "lua" folder
	err := loadLuaModules(L)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Run user-provided Lua script along with the custom module
	userScript, err := ioutil.ReadFile(luaScriptFile)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if err := L.DoString(string(userScript)); err != nil {
		fmt.Println("Error:", err)
		return
	}
}
package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

type LuaStateManager struct {
	state          *lua.LState
	addedPaths     map[string]bool
	addedPathMutex sync.Mutex
}

type NewLuaStateManagerInput struct {
	override string
}

func NewLuaStateManager(input NewLuaStateManagerInput) *LuaStateManager {

	state := lua.NewState()

	if input.override != "" {
		wrapperScript := `
		local original_require = require

		function mergeTables(t1, t2)
			for k, v in pairs(t2) do
				if type(v) == "table" and type(t1[k]) == "table" then
					t1[k] = mergeTables(t1[k], v)
				else
					t1[k] = v
				end
			end
			return t1
		end

		function require(moduleName)
			-- Check if the module name starts with './'
			if string.sub(moduleName, 1, 2) == "./" then
				error("Relative paths are not supported")
			end

			local overrideModuleName = moduleName .. "-` + input.override + `"
		
			if package.loaded[overrideModuleName] then
				return package.loaded[overrideModuleName]
			end
			
			local status, overrideModule = pcall(original_require, overrideModuleName)
			
			originalModule = original_require(moduleName)
		
			if not status then
				return originalModule
			end
		
			if type(originalModule) == "table" and type(overrideModule) == "table" then
				originalModule = mergeTables(originalModule, overrideModule)
				return originalModule
			end
			
			return overrideModule
		end
		`
		if err := state.DoString(wrapperScript); err != nil {
			fmt.Println("Error:", err)
			return nil
		}
	}

	return &LuaStateManager{state: state, addedPaths: make(map[string]bool), addedPathMutex: sync.Mutex{}}
}

func (l *LuaStateManager) AddGlobalFunction(name string, function func(L *lua.LState) int) {
	l.state.SetGlobal(name, l.state.NewFunction(function))
}

func (l *LuaStateManager) AddPath(path string) error {
	l.addedPathMutex.Lock()
	defer l.addedPathMutex.Unlock()
	// check if cwd is already in the map
	if _, exists := l.addedPaths[path]; exists {
		return nil // cwd is already in the map, do nothing
	}

	// check if path is valid and exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("cwd path does not exist: %s", path)
	}

	path = strings.ReplaceAll(path, "\\", "\\\\")

	err := l.state.DoString("package.path = package.path .. ';" + path + "\\?.lua'")

	if err != nil {
		return err
	}

	// if there was no error, add cwd to the map
	l.addedPaths[path] = true

	return nil // nil or error
}

func (l *LuaStateManager) SetGlobalTable(name string, table *lua.LTable) {
	l.state.SetGlobal(name, table)
}

func (l *LuaStateManager) Close() {
	l.state.Close()
}

// run script
func RunScriptRaw(lua *lua.LState, script string) (lua.LValue, error) {
	if err := lua.DoString(script); err != nil {
		return nil, err
	}

	returnedLuaValue := lua.Get(-1)

	return returnedLuaValue, nil
}

// run script
func RunScript[T any](state *lua.LState, script string, processValue func(state *lua.LState, value lua.LValue) (T, error)) (T, error) {

	returnedLuaValue, err := RunScriptRaw(state, script)

	if err != nil {
		return null[T](), err
	}

	processedValue, err := processValue(state, returnedLuaValue)

	fmt.Println("processedValue:", processedValue)

	if err != nil {
		return null[T](), err
	}

	return processedValue, nil
}

// get returned table from script
func GetReturnedLuaTable(value lua.LValue) (*lua.LTable, error) {
	// Get the arguments from Lua
	dataTable, ok := value.(*lua.LTable)

	if !ok {
		return nil, fmt.Errorf("expected a table, got %T", value)
	}

	return dataTable, nil
}

// execute lua function if it is a function
func RunReturnedLuaFunction(state *lua.LState, value lua.LValue) (lua.LValue, error) {
	// Call the function and use its return value
	err := state.CallByParam(lua.P{
		Fn:      value,
		NRet:    1,
		Protect: true,
	}, lua.LNil)

	if err != nil {
		return null[lua.LValue](), err
	}

	// Get the function's return value
	value = state.Get(-1)
	state.Pop(1)

	return value, nil
}

// get returned table from script
func GetReturnedMap(state *lua.LState, value lua.LValue) (interface{}, error) {

	fmt.Println("value type:", value.Type())

    // Check if the returned value is a function
    if value.Type() == lua.LTFunction {
        value, err := RunReturnedLuaFunction(state, value)

		if err != nil {
			return null[interface{}](), err
		}

		return GetReturnedMap(state, value)
    }

	// Get the arguments from Lua
	dataTable, err := GetReturnedLuaTable(value)

	if err != nil {
		return nil, err
	}

	return luaTableToMap(state, dataTable), nil
}

// get returned string from script
func GetReturnedString(value lua.LValue) (string, error) {
	// Get the arguments from Lua
	dataString, ok := value.(lua.LString)

	if !ok {
		return "", fmt.Errorf("expected a string, got %T", value)
	}

	return string(dataString), nil
}

// Function to recursively convert Lua table to Go map
func luaValueToInterface(state *lua.LState, value lua.LValue) interface{} {
	switch value.Type() {
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTNumber:
		return float64(value.(lua.LNumber))
	case lua.LTString:
		return string(value.(lua.LString))
	case lua.LTTable:
		return luaTableToMap(state, value.(*lua.LTable))
	case lua.LTFunction:
		newValue, error := RunReturnedLuaFunction(state, value)

		if error != nil {
			panic(error)
		}

		return newValue
	default:
		return nil
	}
}

// convert Lua table to Go interface
func luaTableToMap(state *lua.LState, table *lua.LTable) interface{} {
	if table.MaxN() > 0 {
		// If the table has sequential integer keys starting from 1, treat it as an array
		arr := make([]interface{}, table.MaxN())
		table.ForEach(func(i lua.LValue, value lua.LValue) {
			idx := int(i.(lua.LNumber))
			arr[idx-1] = luaValueToInterface(state, value)
		})
		return arr
	}

	// If not, treat it as a map
	result := make(map[string]interface{})

	table.ForEach(func(key, value lua.LValue) {
		result[key.String()] = luaValueToInterface(state, value)
	})

	return result
}

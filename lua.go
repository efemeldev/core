package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// run script
func RunScriptRaw(lua *lua.LState, script string) (lua.LValue, error) {
	if err := lua.DoString(script); err != nil {
		return nil, err
	}

	returnedLuaValue := lua.Get(-1)

	return returnedLuaValue, nil
}

// run script
func RunScript[T any](lua *lua.LState, script string, processValue func(value lua.LValue) (T, error)) (T, error) {

	returnedLuaValue, err := RunScriptRaw(lua, script)

	if err != nil {
		return null[T](), err
	}

	processedValue, err := processValue(returnedLuaValue)

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

// get returned table from script
func GetReturnedMap(value lua.LValue) (interface{}, error) {
	// Get the arguments from Lua
	dataTable, err := GetReturnedLuaTable(value)

	if err != nil {
		return nil, err
	}

	return luaTableToMap(dataTable), nil
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

type LuaStateManager struct {
	state          *lua.LState
	addedPaths     map[string]bool
	addedPathMutex sync.Mutex
}

func NewLuaStateManager() *LuaStateManager {
	return &LuaStateManager{state: lua.NewState(), addedPaths: make(map[string]bool), addedPathMutex: sync.Mutex{}}
}

func (l *LuaStateManager) AddGlobalFunction(name string, function func(L *lua.LState) int) {
	l.state.SetGlobal(name, l.state.NewFunction(function))
}

func (l *LuaStateManager) LoadCustomLuaModule(module string) error {
	err := l.state.DoString(module)
	return err // nil or error
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

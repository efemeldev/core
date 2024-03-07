package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type LuaStateBuilder struct {
	state         *lua.LState
	actions       []func(L *lua.LState) error
	canAddActions bool
}

// NewLuaBuilder creates a new LuaStateBuilder with optional state
func NewLuaStateBuilder(state *lua.LState) *LuaStateBuilder {
	return &LuaStateBuilder{
		state:         state,
		actions:       make([]func(L *lua.LState) error, 0),
		canAddActions: true,
	}
}

// add action to the builder
func (l *LuaStateBuilder) AddAction(action func(L *lua.LState) error) *LuaStateBuilder {
	if !l.canAddActions {
		panic("cannot add actions after building the state")
	}

	l.actions = append(l.actions, action)
	return l
}

func (l *LuaStateBuilder) AddGlobalFunction(name string, function func(L *lua.LState) int) *LuaStateBuilder {
	l.AddAction(func(L *lua.LState) error {
		L.SetGlobal(name, L.NewFunction(function))
		return nil
	})
	return l
}

// TODO: Add custom module loader
func (l *LuaStateBuilder) LoadCustomLuaModule(module string) *LuaStateBuilder {
	l.AddAction(func(L *lua.LState) error {
		if err := L.DoString(module); err != nil {
			return err
		}
		return nil
	})

	return l
}

// set current working directory so that require can find the modules
func (l *LuaStateBuilder) SetCWD(cwd string) *LuaStateBuilder {
	l.AddAction(func(L *lua.LState) error {
		// check if path is valid and exists
		if _, err := os.Stat(cwd); os.IsNotExist(err) {
			return fmt.Errorf("cwd path does not exist: %s", cwd)
		}

		cwd := strings.ReplaceAll(cwd, "\\", "\\\\")
		if err := L.DoString("package.path = package.path .. ';" + cwd + "/?.lua'"); err != nil {
			return err
		}
		return nil
	})

	return l
}

// process actions and return the state
func (l *LuaStateBuilder) Build() (error) {
	if l.state == nil {
		l.state = lua.NewState()
	}
	
	for _, action := range l.actions {
		if err := action(l.state); err != nil {
			return err
		}
	}

	// clear the actions
	l.actions = make([]func(L *lua.LState) error, 0)


	return nil
}

// clone the state, add actions and return the new state
func (l *LuaStateBuilder) Clone() (*LuaStateBuilder, error) {
	thread, _ := l.state.NewThread()
	return NewLuaStateBuilder(thread), nil
}

// set global table from script
func (l *LuaStateBuilder) SetGlobalTableFromFile(name string, file string) *LuaStateBuilder {

	// get folder of the file
	l.SetCWD(filepath.Dir(file))

	l.AddAction(func(L *lua.LState) error {

		script, err := os.ReadFile(file)

		if err != nil {
			return err
		}

		if err := L.DoString(string(script)); err != nil {
			return err
		}

		returnedValue := L.Get(-1)

		// Get the arguments from Lua
		dataTable, ok := returnedValue.(*lua.LTable)

		if !ok {
			return fmt.Errorf("expected a table, got %T", returnedValue)
		}

		L.SetGlobal(name, dataTable)
		return nil
	})

	return l
}

// close the state
func (l *LuaStateBuilder) Close() {
	if l.state != nil {
		l.state.Close()
	}
}

// run script
func RunScript[T any](lua *LuaStateBuilder, script string, getValue func(state *lua.LState) (T, error))  (T, error) {
	if err := lua.state.DoString(script); err != nil {
		return null[T](), err
	}

	returnedValue, err := getValue(lua.state)

	if err != nil {
		return null[T](), err
	}

	return returnedValue, nil
}

func RunFile[T any](lua *LuaStateBuilder, file string, getValue func(state *lua.LState) (T, error)) (T, error) {
	script, err := os.ReadFile(file)

	if err != nil {
		return null[T](), err
	}

	res, err := RunScript(lua, string(script), getValue)

	if err != nil {
		return null[T](), err
	}

	return res, nil
}


// get returned table from script
func GetReturnedTable(state *lua.LState) (interface{}, error) {
	returnedValue := state.Get(-1)

	// Get the arguments from Lua
	dataTable, ok := returnedValue.(*lua.LTable)

	if !ok {
		return nil, fmt.Errorf("expected a table, got %T", returnedValue)
	}

	return luaTableToMap(dataTable), nil
}

// get returned string from script
func GetReturnedString(state *lua.LState) (string, error) {
	returnedValue := state.Get(-1)

	// Get the arguments from Lua
	dataString, ok := returnedValue.(lua.LString)

	if !ok {
		return "", fmt.Errorf("expected a string, got %T", returnedValue)
	}

	return string(dataString), nil
}
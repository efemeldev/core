package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

var (
	cwdMap = make(map[string]bool)
	cwdMutex sync.Mutex
)

func AddGlobalFunction(state *lua.LState, name string, function func(L *lua.LState) int) {
	state.SetGlobal(name, state.NewFunction(function))
}

func LoadCustomLuaModule(state *lua.LState, module string) error {
	err := state.DoString(module)
	return err // nil or error
}

// set current working directory so that require can find the modules
func SetCWD(state *lua.LState, cwd string) error {
	cwdMutex.Lock()
	defer cwdMutex.Unlock()
    // check if cwd is already in the map
    if _, exists := cwdMap[cwd]; exists {
        return nil // cwd is already in the map, do nothing
    }

	// check if path is valid and exists
	if _, err := os.Stat(cwd); os.IsNotExist(err) {
		return fmt.Errorf("cwd path does not exist: %s", cwd)
	}

	cwd = strings.ReplaceAll(cwd, "\\", "\\\\")

	err := state.DoString("package.path = package.path .. ';" + cwd + "\\?.lua'")

    if err != nil {
		return err
    }

	// if there was no error, add cwd to the map
	cwdMap[cwd] = true

    return nil // nil or error
}

// set global table from script
func SetGlobalTableFromFile(state *lua.LState, name string, file string) error {

	// get folder of the file
	SetCWD(state, filepath.Dir(file))

	script, err := os.ReadFile(file)

	if err != nil {
		return err
	}

	if err := state.DoString(string(script)); err != nil {
		return err
	}

	returnedValue := state.Get(-1)

	// Get the arguments from Lua
	dataTable, ok := returnedValue.(*lua.LTable)

	if !ok {
		return fmt.Errorf("expected a table, got %T", returnedValue)
	}

	state.SetGlobal(name, dataTable)
	return nil
	
}

// run script
func RunScript[T any](lua *lua.LState, script string, getValue func(state *lua.LState) (T, error))  (T, error) {
	if err := lua.DoString(script); err != nil {
		return null[T](), err
	}

	returnedValue, err := getValue(lua)

	if err != nil {
		return null[T](), err
	}

	return returnedValue, nil
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

type LuaStateBuilder func(state *lua.LState) (*lua.LState, error)

// LuaStatePool represents a pool of Lua states
type LuaStatePool struct {
    pool *sync.Pool
	size int
}

func NewLuaStatePool(initialSize int, stateSetup func(L *lua.LState)(*lua.LState, error)) *LuaStatePool {
	pool := sync.Pool{}
	// Initialize the pool with initialSize Lua states
	for i := 0; i < initialSize; i++ {
		state, err := stateSetup(lua.NewState())

		if err != nil {
			panic(err)
		}

		pool.Put(state)

		fmt.Printf("Initialized Lua state %d\n", i+1)
	}
	return &LuaStatePool{pool: &pool, size: initialSize } // Fix: Use the sync.Pool value directly instead of copying it
}

// Get retrieves a Lua state from the pool
func (p *LuaStatePool) Get() *lua.LState {
	if state, ok := p.pool.Get().(*lua.LState); ok {
		return state
	}
	return lua.NewState() // If pool is empty, create a new Lua state
}

// Put returns a Lua state to the pool
func (p *LuaStatePool) Put(L *lua.LState) {
	L.SetTop(0) // Reset stack
	p.pool.Put(L)
}

// Close closes all Lua states in the pool
func (p *LuaStatePool) Close() {
	// Close all Lua states in the pool
	for i := 0; i < p.size; i++ {
		if state := p.Get(); state != nil {
			state.Close()
			fmt.Printf("Closed Lua state %d\n", i)
			i++
		} else {
			break
		}
	}
}
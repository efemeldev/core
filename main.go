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
			
			luaState, err := luaState.Clone()

			if err != nil {
				fmt.Println("Error:", err)
				return
			}


			if err := luaState.SetCWD(getPathToFile(filename)).Build(); err != nil {
				fmt.Println("Error:", err)
				return
			}

			res, err := RunFile(luaState, filename, GetReturnedTable)

			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			formattedData, err := formatter.Marshal(res)

			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			// // Write the result to the output file
			if err := os.WriteFile(outputFilename, formattedData, 0644); err != nil {
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

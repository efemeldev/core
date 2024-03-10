package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

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

type FileData struct {
	Filename string
	Data     []byte
}

func main() {

	// Define command-line flags
	outputFormat := flag.String("output", "", "Output format")
	outputFileExtension := flag.String("suffix", "", "Output file extension")
	dryRun := flag.Bool("dryrun", false, "Dry run")
	varsFile := flag.String("vars", "", "File with vars to be used in the script")
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

	luaStateManagerPool := NewLuaStateManagerPool(10, func(luaStateManager *LuaStateManager) (*LuaStateManager, error) {

		// load all modules
		for _, module := range luaModules {
			loadedModule := handleError(loadLuaAssetModule(module))

			if err := luaStateManager.LoadCustomLuaModule(loadedModule); err != nil {
				return nil, err
			}
		}

		luaStateManager.AddGlobalFunction("testAdd", func(L *lua.LState) int {
			a := L.ToInt(1)
			b := L.ToInt(2)

			result := add(a, b)

			L.Push(lua.LNumber(result))
			return 1
		})

		// load vars file as a global table
		if *varsFile != "" {

			luaStateManager.AddPath(getPathToFile(*varsFile))

			// load script

			script, err := os.ReadFile(*varsFile)

			if err != nil {
				return nil, err
			}

			value, err := RunScript(luaStateManager.state, string(script), GetReturnedLuaTable)

			if err != nil {
				return nil, err
			}

			luaStateManager.SetGlobalTable("vars", value)
		}

		return luaStateManager, nil
	})

	defer luaStateManagerPool.Close()

	fileDataChannel := make(chan FileData, len(filenames))

	pooledProcessor := Poolable(func(input FileData) {
		// start := time.Now()
		outputFilename := generateOutputFilename(input.Filename, formatter.suffix)

		luaStateManager := luaStateManagerPool.Get()

		luaStateManager.AddPath(getPathToFile(input.Filename))

		// get package.path from lua

		res, err := RunScript(luaStateManager.state, string(input.Data), GetReturnedMap)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		formattedData, err := formatter.Marshal(res)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// elapsed := time.Since(start)

		// fmt.Printf("[%s] processed in %s\n", outputFilename, elapsed)

		// Push formatted data and filename into the channel
		fileDataChannel <- FileData{Filename: outputFilename, Data: formattedData}

		luaStateManagerPool.Put(luaStateManager)
	})

	filenameDataChannel := make(chan FileData)

	// 10 concurrent goroutines to process the files
	go pooledProcessor.Run(5, filenameDataChannel)

	// loop through the filenames and process each one in a separate goroutine
	for _, filename := range filenames {

		script, err := os.ReadFile(filename)

		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		filenameDataChannel <- FileData{Filename: filename, Data: script}
	}

	close(filenameDataChannel)

	// create wait group to wait for all files to be processed
	wg := sync.WaitGroup{}
	wg.Add(1)

	if *dryRun {
		go func() {
			defer wg.Done()
			for fileData := range fileDataChannel {
				fmt.Println(string(fileData.Filename))
			}
		}()
	} else {
		go func() {
			defer wg.Done()
			for fileData := range fileDataChannel {
				if err := os.WriteFile(fileData.Filename, fileData.Data, 0644); err != nil {
					fmt.Println("Error:", err)
					return
				}
			}
		}()
	}

	pooledProcessor.Wait()

	close(fileDataChannel)

	wg.Wait()
}

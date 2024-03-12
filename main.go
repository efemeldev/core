package main

import (
	fileprocessors "efemel/services/fileprocessors"
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
	Filename       string
	OutputFilename string
	Data           []byte
}

func main() {

	// Define command-line flags
	outputFormat := flag.String("output", "", "Output format")
	outputFileExtension := flag.String("suffix", "", "Output file extension")
	dryRun := flag.Bool("dryrun", false, "Dry run")
	varsFile := flag.String("vars", "", "File with vars to be used in the script")
	workerCount := flag.Int("workers", 2, "Number of workers")
	writerCount := flag.Int("writers", 1, "Number of writers")
	inputChannelBufferSize := flag.Int("input-buffer", 10, "Input channel buffer size")
	outputChannelBufferSize := flag.Int("output-buffer", 10, "Output channel buffer size")
	flag.Parse()

	// Check if output file is provided
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . -output <yaml|json> <efemel script glob>")
		return
	}

	filenames := handleError(getAllFilesFromGlobs(flag.Args()))

	formatter := handleError(getFormatter(*outputFormat, *outputFileExtension))

	luaModules := handleError(findAllLuaAssetModules("lua/"))

	// Initialize Lua workers
	worker := func(id int, jobs <-chan FileData, results chan<- FileData, wg *sync.WaitGroup) {

		luaStateManager := NewLuaStateManager()

		for _, module := range luaModules {
			loadedModule := handleError(loadLuaAssetModule(module))

			if err := luaStateManager.LoadCustomLuaModule(loadedModule); err != nil {
				panic(err)
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
				panic(err)
			}

			value, err := RunScript(luaStateManager.state, string(script), GetReturnedLuaTable)

			if err != nil {
				panic(err)
			}

			luaStateManager.SetGlobalTable("vars", value)
		}

		defer wg.Done()

		for job := range jobs {
			luaStateManager.AddPath(getPathToFile(job.Filename))

			// get package.path from lua

			res, err := RunScript(luaStateManager.state, string(job.Data), GetReturnedMap)
			if err != nil {
				panic(err)
			}

			formattedData, err := formatter.Marshal(res)
			if err != nil {
				panic(err)
			}

			// elapsed := time.Since(start)

			// fmt.Printf("[%s] processed in %s\n", outputFilename, elapsed)

			// Push formatted data and filename into the channel
			results <- FileData{Filename: job.Filename, OutputFilename: job.OutputFilename, Data: formattedData}
		}

		fmt.Println("worker", id, "shutting down")
		luaStateManager.Close()
	}

	dataInputChannel := make(chan FileData, *inputChannelBufferSize)
	dataOutputChannel := make(chan FileData, *outputChannelBufferSize)

	// Create a pool of workers
	var wg sync.WaitGroup
	wg.Add(*workerCount)

	for i := 1; i <= *workerCount; i++ {
		go worker(i, dataInputChannel, dataOutputChannel, &wg)
	}

	fileProcessor := fileprocessors.NewLocalFileProcessor()


	// Producer goroutine to send jobs
	go func() {
		defer close(dataInputChannel)

		// loop through the filenames and process each one in a separate goroutine
		for _, filename := range filenames {

			fmt.Printf("Processing %s\n", filename)

			script, err := fileProcessor.ReadFile(filename)

			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			outputFileName := generateOutputFilename(filename, formatter.suffix)

			dataInputChannel <- FileData{Filename: filename, OutputFilename: outputFileName, Data: script}
		}
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(dataOutputChannel) // Close the results channel after all workers finish
	}()

	(func(){
		if *dryRun {
			for fileData := range dataOutputChannel {
				fmt.Println(string(fileData.Filename))
			}
			return
		}

		// Write files
		writeWaitGroup := sync.WaitGroup{}

		writeWaitGroup.Add(*writerCount)

		for i := 0; i < *writerCount; i++ {

			go func() {
				defer writeWaitGroup.Done()

				for fileData := range dataOutputChannel {
					if err := fileProcessor.WriteFile(fileData.OutputFilename, fileData.Data); err != nil {
						panic(err)
					}
				}
			}()
		}

		writeWaitGroup.Wait()

	})()

	fmt.Println("All jobs are done")
}

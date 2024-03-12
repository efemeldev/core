package main

import (
	fileprocessors "efemel/services/fileprocessors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// Define a Go function that you want to expose to Lua
func add(a, b int) int {
	return a + b
}

func generateOutputFilename(path, filename, suffix string) string {
	// Extract the input Lua file name without extension
	newFileName := strings.TrimSuffix(filename, filepath.Ext(filename)) + "." + suffix

	// Merge path and filename
	fullFilename := filepath.Join(path, newFileName)

	// Define the output YAML file name
	return fullFilename
}

type FileData struct {
	Filename       string
	FilePath       string
	OutputFilename string
	Data           []byte
}

type OutputFileData struct {
	Filename       string
	FilePath       string
	OutputFilename string
	Data           interface{}
}

type InitialiseLuaStateInput struct {
	luaModules []string
	varsScript string
	varsPath   string
}

func initialiseLuaState(input InitialiseLuaStateInput) *LuaStateManager {
	luaStateManager := NewLuaStateManager()

	for _, module := range input.luaModules {
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
	if input.varsScript != "" {

		luaStateManager.AddPath(input.varsPath)

		value, err := RunScript(luaStateManager.state, input.varsScript, GetReturnedLuaTable)

		if err != nil {
			panic(err)
		}

		luaStateManager.SetGlobalTable("vars", value)
	}

	return luaStateManager
}

func main() {

	// Define command-line flags
	outputFormat := flag.String("format", "", "Output format")
	outputFileExtension := flag.String("suffix", "", "Output file extension")
	dryRun := flag.Bool("dryrun", false, "Dry run")
	varsFile := flag.String("vars", "", "File with vars to be used in the script")
	workerCount := flag.Int("workers", 2, "Number of workers")
	writerCount := flag.Int("writers", 1, "Number of writers")
	inputChannelBufferSize := flag.Int("input-buffer", 10, "Input channel buffer size")
	outputChannelBufferSize := flag.Int("output-buffer", 10, "Output channel buffer size")
	outputFilePath := flag.String("output-path", "./", "Output path")
	flag.Parse()

	// Check if output file is provided
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . -output <yaml|json> <efemel script glob>")
		return
	}

	fileProcessor := fileprocessors.NewLocalFileProcessor()

	filenames := handleError(fileProcessor.FindFiles(flag.Args()))

	formatter := handleError(getFormatter(*outputFormat, *outputFileExtension))

	luaModules := handleError(findAllLuaAssetModules("lua/"))

	varsScript := handleError(fileProcessor.ReadFile(*varsFile))

	initLuaStateInput := InitialiseLuaStateInput{
		luaModules: luaModules,
		varsScript: string(varsScript),
		varsPath:   fileProcessor.GetPathToFile(*varsFile),
	}

	// Initialize Lua workers
	worker := func(id int, jobs <-chan FileData, results chan<- OutputFileData, wg *sync.WaitGroup) {

		luaStateManager := initialiseLuaState(initLuaStateInput)

		defer wg.Done()

		for job := range jobs {
			// to handle relative imports to the file
			luaStateManager.AddPath(job.FilePath)

			res, err := RunScript(luaStateManager.state, string(job.Data), GetReturnedMap)

			if err != nil {
				panic(err)
			}

			results <- OutputFileData{
				Filename:       job.Filename,
				FilePath:       job.FilePath,
				OutputFilename: job.OutputFilename,
				Data:           res,
			}
		}

		fmt.Println("worker", id, "shutting down")
		luaStateManager.Close()
	}

	dataInputChannel := make(chan FileData, *inputChannelBufferSize)
	dataOutputChannel := make(chan OutputFileData, *outputChannelBufferSize)

	// Create a pool of workers
	var wg sync.WaitGroup
	wg.Add(*workerCount)

	for i := 1; i <= *workerCount; i++ {
		go worker(i, dataInputChannel, dataOutputChannel, &wg)
	}

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

			outputFileName := generateOutputFilename(*outputFilePath, filename, formatter.suffix)

			dataInputChannel <- FileData{
				Filename:       filename,
				FilePath:       fileProcessor.GetPathToFile(filename),
				OutputFilename: outputFileName,
				Data:           script,
			}
		}
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(dataOutputChannel) // Close the results channel after all workers finish
	}()

	(func() {
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
					formattedData, err := formatter.Marshal(fileData.Data)

					if err != nil {
						panic(err)
					}

					fmt.Println("Writing", fileData.OutputFilename)

					if err := fileProcessor.WriteFile(fileData.OutputFilename, formattedData); err != nil {
						panic(err)
					}
				}
			}()
		}

		writeWaitGroup.Wait()

	})()

	fmt.Println("All jobs are done")
}

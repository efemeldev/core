package main

import (
	fileprocessors "efemel/services/fileprocessors"
	"flag"
	"fmt"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

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

// Define a Go function that you want to expose to Lua
func add(a, b int) int {
	return a + b
}

func luaAdd(L *lua.LState) int {
	a := L.ToInt(1)
	b := L.ToInt(2)

	result := add(a, b)

	L.Push(lua.LNumber(result))
	return 1
}

type RunInput struct {
	fileProcessor fileprocessors.FileProcessor
	formatter     Formatter
	luaStateManagerBuilder func() *LuaStateManager
	filenames	 []string
	inputChannelBufferSize int
	outputChannelBufferSize int
	workerCount int
	writerCount int
	outputFilePath string
	dryRun bool
}

func run(input RunInput) {
	// Initialize Lua workers
	worker := func(id int, jobs <-chan FileData, results chan<- OutputFileData, wg *sync.WaitGroup) {
		// workers can't share the same Lua state, so we need to create a new one for each worker
		luaStateManager := input.luaStateManagerBuilder()

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

	dataInputChannel := make(chan FileData, input.inputChannelBufferSize)
	dataOutputChannel := make(chan OutputFileData, input.outputChannelBufferSize)

	// Create a pool of workers
	var wg sync.WaitGroup
	wg.Add(input.workerCount)

	for i := 1; i <= input.workerCount; i++ {
		go worker(i, dataInputChannel, dataOutputChannel, &wg)
	}

	// Producer goroutine to send jobs
	go func() {
		defer close(dataInputChannel)

		// loop through the filenames and process each one in a separate goroutine
		for _, filename := range input.filenames {

			fmt.Printf("Processing %s\n", filename)

			script, err := input.fileProcessor.ReadFile(filename)

			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			outputFileName := generateOutputFilename(input.outputFilePath, filename, input.formatter.suffix)

			dataInputChannel <- FileData{
				Filename:       filename,
				FilePath:       input.fileProcessor.GetPathToFile(filename),
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
	

	if input.dryRun {
		for fileData := range dataOutputChannel {
			fmt.Println(string(fileData.Filename))
		}
		return
	}

	// Write files
	writeWaitGroup := sync.WaitGroup{}
	writeWaitGroup.Add(input.writerCount)

	for i := 0; i < input.writerCount; i++ {
		go func() {
			defer writeWaitGroup.Done()
			for fileData := range dataOutputChannel {
				formattedData, err := input.formatter.Marshal(fileData.Data)

				if err != nil {
					panic(err)
				}

				fmt.Println("Writing", fileData.OutputFilename)

				if err :=  input.fileProcessor.WriteFile(fileData.OutputFilename, formattedData); err != nil {
					panic(err)
				}
			}
		}()
	}

	writeWaitGroup.Wait()
}

func loadGlobalVars(fileProcessor fileprocessors.FileProcessor, varsFile string) (lua.LTable, error) {

	if varsFile == "" {
		return null[lua.LTable](), nil
	}

	varsScript := string(handleError(fileProcessor.ReadFile(varsFile)))
	varsPath := fileProcessor.GetPathToFile(varsFile)

	luaStateManager := NewLuaStateManager()
	defer luaStateManager.Close()

	luaStateManager.AddPath(varsPath)

	value, err := RunScript(luaStateManager.state, varsScript, GetReturnedLuaTable)

	if err != nil {
		return null[lua.LTable](), err
	}

	return *value, nil
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
	outputFilePath := flag.String("output-path", "./build", "Output path")
	flag.Parse()

	// Check if output file is provided
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . -output <yaml|json> <efemel script glob>")
		return
	}

	fileProcessor := fileprocessors.NewLocalFileProcessor()
	filenames := handleError(fileProcessor.FindFiles(flag.Args()))
	formatter := handleError(getFormatter(*outputFormat, *outputFileExtension))


	globalVars, err := loadGlobalVars(fileProcessor, *varsFile)

	if err != nil {
		panic(err) 
	}

	luaStateManagerBuilder := func () *LuaStateManager {
		luaStateManager := NewLuaStateManager()

		luaStateManager.AddGlobalFunction("testAdd", luaAdd)

		if varsFile != nil {
			// validate that Lua state doesn't have a global variable with the same name
			// custom modules could accidentally overwrite the global variable
			existingGlobalVars := luaStateManager.state.GetGlobal("vars")

			if existingGlobalVars != lua.LNil {
				panic("Global variable 'vars' already exists in the Lua state")
			}

			luaStateManager.SetGlobalTable("vars", &globalVars)
		}
		return luaStateManager
	}

	run(RunInput{
		fileProcessor: fileProcessor,
		formatter:     *formatter,
		luaStateManagerBuilder: luaStateManagerBuilder,
		filenames:     filenames,
		inputChannelBufferSize: *inputChannelBufferSize,
		outputChannelBufferSize: *outputChannelBufferSize,
		workerCount: *workerCount,
		writerCount: *writerCount,
		outputFilePath: *outputFilePath,
		dryRun: *dryRun,
	})

	fmt.Println("All jobs are done")
}

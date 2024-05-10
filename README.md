# efemel

Functional markup language for writing configuration files.

File extensions: `efemel` and `fmel`

## Running

Generate a YAML file named `script.yaml` from the `samples/script.lua` file.

```bash
go run . -format yaml samples/script.lua
```

Generate a JSON file named `script.json` from the `samples/script.lua` file.

```bash
go run . -format json samples/script.lua
```

You can also specify a glob of input files.

```bash
go run . -format yaml samples/*.lua
```

It will generate a YAML file for each input file in the output folder.

## Building

To build the project, run the following command:

```bash
go build -ldflags="-s -w" -o efemel
```

## Todo

- [x] Get rid of the need to wrap the configuration in a function; only do a return {}
- [x] Refactor `main.go`
- [x] Set CWD per input script so that lua modules can be referenced from the same folder
- [x] Error handler
- [x] Add support for multiple file input using glob
- [x] Multi core configuration generation
- [x] Add variable input (like tfvars) and implement variable usage in code
- [x] Use a builder pattern to add formatters, custom modules, lua functions, etc
- [x] Implement dry run
- [x] Concurrency controls
- [x] File reader/writer package that can write to filesystem or keep everything in memory (for testing)
- [x] Specify output file path
- [x] Deprecate built in lua modules
- [x] Initialise global variables outside of the state manager
- [x] Refactor the var loading code by overwriting the require function for better DX
- [x] Implement stacking of required lua files and merging of the result
- [x] Implement returned function execution before formatting
- [x] Don't allow relative imports
- [ ] Convert all lua file processing to use coroutines
- [ ] Implement file watching
- [ ] Implement a pipeline module that can be used to chain functions
- [ ] Convert writers to a module that can be extended so we're not limited with just writing to files
- [ ] Implement hooks for output file generation (using coroutines)
- [ ] Load config from a file (should support the same lua config files)
- [ ] Organise project structure
- [ ] Add better logging
- [ ] Add tests
- [ ] Figure out signing mac binaries
- [ ] Implement statistics capture
- [ ] Add CI/CD
- [ ] Add documentation
- [ ] Add examples
- [ ] Convert yml/json to lua ?
- [ ] ProductHunt

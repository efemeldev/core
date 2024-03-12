# efemel

Functional markup language for writing configuration files.

File extensions: `efemel` and `fmel`

## Running

Generate a YAML file named `script.yaml` from the `samples/script.lua` file.

```bash
go run . ./samples/script.lua
```

Generate a JSON file named `script.json` from the `samples/script.lua` file.

```bash
go run . -output json ./samples/script.lua
```

## Adding assets

To add assets to the project, run the following command:

```bash
go-bindata -o assets.go -pkg main ./lua
```

Only then you will have access to the `lua` folder and its contents.

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
- [ ] File reader/writer package that can write to filesystem or keep everything in memory (for testing)
- [ ] Module loader package that either loads local files or assets
- [ ] Add better logging
- [ ] Standardise output module injection
- [ ] Organise project structure
- [ ] Add tests
- [ ] Figure out signing mac binaries
- [ ] Implement statistics capture
- [ ] Implement post processing function that can be used to modify the output before it is written to a file
- [ ] Implement concat strings function for Lua
- [ ] Implement merge tables function for Lua
- [ ] Add CI/CD
- [ ] Add documentation
- [ ] Add examples

## Multiple input files

When I get to supporting multiple input files, I will need to:

1. Reuse compiled lua code to increase performance https://github.com/yuin/gopher-lua?tab=readme-ov-file#sharing-lua-byte-code-between-lstates
2. Figure out how to compile files on multiple cores

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
- [ ] Use a builder pattern to add formatters, custom modules, lua functions, etc
- [ ] Add variable input (like tfvars) and implement variable usage in code
- [ ] Standardise output module injection
- [ ] Organise project structure
- [ ] Implement custom modules folder that allows users to override any existing ones
- [ ] Figure out signing mac binaries
- [ ] Add tests
- [ ] Add CI/CD
- [ ] Add documentation
- [ ] Add examples

## Multiple input files

When I get to supporting multiple input files, I will need to:

1. Reuse compiled lua code to increase performance https://github.com/yuin/gopher-lua?tab=readme-ov-file#sharing-lua-byte-code-between-lstates
2. Figure out how to compile files on multiple cores

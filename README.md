# efemel

Functional markup language for writing configuration files.

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

- [ ] Refactor `main.go` into multiple files
- [ ] Standardise output module injection
- [ ] Organise project structure

- [ ] Add variable input (like tfvars) and implement variable usage in code
- [ ] Add glob variable input and output generation per var file
- [ ] Add support for multiple file input using glob
- [ ] Convert main function into one that runs an array of functions that generate the final output. Note that these functions should have access to all previous outputs
- [ ] Implement custom modules folder that allows users to override any existing ones
- [ ] Figure out signing mac binaries
- [ ] Add tests
- [ ] Add CI/CD
- [ ] Add documentation
- [ ] Add examples

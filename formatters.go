package main

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"
)

type Formatter struct {
	Marshal func(v interface{}) ([]byte, error)
	suffix  string
}

func getSuffix(suffix string, defaultSuffix string) string {
	if suffix != "" {
		return suffix
	}
	return defaultSuffix
}

// Function that takes in a parameter called format and it looks in the struct
// and either returns the formatter function or throws an error
func getFormatter(format string, userSuffix string) (*Formatter, error) {

	if format == "" {
		return nil, fmt.Errorf("output format not provided")
	}

	switch format {
	case "json":
		return &Formatter{Marshal: json.Marshal, suffix: getSuffix("json", userSuffix)}, nil
	case "yaml":
		return &Formatter{Marshal: yaml.Marshal, suffix: getSuffix("yaml", userSuffix)}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

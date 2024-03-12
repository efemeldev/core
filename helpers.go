package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
)

func null[T any]() T {
	var zero T
	return zero
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

func generateOutputFilename(path, filename, suffix string) string {
	// Extract the input Lua file name without extension
	newFileName := strings.TrimSuffix(filename, filepath.Ext(filename)) + "." + suffix

	// Merge path and filename
	fullFilename := filepath.Join(path, newFileName)

	// Define the output YAML file name
	return fullFilename
}
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

func extractOutputFilePath(path, filename string) string {
    // Extract the directory part of the filename
    subPath := filepath.Dir(filename)
    // Join the provided path and the subPath
    return filepath.Join(path, subPath)
}

func extractFilename(filename string) string {
	// Extract the filename from the provided path
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}

func generateOutputFilename(path, filename, suffix string) string {
	// Extract the input Lua file name without extension
	newFileName := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Merge path and filename
	fullFilename := filepath.Join(path, newFileName)+ "." + suffix

	// Define the output YAML file name
	return fullFilename
}

package fileprocessors

import (
	"os"
	"path/filepath"
)

// LocalFileProcessor implements the FileProcessor interface for local file processing.
type LocalFileProcessor struct {
    // Any specific configuration or dependencies can be added here.
}

// creates a new LocalFileProcessor
func NewLocalFileProcessor() *LocalFileProcessor {
	return &LocalFileProcessor{}
}

// ReadFile reads a file from local storage.
func (l *LocalFileProcessor) ReadFile(filePath string) ([]byte, error) {
    // Read the file from local storage
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    return data, nil
}

// Find all files matching a glob pattern
func (l *LocalFileProcessor) FindFiles(globs []string) ([]string, error) {
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

// WriteFile writes data to a file in local storage.
func (l *LocalFileProcessor) WriteFile(filePath string, data []byte) error {
    // Write the data to a file in local storage
    err := os.WriteFile(filePath, data, 0644)
    if err != nil {
        return err
    }
    return nil
}
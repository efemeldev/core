package fileprocessors

import (
	"fmt"
	"path/filepath"
)

// MemoryFileProcessor implements the FileProcessor interface for file processing in memory.
type MemoryFileProcessor struct {
    // Any specific configuration or dependencies can be added here.
    data map[string][]byte // For simplicity, using a map to store file data in memory
}

// ReadFile reads a file from memory.
func (m *MemoryFileProcessor) ReadFile(filePath string) ([]byte, error) {
    // Check if file exists in memory
    if data, ok := m.data[filePath]; ok {
        return data, nil
    }
    return nil, fmt.Errorf("file not found in memory: %s", filePath)
}

// FindFiles finds all files matching a glob pattern in memory.
func (m *MemoryFileProcessor) FindFiles(globs []string) ([]string, error) {
    var result []string
    for _, glob := range globs {
        for filePath := range m.data {
            matched, err := filepath.Match(glob, filePath)

            if err != nil {
                return nil, err
            }

            if matched {
                result = append(result, filePath)
            }
        }
    }
    return result, nil
}

// WriteFile writes data to memory.
func (m *MemoryFileProcessor) WriteFile(filePath string, data []byte) error {
    // Write the data to memory
    m.data[filePath] = data
    return nil
}
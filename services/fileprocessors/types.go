package fileprocessors

type FileProcessor interface {
	ReadFile(filePath string) ([]byte, error)
	WriteFile(filePath string, data []byte) error
	FindFiles(globs []string) ([]string, error)
	GetPathToFile(filename string) string
}
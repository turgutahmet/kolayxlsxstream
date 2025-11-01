package kolayxlsxstream

import (
	"os"
)

// FileSink writes data to a local file
type FileSink struct {
	file *os.File
	path string
}

// NewFileSink creates a new FileSink that writes to the specified file path
func NewFileSink(path string) (*FileSink, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return &FileSink{
		file: file,
		path: path,
	}, nil
}

// Write implements io.Writer interface
func (fs *FileSink) Write(p []byte) (n int, err error) {
	return fs.file.Write(p)
}

// Close implements io.Closer interface
func (fs *FileSink) Close() error {
	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

// Path returns the file path
func (fs *FileSink) Path() string {
	return fs.path
}

package kolayxlsxstream

import (
	"compress/flate"
	"io"
)

// newFlateWriter creates a new flate writer with the specified compression level
func newFlateWriter(w io.Writer, level int) (io.WriteCloser, error) {
	return flate.NewWriter(w, level)
}

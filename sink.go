package kolayxlsxstream

import (
	"io"
)

// Sink is the interface that wraps basic Write and Close methods for streaming data.
// Implementations can write to local files, S3, or any other destination.
type Sink interface {
	io.Writer
	io.Closer
}

// Stats contains statistics about the written XLSX file
type Stats struct {
	TotalRows      int64   // Total number of data rows written (excluding headers)
	TotalSheets    int     // Total number of sheets created
	FileSize       int64   // Total file size in bytes
	Duration       float64 // Total duration in seconds
	RowsPerSecond  float64 // Average rows per second
	BytesPerSecond float64 // Average bytes per second
}

// Config holds configuration for the XLSX writer
type Config struct {
	// CompressionLevel sets the ZIP compression level (0-9, default: 6)
	// 0 = no compression, 9 = maximum compression
	CompressionLevel int

	// BufferSize sets the buffer size in bytes (default: 64KB)
	BufferSize int

	// MaxRowsPerSheet sets the maximum rows per sheet (default: 1048576)
	// When this limit is reached, a new sheet is automatically created
	MaxRowsPerSheet int

	// SheetNamePrefix is the prefix for auto-generated sheet names (default: "Sheet")
	SheetNamePrefix string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		CompressionLevel: 6,
		BufferSize:       64 * 1024, // 64KB
		MaxRowsPerSheet:  1048576,   // Excel's maximum rows per sheet
		SheetNamePrefix:  "Sheet",
	}
}

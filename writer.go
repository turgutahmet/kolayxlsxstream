package kolayxlsxstream

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"time"
)

// Writer is the main XLSX streaming writer
type Writer struct {
	sink   Sink
	config *Config

	zipWriter *zip.Writer
	started   bool
	finished  bool

	currentSheet      int
	currentSheetRows  int
	currentSheetIndex int
	sheetWriters      []*sheetWriter
	totalRows         int64
	startTime         time.Time
	bytesWritten      int64
}

// sheetWriter handles writing to a single sheet
type sheetWriter struct {
	writer      io.Writer
	rowCount    int
	sheetIndex  int
	headersDone bool
	closed      bool
}

// NewWriter creates a new XLSX writer with the given sink and optional config
func NewWriter(sink Sink, config ...*Config) *Writer {
	cfg := DefaultConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return &Writer{
		sink:         sink,
		config:       cfg,
		sheetWriters: make([]*sheetWriter, 0),
	}
}

// StartFile initializes the XLSX file and optionally writes headers
func (w *Writer) StartFile(headers ...[]interface{}) error {
	if w.started {
		return fmt.Errorf("file already started")
	}

	w.started = true
	w.startTime = time.Now()
	w.zipWriter = zip.NewWriter(w.sink)

	// Set compression level
	w.zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return newFlateWriter(out, w.config.CompressionLevel)
	})

	// Write _rels/.rels
	if err := w.writeZipFile("_rels/.rels", []byte(relsXML)); err != nil {
		return fmt.Errorf("failed to write _rels/.rels: %w", err)
	}

	// Write xl/styles.xml
	if err := w.writeZipFile("xl/styles.xml", []byte(stylesXML)); err != nil {
		return fmt.Errorf("failed to write styles.xml: %w", err)
	}

	// Start the first sheet
	if err := w.startNewSheet(); err != nil {
		return err
	}

	// Write headers if provided
	if len(headers) > 0 && len(headers[0]) > 0 {
		if err := w.WriteRow(headers[0]); err != nil {
			return fmt.Errorf("failed to write headers: %w", err)
		}
		w.sheetWriters[0].headersDone = true
		w.totalRows-- // Don't count header row in statistics
	}

	return nil
}

// WriteRow writes a single row to the current sheet
func (w *Writer) WriteRow(values []interface{}) error {
	if !w.started {
		return fmt.Errorf("file not started, call StartFile first")
	}
	if w.finished {
		return fmt.Errorf("file already finished")
	}

	// Check if we need to start a new sheet
	currentWriter := w.sheetWriters[w.currentSheetIndex]
	if currentWriter.rowCount >= w.config.MaxRowsPerSheet {
		if err := w.startNewSheet(); err != nil {
			return err
		}
		currentWriter = w.sheetWriters[w.currentSheetIndex]
	}

	// Generate and write the row XML
	rowXML := generateRow(currentWriter.rowCount, values)
	if _, err := currentWriter.writer.Write([]byte(rowXML + "\n")); err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	currentWriter.rowCount++
	w.totalRows++

	return nil
}

// WriteRows writes multiple rows to the current sheet
func (w *Writer) WriteRows(rows [][]interface{}) error {
	for _, row := range rows {
		if err := w.WriteRow(row); err != nil {
			return err
		}
	}
	return nil
}

// FinishFile finalizes the XLSX file and returns statistics
func (w *Writer) FinishFile() (*Stats, error) {
	if !w.started {
		return nil, fmt.Errorf("file not started")
	}
	if w.finished {
		return nil, fmt.Errorf("file already finished")
	}

	w.finished = true

	// Close the last sheet (others were closed when new sheets were created)
	if len(w.sheetWriters) > 0 {
		lastSheet := w.sheetWriters[len(w.sheetWriters)-1]
		if !lastSheet.closed {
			if _, err := lastSheet.writer.Write([]byte(worksheetFooter)); err != nil {
				return nil, fmt.Errorf("failed to write worksheet footer: %w", err)
			}
			lastSheet.closed = true
		}
	}

	// Write xl/workbook.xml
	workbookXML := generateWorkbookXML(len(w.sheetWriters), w.config.SheetNamePrefix)
	if err := w.writeZipFile("xl/workbook.xml", []byte(workbookXML)); err != nil {
		return nil, fmt.Errorf("failed to write workbook.xml: %w", err)
	}

	// Write xl/_rels/workbook.xml.rels
	workbookRelsXML := generateWorkbookRelsXML(len(w.sheetWriters))
	if err := w.writeZipFile("xl/_rels/workbook.xml.rels", []byte(workbookRelsXML)); err != nil {
		return nil, fmt.Errorf("failed to write workbook.xml.rels: %w", err)
	}

	// Write [Content_Types].xml
	contentTypesXML := generateContentTypesXML(len(w.sheetWriters))
	if err := w.writeZipFile("[Content_Types].xml", []byte(contentTypesXML)); err != nil {
		return nil, fmt.Errorf("failed to write [Content_Types].xml: %w", err)
	}

	// Close the ZIP writer
	if err := w.zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	// Close the sink
	if err := w.sink.Close(); err != nil {
		return nil, fmt.Errorf("failed to close sink: %w", err)
	}

	// Calculate statistics
	duration := time.Since(w.startTime).Seconds()
	stats := &Stats{
		TotalRows:   w.totalRows,
		TotalSheets: len(w.sheetWriters),
		Duration:    duration,
	}

	if duration > 0 {
		stats.RowsPerSecond = float64(w.totalRows) / duration
	}

	return stats, nil
}

// SetCompressionLevel sets the ZIP compression level (0-9)
func (w *Writer) SetCompressionLevel(level int) error {
	if w.started {
		return fmt.Errorf("cannot change compression level after file started")
	}
	if level < 0 || level > 9 {
		return fmt.Errorf("compression level must be between 0 and 9")
	}
	w.config.CompressionLevel = level
	return nil
}

// SetBufferSize sets the buffer size
func (w *Writer) SetBufferSize(size int) error {
	if w.started {
		return fmt.Errorf("cannot change buffer size after file started")
	}
	if size < 1024 {
		return fmt.Errorf("buffer size must be at least 1024 bytes")
	}
	w.config.BufferSize = size
	return nil
}

// SetMaxRowsPerSheet sets the maximum rows per sheet
func (w *Writer) SetMaxRowsPerSheet(rows int) error {
	if w.started {
		return fmt.Errorf("cannot change max rows per sheet after file started")
	}
	if rows < 1 || rows > 1048576 {
		return fmt.Errorf("max rows per sheet must be between 1 and 1048576")
	}
	w.config.MaxRowsPerSheet = rows
	return nil
}

// startNewSheet creates a new worksheet in the ZIP
func (w *Writer) startNewSheet() error {
	// If there's a previous sheet, close it by writing footer
	if len(w.sheetWriters) > 0 {
		prevSheet := w.sheetWriters[len(w.sheetWriters)-1]
		if !prevSheet.closed {
			if _, err := prevSheet.writer.Write([]byte(worksheetFooter)); err != nil {
				return fmt.Errorf("failed to close previous sheet: %w", err)
			}
			prevSheet.closed = true
		}
	}

	sheetNum := len(w.sheetWriters) + 1
	sheetName := fmt.Sprintf("xl/worksheets/sheet%d.xml", sheetNum)

	// Create the sheet file in the ZIP
	writer, err := w.zipWriter.Create(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %d: %w", sheetNum, err)
	}

	// Write worksheet header
	if _, err := writer.Write([]byte(worksheetHeader + "\n")); err != nil {
		return fmt.Errorf("failed to write worksheet header: %w", err)
	}

	// Create sheet writer
	sw := &sheetWriter{
		writer:     writer,
		rowCount:   0,
		sheetIndex: sheetNum - 1,
	}

	w.sheetWriters = append(w.sheetWriters, sw)
	w.currentSheetIndex = len(w.sheetWriters) - 1

	return nil
}

// writeZipFile writes a complete file to the ZIP archive
func (w *Writer) writeZipFile(name string, data []byte) error {
	writer, err := w.zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, bytes.NewReader(data))
	return err
}

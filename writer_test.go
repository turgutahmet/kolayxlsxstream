package kolayxlsxstream

import (
	"archive/zip"
	"fmt"
	"os"
	"testing"
)

func TestBasicWrite(t *testing.T) {
	// Create temporary file
	tmpFile := "test_output.xlsx"
	defer os.Remove(tmpFile)

	// Create sink and writer
	sink, err := NewFileSink(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	writer := NewWriter(sink)

	// Start file with headers
	headers := []interface{}{"Name", "Age", "Email"}
	if err := writer.StartFile(headers); err != nil {
		t.Fatalf("Failed to start file: %v", err)
	}

	// Write some rows
	rows := [][]interface{}{
		{"John Doe", 30, "john@example.com"},
		{"Jane Smith", 25, "jane@example.com"},
		{"Bob Johnson", 35, "bob@example.com"},
	}

	if err := writer.WriteRows(rows); err != nil {
		t.Fatalf("Failed to write rows: %v", err)
	}

	// Finish file
	stats, err := writer.FinishFile()
	if err != nil {
		t.Fatalf("Failed to finish file: %v", err)
	}

	// Verify statistics
	if stats.TotalRows != 3 {
		t.Errorf("Expected 3 rows, got %d", stats.TotalRows)
	}

	if stats.TotalSheets != 1 {
		t.Errorf("Expected 1 sheet, got %d", stats.TotalSheets)
	}

	// Verify file exists and is a valid ZIP
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Output file does not exist")
	}

	// Open as ZIP and verify structure
	zipReader, err := zip.OpenReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open output as ZIP: %v", err)
	}
	defer zipReader.Close()

	expectedFiles := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"xl/workbook.xml",
		"xl/_rels/workbook.xml.rels",
		"xl/worksheets/sheet1.xml",
		"xl/styles.xml",
	}

	fileMap := make(map[string]bool)
	for _, f := range zipReader.File {
		fileMap[f.Name] = true
	}

	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("Expected file %s not found in ZIP", expected)
		}
	}
}

func TestMultiSheet(t *testing.T) {
	tmpFile := "test_multisheet.xlsx"
	defer os.Remove(tmpFile)

	sink, err := NewFileSink(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	// Configure for small sheets to test multi-sheet functionality
	config := DefaultConfig()
	config.MaxRowsPerSheet = 10

	writer := NewWriter(sink, config)

	if err := writer.StartFile([]interface{}{"ID", "Value"}); err != nil {
		t.Fatalf("Failed to start file: %v", err)
	}

	// Write 25 rows (should create 3 sheets: 10 + 10 + 5)
	for i := 1; i <= 25; i++ {
		if err := writer.WriteRow([]interface{}{i, fmt.Sprintf("Value %d", i)}); err != nil {
			t.Fatalf("Failed to write row %d: %v", i, err)
		}
	}

	stats, err := writer.FinishFile()
	if err != nil {
		t.Fatalf("Failed to finish file: %v", err)
	}

	if stats.TotalSheets != 3 {
		t.Errorf("Expected 3 sheets, got %d", stats.TotalSheets)
	}

	if stats.TotalRows != 25 {
		t.Errorf("Expected 25 rows, got %d", stats.TotalRows)
	}
}

func TestDataTypes(t *testing.T) {
	tmpFile := "test_datatypes.xlsx"
	defer os.Remove(tmpFile)

	sink, err := NewFileSink(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	writer := NewWriter(sink)

	if err := writer.StartFile([]interface{}{"String", "Int", "Float", "Bool", "Nil"}); err != nil {
		t.Fatalf("Failed to start file: %v", err)
	}

	// Test different data types
	row := []interface{}{
		"Hello World",
		42,
		3.14159,
		true,
		nil,
	}

	if err := writer.WriteRow(row); err != nil {
		t.Fatalf("Failed to write row: %v", err)
	}

	stats, err := writer.FinishFile()
	if err != nil {
		t.Fatalf("Failed to finish file: %v", err)
	}

	if stats.TotalRows != 1 {
		t.Errorf("Expected 1 row, got %d", stats.TotalRows)
	}
}

func TestCompression(t *testing.T) {
	tmpFile := "test_compression.xlsx"
	defer os.Remove(tmpFile)

	sink, err := NewFileSink(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	writer := NewWriter(sink)

	// Test setting compression level
	if err := writer.SetCompressionLevel(9); err != nil {
		t.Fatalf("Failed to set compression level: %v", err)
	}

	if err := writer.StartFile([]interface{}{"Data"}); err != nil {
		t.Fatalf("Failed to start file: %v", err)
	}

	// Write some data
	for i := 0; i < 100; i++ {
		if err := writer.WriteRow([]interface{}{fmt.Sprintf("Row %d", i)}); err != nil {
			t.Fatalf("Failed to write row: %v", err)
		}
	}

	stats, err := writer.FinishFile()
	if err != nil {
		t.Fatalf("Failed to finish file: %v", err)
	}

	if stats.TotalRows != 100 {
		t.Errorf("Expected 100 rows, got %d", stats.TotalRows)
	}
}

func TestErrorHandling(t *testing.T) {
	tmpFile := "test_error.xlsx"
	defer os.Remove(tmpFile)

	sink, err := NewFileSink(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	writer := NewWriter(sink)

	// Test writing before starting
	if err := writer.WriteRow([]interface{}{"test"}); err == nil {
		t.Error("Expected error when writing before starting file")
	}

	// Start file
	if err := writer.StartFile(); err != nil {
		t.Fatalf("Failed to start file: %v", err)
	}

	// Test starting twice
	if err := writer.StartFile(); err == nil {
		t.Error("Expected error when starting file twice")
	}

	// Finish file
	if _, err := writer.FinishFile(); err != nil {
		t.Fatalf("Failed to finish file: %v", err)
	}

	// Test writing after finishing
	if err := writer.WriteRow([]interface{}{"test"}); err == nil {
		t.Error("Expected error when writing after finishing file")
	}

	// Test finishing twice
	if _, err := writer.FinishFile(); err == nil {
		t.Error("Expected error when finishing file twice")
	}
}

func BenchmarkWriteRows(b *testing.B) {
	tmpFile := "benchmark_output.xlsx"
	defer os.Remove(tmpFile)

	sink, err := NewFileSink(tmpFile)
	if err != nil {
		b.Fatalf("Failed to create sink: %v", err)
	}

	writer := NewWriter(sink)

	if err := writer.StartFile([]interface{}{"ID", "Name", "Email", "Score"}); err != nil {
		b.Fatalf("Failed to start file: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		row := []interface{}{
			i,
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i),
			float64(i % 100),
		}
		if err := writer.WriteRow(row); err != nil {
			b.Fatalf("Failed to write row: %v", err)
		}
	}

	b.StopTimer()

	if _, err := writer.FinishFile(); err != nil {
		b.Fatalf("Failed to finish file: %v", err)
	}
}

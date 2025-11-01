package main

import (
	"fmt"
	"log"

	"github.com/turgutahmet/kolayxlsxstream"
)

func main() {
	// Create a file sink
	sink, err := kolayxlsxstream.NewFileSink("output.xlsx")
	if err != nil {
		log.Fatalf("Failed to create file sink: %v", err)
	}

	// Create a writer
	writer := kolayxlsxstream.NewWriter(sink)

	// Start the file with headers
	headers := []interface{}{"Name", "Email", "Phone", "Age"}
	if err := writer.StartFile(headers); err != nil {
		log.Fatalf("Failed to start file: %v", err)
	}

	// Write some rows
	rows := [][]interface{}{
		{"John Doe", "john@example.com", "+1234567890", 30},
		{"Jane Smith", "jane@example.com", "+0987654321", 25},
		{"Bob Johnson", "bob@example.com", "+1122334455", 35},
		{"Alice Williams", "alice@example.com", "+5544332211", 28},
	}

	for _, row := range rows {
		if err := writer.WriteRow(row); err != nil {
			log.Fatalf("Failed to write row: %v", err)
		}
	}

	// Or write multiple rows at once
	moreRows := [][]interface{}{
		{"Charlie Brown", "charlie@example.com", "+9988776655", 40},
		{"Diana Prince", "diana@example.com", "+1231231234", 32},
	}

	if err := writer.WriteRows(moreRows); err != nil {
		log.Fatalf("Failed to write rows: %v", err)
	}

	// Finish the file and get statistics
	stats, err := writer.FinishFile()
	if err != nil {
		log.Fatalf("Failed to finish file: %v", err)
	}

	// Print statistics
	fmt.Printf("File created successfully!\n")
	fmt.Printf("Total rows: %d\n", stats.TotalRows)
	fmt.Printf("Total sheets: %d\n", stats.TotalSheets)
	fmt.Printf("Duration: %.2f seconds\n", stats.Duration)
	fmt.Printf("Rows per second: %.2f\n", stats.RowsPerSecond)
}

package main

import (
	"fmt"
	"log"

	"github.com/turgutahmet/kolayxlsxstream"
)

func main() {
	// Create a file sink
	sink, err := kolayxlsxstream.NewFileSink("large_output.xlsx")
	if err != nil {
		log.Fatalf("Failed to create file sink: %v", err)
	}

	// Create a writer with custom configuration
	config := kolayxlsxstream.DefaultConfig()
	config.CompressionLevel = 1 // Fast compression for speed
	config.MaxRowsPerSheet = 100000 // Smaller sheets for demonstration

	writer := kolayxlsxstream.NewWriter(sink, config)

	// Start the file with headers
	headers := []interface{}{"ID", "Name", "Email", "Score", "Timestamp"}
	if err := writer.StartFile(headers); err != nil {
		log.Fatalf("Failed to start file: %v", err)
	}

	// Write 250,000 rows (will create 3 sheets with 100k rows each)
	totalRows := 250000
	batchSize := 1000

	fmt.Printf("Writing %d rows in batches of %d...\n", totalRows, batchSize)

	for i := 0; i < totalRows; i += batchSize {
		batch := make([][]interface{}, batchSize)
		for j := 0; j < batchSize && i+j < totalRows; j++ {
			rowNum := i + j + 1
			batch[j] = []interface{}{
				rowNum,
				fmt.Sprintf("User %d", rowNum),
				fmt.Sprintf("user%d@example.com", rowNum),
				float64(rowNum % 100),
				fmt.Sprintf("2025-01-%02d", (rowNum%28)+1),
			}
		}

		if err := writer.WriteRows(batch); err != nil {
			log.Fatalf("Failed to write batch: %v", err)
		}

		if (i+batchSize)%50000 == 0 {
			fmt.Printf("Progress: %d rows written\n", i+batchSize)
		}
	}

	// Finish the file and get statistics
	stats, err := writer.FinishFile()
	if err != nil {
		log.Fatalf("Failed to finish file: %v", err)
	}

	// Print statistics
	fmt.Printf("\nFile created successfully!\n")
	fmt.Printf("Total rows: %d\n", stats.TotalRows)
	fmt.Printf("Total sheets: %d\n", stats.TotalSheets)
	fmt.Printf("Duration: %.2f seconds\n", stats.Duration)
	fmt.Printf("Rows per second: %.0f\n", stats.RowsPerSecond)
}

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/turgutahmet/kolayxlsxstream"
)

// This example converts a CSV file to XLSX format

func main() {
	csvPath := flag.String("input", "input.csv", "Input CSV file path")
	xlsxPath := flag.String("output", "output.xlsx", "Output XLSX file path")
	hasHeaders := flag.Bool("headers", true, "First row contains headers")
	flag.Parse()

	fmt.Printf("Converting CSV to XLSX...\n")
	fmt.Printf("  Input:  %s\n", *csvPath)
	fmt.Printf("  Output: %s\n\n", *xlsxPath)

	// Open CSV file
	csvFile, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer csvFile.Close()

	// Create CSV reader
	csvReader := csv.NewReader(csvFile)
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true

	// Create XLSX writer
	sink, err := kolayxlsxstream.NewFileSink(*xlsxPath)
	if err != nil {
		log.Fatalf("Failed to create XLSX file: %v", err)
	}

	writer := kolayxlsxstream.NewWriter(sink)

	// Read and convert
	rowCount := 0
	firstRow := true

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error reading CSV at row %d: %v", rowCount+1, err)
		}

		// Convert []string to []interface{}
		row := make([]interface{}, len(record))
		for i, v := range record {
			row[i] = v
		}

		// Handle first row
		if firstRow {
			firstRow = false
			if *hasHeaders {
				if err := writer.StartFile(row); err != nil {
					log.Fatal(err)
				}
				rowCount++
				continue
			} else {
				if err := writer.StartFile(); err != nil {
					log.Fatal(err)
				}
			}
		}

		// Write data row
		if err := writer.WriteRow(row); err != nil {
			log.Fatalf("Failed to write row %d: %v", rowCount+1, err)
		}

		rowCount++
		if rowCount%10000 == 0 {
			fmt.Printf("  Progress: %d rows processed\n", rowCount)
		}
	}

	// Finish
	stats, err := writer.FinishFile()
	if err != nil {
		log.Fatalf("Failed to finish file: %v", err)
	}

	// Print results
	csvInfo, _ := os.Stat(*csvPath)
	xlsxInfo, _ := os.Stat(*xlsxPath)

	fmt.Printf("\n✅ Conversion completed!\n")
	fmt.Printf("Total rows:     %d", rowCount)
	if *hasHeaders {
		fmt.Printf(" (including header)")
	}
	fmt.Printf("\n")
	fmt.Printf("Data rows:      %d\n", stats.TotalRows)
	fmt.Printf("Total sheets:   %d\n", stats.TotalSheets)
	fmt.Printf("Duration:       %.2f seconds\n", stats.Duration)
	fmt.Printf("Rows/second:    %.0f\n", stats.RowsPerSecond)
	fmt.Printf("CSV size:       %.2f MB\n", float64(csvInfo.Size())/1024/1024)
	fmt.Printf("XLSX size:      %.2f MB\n", float64(xlsxInfo.Size())/1024/1024)
	fmt.Printf("Size ratio:     %.1f%%\n", float64(xlsxInfo.Size())/float64(csvInfo.Size())*100)
}

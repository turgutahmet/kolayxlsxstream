package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/turgutahmet/kolayxlsxstream"
)

// This example demonstrates streaming query results to XLSX
// It creates a sample SQLite database and exports it

func main() {
	output := flag.String("output", "database_export.xlsx", "Output XLSX file")
	sampleSize := flag.Int("sample", 100000, "Sample database size")
	flag.Parse()

	fmt.Printf("Database to XLSX Streaming Example\n\n")

	// Create and populate sample database
	fmt.Printf("Creating sample database with %d records...\n", *sampleSize)
	db := createSampleDatabase(*sampleSize)
	defer db.Close()

	// Export to XLSX
	fmt.Printf("Exporting to %s...\n\n", *output)
	exportToXLSX(db, *output)
}

func createSampleDatabase(size int) *sql.DB {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}

	// Create table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			email TEXT,
			age INTEGER,
			city TEXT,
			score REAL,
			created_at TEXT
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Insert sample data
	stmt, _ := db.Prepare(`
		INSERT INTO users (name, email, age, city, score, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	defer stmt.Close()

	cities := []string{"Istanbul", "Ankara", "Izmir", "Bursa", "Antalya"}
	for i := 1; i <= size; i++ {
		stmt.Exec(
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i),
			20+i%60,
			cities[i%len(cities)],
			float64(i%100)+0.5,
			time.Now().Add(-time.Duration(i)*time.Hour).Format("2006-01-02 15:04:05"),
		)
	}

	fmt.Printf("✅ Database created with %d records\n\n", size)
	return db
}

func exportToXLSX(db *sql.DB, outputPath string) {
	// Query all data
	rows, err := db.Query(`
		SELECT id, name, email, age, city, score, created_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Create XLSX writer
	sink, err := kolayxlsxstream.NewFileSink(outputPath)
	if err != nil {
		log.Fatal(err)
	}

	writer := kolayxlsxstream.NewWriter(sink)

	// Write headers
	headers := []interface{}{"ID", "Name", "Email", "Age", "City", "Score", "Created At"}
	if err := writer.StartFile(headers); err != nil {
		log.Fatal(err)
	}

	// Stream rows from database to XLSX
	rowCount := 0
	startTime := time.Now()

	for rows.Next() {
		var id, age int
		var name, email, city, createdAt string
		var score float64

		if err := rows.Scan(&id, &name, &email, &age, &city, &score, &createdAt); err != nil {
			log.Fatal(err)
		}

		// Write row
		row := []interface{}{id, name, email, age, city, score, createdAt}
		if err := writer.WriteRow(row); err != nil {
			log.Fatal(err)
		}

		rowCount++
		if rowCount%10000 == 0 {
			elapsed := time.Since(startTime).Seconds()
			fmt.Printf("  Progress: %d rows (%.0f rows/sec)\n", rowCount, float64(rowCount)/elapsed)
		}
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	// Finish file
	stats, err := writer.FinishFile()
	if err != nil {
		log.Fatal(err)
	}

	// Print summary
	fmt.Printf("\n✅ Export completed!\n")
	fmt.Printf("Total rows:    %d\n", stats.TotalRows)
	fmt.Printf("Total sheets:  %d\n", stats.TotalSheets)
	fmt.Printf("Duration:      %.2f seconds\n", stats.Duration)
	fmt.Printf("Rows/second:   %.0f\n", stats.RowsPerSecond)
	fmt.Printf("File:          %s\n", outputPath)
}

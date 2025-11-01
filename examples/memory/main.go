package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/turgutahmet/kolayxlsxstream"
)

// This example demonstrates constant memory usage regardless of file size

func main() {
	// Create CPU and memory profiles
	cpuFile, _ := os.Create("cpu.prof")
	memFile, _ := os.Create("mem.prof")
	defer cpuFile.Close()
	defer memFile.Close()

	pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()

	// Print initial memory stats
	printMemStats("Initial")

	// Create a large file
	sink, err := kolayxlsxstream.NewFileSink("memory_test.xlsx")
	if err != nil {
		log.Fatal(err)
	}

	config := kolayxlsxstream.DefaultConfig()
	config.CompressionLevel = 1 // Fast compression
	writer := kolayxlsxstream.NewWriter(sink, config)

	writer.StartFile([]interface{}{"ID", "Name", "Email", "Score", "Timestamp", "Status", "Notes"})

	printMemStats("After starting file")

	// Write 1 million rows and monitor memory
	totalRows := 1000000
	checkInterval := 100000

	fmt.Printf("\nWriting %d rows and monitoring memory...\n\n", totalRows)

	for i := 1; i <= totalRows; i++ {
		writer.WriteRow([]interface{}{
			i,
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i),
			float64(i % 100),
			"2025-01-01 12:00:00",
			"active",
			fmt.Sprintf("Notes for user %d with some longer text to increase data size", i),
		})

		if i%checkInterval == 0 {
			printMemStats(fmt.Sprintf("After %d rows", i))
		}
	}

	stats, err := writer.FinishFile()
	if err != nil {
		log.Fatal(err)
	}

	printMemStats("After finishing file")

	// Write memory profile
	runtime.GC()
	pprof.WriteHeapProfile(memFile)

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("Memory Usage Test Results")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Total rows:     %d\n", stats.TotalRows)
	fmt.Printf("Total sheets:   %d\n", stats.TotalSheets)
	fmt.Printf("Duration:       %.2f seconds\n", stats.Duration)
	fmt.Printf("Rows/second:    %.0f\n", stats.RowsPerSecond)

	fileInfo, _ := os.Stat("memory_test.xlsx")
	fmt.Printf("File size:      %.2f MB\n", float64(fileInfo.Size())/1024/1024)
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("\nProfiles saved:")
	fmt.Println("  - cpu.prof  (analyze with: go tool pprof cpu.prof)")
	fmt.Println("  - mem.prof  (analyze with: go tool pprof mem.prof)")
	fmt.Println("\nKey observation: Memory usage remains constant despite writing 1M rows!")
}

func printMemStats(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("[%s]\n", label)
	fmt.Printf("  Alloc:      %.2f MB (currently allocated)\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("  TotalAlloc: %.2f MB (cumulative allocated)\n", float64(m.TotalAlloc)/1024/1024)
	fmt.Printf("  Sys:        %.2f MB (obtained from system)\n", float64(m.Sys)/1024/1024)
	fmt.Printf("  NumGC:      %d (garbage collections)\n\n", m.NumGC)
}

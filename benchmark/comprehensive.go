package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/turgutahmet/kolayxlsxstream"
)

type BenchmarkResult struct {
	Rows          int
	Duration      float64
	RowsPerSecond float64
	MemoryMB      float64
	MemoryDelta   float64
	FileSize      int64
	FileSizeMB    float64
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  KolayXlsxStream - Comprehensive Benchmark Suite            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Test configurations matching PHP benchmark
	testSizes := []int{
		100, 500, 1000, 5000, 10000, 25000, 50000,
		100000, 250000, 500000, 750000, 1000000,
		1500000, 2000000,
	}

	fmt.Println("📊 Running Local File System Tests...")
	fmt.Println()
	localResults := make(map[int]*BenchmarkResult)

	for _, size := range testSizes {
		if size > 2000000 {
			fmt.Printf("⏭️  Skipping %d rows for local (too large)\n", size)
			continue
		}

		fmt.Printf("Testing %d rows (local)... ", size)
		result := benchmarkLocal(size)
		localResults[size] = result

		fmt.Printf("✓ %.2fs | %.0f rows/s | %.2f MB memory\n",
			result.Duration, result.RowsPerSecond, result.MemoryMB)

		// Cleanup
		os.Remove(fmt.Sprintf("benchmark_%d.xlsx", size))
	}

	fmt.Println()
	fmt.Println("☁️  Running S3 Streaming Tests...")
	fmt.Println()

	s3Results := make(map[int]*BenchmarkResult)

	for _, size := range testSizes {
		fmt.Printf("Testing %d rows (S3)... ", size)
		result := benchmarkS3(size)
		s3Results[size] = result

		if result != nil {
			fmt.Printf("✓ %.2fs | %.0f rows/s | %.2f MB (±%.2f) memory\n",
				result.Duration, result.RowsPerSecond, result.MemoryMB, result.MemoryDelta)
		} else {
			fmt.Println("✗ Failed")
		}

		time.Sleep(2 * time.Second) // Cooldown between tests
	}

	// Print comprehensive table
	printComprehensiveTable(testSizes, localResults, s3Results)

	// Generate markdown table
	generateMarkdownTable(testSizes, localResults, s3Results)
}

func benchmarkLocal(rows int) *BenchmarkResult {
	filename := fmt.Sprintf("benchmark_%d.xlsx", rows)

	// Measure initial memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	startTime := time.Now()

	// Create and write
	sink, _ := kolayxlsxstream.NewFileSink(filename)
	config := kolayxlsxstream.DefaultConfig()
	config.CompressionLevel = 1 // Fast compression like PHP
	writer := kolayxlsxstream.NewWriter(sink, config)

	writer.StartFile([]interface{}{"ID", "Name", "Email", "Score", "Status"})

	for i := 1; i <= rows; i++ {
		writer.WriteRow([]interface{}{
			i,
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i),
			float64(i % 100),
			"active",
		})
	}

	stats, _ := writer.FinishFile()
	duration := time.Since(startTime).Seconds()

	// Measure final memory
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Get file size
	fileInfo, _ := os.Stat(filename)

	return &BenchmarkResult{
		Rows:          rows,
		Duration:      duration,
		RowsPerSecond: stats.RowsPerSecond,
		MemoryMB:      float64(m2.Alloc) / 1024 / 1024,
		MemoryDelta:   float64(int64(m2.Alloc)-int64(m1.Alloc)) / 1024 / 1024,
		FileSize:      fileInfo.Size(),
		FileSizeMB:    float64(fileInfo.Size()) / 1024 / 1024,
	}
}

func benchmarkS3(rows int) *BenchmarkResult {
	ctx := context.Background()

	// Use credentials from .env.example
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"AKIA2NZP6PY6EI2T33H4",
			"yo5JTW+97yyR4ET8C4vGkhogH1PQAW+HnuXlDIcq",
			"",
		)),
	)
	if err != nil {
		return nil
	}

	client := s3.NewFromConfig(cfg)
	bucket := "uploadfilewithgrant"
	key := fmt.Sprintf("benchmarks/go_benchmark_%d_%d.xlsx", rows, time.Now().Unix())

	// Measure memory before
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	minMem := m1.Alloc
	maxMem := m1.Alloc

	startTime := time.Now()

	// Create S3 sink
	s3Options := kolayxlsxstream.DefaultS3Options()
	s3Options.PartSize = 32 * 1024 * 1024 // 32MB like PHP

	sink, err := kolayxlsxstream.NewS3Sink(ctx, client, bucket, key, s3Options)
	if err != nil {
		return nil
	}

	config := kolayxlsxstream.DefaultConfig()
	config.CompressionLevel = 1
	writer := kolayxlsxstream.NewWriter(sink, config)

	writer.StartFile([]interface{}{"ID", "Name", "Email", "Score", "Status"})

	// Write rows and track memory
	checkInterval := 1000
	for i := 1; i <= rows; i++ {
		writer.WriteRow([]interface{}{
			i,
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i),
			float64(i % 100),
			"active",
		})

		// Track memory fluctuation
		if i%checkInterval == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.Alloc < minMem {
				minMem = m.Alloc
			}
			if m.Alloc > maxMem {
				maxMem = m.Alloc
			}
		}
	}

	stats, err := writer.FinishFile()
	if err != nil {
		return nil
	}

	duration := time.Since(startTime).Seconds()

	// Final memory measurement
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	avgMem := float64(m2.Alloc) / 1024 / 1024
	memDelta := float64(maxMem-minMem) / 1024 / 1024

	return &BenchmarkResult{
		Rows:          rows,
		Duration:      duration,
		RowsPerSecond: stats.RowsPerSecond,
		MemoryMB:      avgMem,
		MemoryDelta:   memDelta,
		FileSize:      sink.TotalBytes(),
		FileSizeMB:    float64(sink.TotalBytes()) / 1024 / 1024,
	}
}

func printComprehensiveTable(sizes []int, local, s3 map[int]*BenchmarkResult) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                              COMPREHENSIVE BENCHMARK RESULTS                                                  ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-10s │ %-15s │ %-12s │ %-10s │ %-15s │ %-18s │ %-10s ║\n",
		"Rows", "Local Speed", "Local Mem", "Local Time", "S3 Speed", "S3 Memory", "File Size")
	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════════════════════════════════════╣")

	for _, size := range sizes {
		localRes := local[size]
		s3Res := s3[size]

		localSpeed := "-"
		localMem := "-"
		localTime := "-"
		s3Speed := "-"
		s3Mem := "-"
		fileSize := "-"

		if localRes != nil {
			localSpeed = fmt.Sprintf("%.0f rows/s", localRes.RowsPerSecond)
			localMem = fmt.Sprintf("%.0f MB", localRes.MemoryMB)
			localTime = fmt.Sprintf("%.2fs", localRes.Duration)
			fileSize = fmt.Sprintf("%.2f MB", localRes.FileSizeMB)
		}

		if s3Res != nil {
			s3Speed = fmt.Sprintf("%.0f rows/s", s3Res.RowsPerSecond)
			s3Mem = fmt.Sprintf("%.0f MB (±%.0f)", s3Res.MemoryMB, s3Res.MemoryDelta)
			if fileSize == "-" {
				fileSize = fmt.Sprintf("%.2f MB", s3Res.FileSizeMB)
			}
		}

		fmt.Printf("║ %-10s │ %-15s │ %-12s │ %-10s │ %-15s │ %-18s │ %-10s ║\n",
			formatNumber(size), localSpeed, localMem, localTime, s3Speed, s3Mem, fileSize)
	}

	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════════════════════════════════════╝")
}

func generateMarkdownTable(sizes []int, local, s3 map[int]*BenchmarkResult) {
	file, _ := os.Create("BENCHMARK_RESULTS.md")
	defer file.Close()

	file.WriteString("# Comprehensive Benchmark Results\n\n")
	file.WriteString("## Test Environment\n")
	file.WriteString(fmt.Sprintf("- **CPU**: %s\n", runtime.GOARCH))
	file.WriteString(fmt.Sprintf("- **Go Version**: %s\n", runtime.Version()))
	file.WriteString("- **OS**: " + runtime.GOOS + "\n")
	file.WriteString("- **Compression**: Level 1 (fastest)\n")
	file.WriteString("- **S3 Part Size**: 32 MB\n\n")

	file.WriteString("## Results\n\n")
	file.WriteString("| Rows | Local Speed | Local Memory | Local Time | S3 Speed | S3 Memory | S3 Time | File Size |\n")
	file.WriteString("|------|-------------|--------------|------------|----------|-----------|---------|----------|\n")

	for _, size := range sizes {
		localRes := local[size]
		s3Res := s3[size]

		localSpeed := "-"
		localMem := "-"
		localTime := "-"
		s3Speed := "-"
		s3Mem := "-"
		s3Time := "-"
		fileSize := "-"

		if localRes != nil {
			localSpeed = fmt.Sprintf("%.0f rows/s", localRes.RowsPerSecond)
			localMem = fmt.Sprintf("%.0f MB", localRes.MemoryMB)
			localTime = fmt.Sprintf("%.2fs", localRes.Duration)
			fileSize = fmt.Sprintf("%.2f MB", localRes.FileSizeMB)
		}

		if s3Res != nil {
			s3Speed = fmt.Sprintf("%.0f rows/s", s3Res.RowsPerSecond)
			s3Mem = fmt.Sprintf("%.0f MB (±%.0f)", s3Res.MemoryMB, s3Res.MemoryDelta)
			s3Time = fmt.Sprintf("%.2fs", s3Res.Duration)
			if fileSize == "-" {
				fileSize = fmt.Sprintf("%.2f MB", s3Res.FileSizeMB)
			}
		}

		file.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			formatNumber(size), localSpeed, localMem, localTime, s3Speed, s3Mem, s3Time, fileSize))
	}

	fmt.Println("\n✅ Markdown table saved to BENCHMARK_RESULTS.md")
}

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%dK", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

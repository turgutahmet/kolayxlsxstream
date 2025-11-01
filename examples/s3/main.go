package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/turgutahmet/kolayxlsxstream"
)

func main() {
	// Parse command line flags
	bucket := flag.String("bucket", os.Getenv("S3_BUCKET"), "S3 bucket name (or set S3_BUCKET env var)")
	key := flag.String("key", fmt.Sprintf("exports/report_%s.xlsx", time.Now().Format("20060102_150405")), "S3 key/path")
	region := flag.String("region", os.Getenv("AWS_REGION"), "AWS region (or set AWS_REGION env var, default: us-east-1)")
	rows := flag.Int("rows", 100000, "Number of rows to generate")
	partSize := flag.Int64("part-size", 10, "S3 multipart upload part size in MB")
	dryRun := flag.Bool("dry-run", false, "Don't actually upload to S3, just simulate")
	flag.Parse()

	// Dry run doesn't need S3 credentials
	if *dryRun {
		fmt.Println("🔵 DRY RUN MODE - No data will be uploaded to S3")
		if *bucket != "" {
			fmt.Printf("Would upload to: s3://%s/%s\n", *bucket, *key)
		}
		fmt.Printf("Generating %d rows locally...\n\n", *rows)
		runLocalSimulation(*rows)
		return
	}

	// Validate required parameters for actual S3 upload
	if *bucket == "" {
		log.Fatal("Error: S3 bucket is required. Use -bucket flag or set S3_BUCKET environment variable.\n\n" +
			"Example:\n" +
			"  go run main.go -bucket my-bucket -key exports/report.xlsx -rows 100000\n\n" +
			"Or with environment variables:\n" +
			"  export S3_BUCKET=my-bucket\n" +
			"  export AWS_REGION=us-east-1\n" +
			"  go run main.go\n\n" +
			"Or use dry-run mode to test locally:\n" +
			"  go run main.go -dry-run -rows 10000\n")
	}

	// Set default region
	if *region == "" {
		*region = "us-east-1"
	}

	fmt.Printf("📦 S3 Upload Configuration:\n")
	fmt.Printf("   Bucket: %s\n", *bucket)
	fmt.Printf("   Key: %s\n", *key)
	fmt.Printf("   Region: %s\n", *region)
	fmt.Printf("   Rows: %d\n", *rows)
	fmt.Printf("   Part Size: %d MB\n\n", *partSize)

	// Load AWS configuration
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(*region),
	)
	if err != nil {
		log.Fatalf("❌ Failed to load AWS config: %v\n\n"+
			"Make sure you have AWS credentials configured:\n"+
			"  1. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables, or\n"+
			"  2. Use AWS CLI: aws configure, or\n"+
			"  3. Use IAM role (if running on EC2/ECS/Lambda)\n", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Test S3 access (optional - some IAM policies don't allow HeadBucket)
	fmt.Println("🔍 Testing S3 access...")
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: bucket,
	})
	if err != nil {
		fmt.Printf("⚠️  Cannot verify bucket access (HeadBucket permission may not be granted)\n")
		fmt.Printf("   Will attempt upload anyway. If upload fails, check:\n")
		fmt.Printf("   1. Bucket exists: %s\n", *bucket)
		fmt.Printf("   2. AWS credentials are correct\n")
		fmt.Printf("   3. Region is correct: %s\n\n", *region)
	} else {
		fmt.Println("✅ S3 bucket accessible")
		fmt.Println()
	}

	// Create S3 sink with custom options
	s3Options := kolayxlsxstream.DefaultS3Options()
	s3Options.PartSize = *partSize * 1024 * 1024 // Convert MB to bytes
	s3Options.ACL = types.ObjectCannedACLPrivate
	s3Options.StorageClass = types.StorageClassStandard
	s3Options.Metadata = map[string]string{
		"generated-by": "kolayxlsxstream",
		"rows":         fmt.Sprintf("%d", *rows),
	}

	sink, err := kolayxlsxstream.NewS3Sink(ctx, client, *bucket, *key, s3Options)
	if err != nil {
		log.Fatalf("❌ Failed to create S3 sink: %v", err)
	}

	// Create writer with balanced compression
	writerConfig := kolayxlsxstream.DefaultConfig()
	writerConfig.CompressionLevel = 6

	writer := kolayxlsxstream.NewWriter(sink, writerConfig)

	// Start the file with headers
	headers := []interface{}{"ID", "Product", "Quantity", "Price", "Total"}
	if err := writer.StartFile(headers); err != nil {
		log.Fatalf("❌ Failed to start file: %v", err)
	}

	// Write data rows with progress
	fmt.Printf("📝 Writing %d rows to S3...\n", *rows)
	startTime := time.Now()

	for i := 1; i <= *rows; i++ {
		row := []interface{}{
			i,
			fmt.Sprintf("Product %d", i),
			i % 100,
			float64(i) * 0.99,
			float64(i%100) * float64(i) * 0.99,
		}

		if err := writer.WriteRow(row); err != nil {
			log.Fatalf("❌ Failed to write row %d: %v", i, err)
		}

		// Progress updates
		if i%10000 == 0 {
			elapsed := time.Since(startTime).Seconds()
			rowsPerSec := float64(i) / elapsed
			fmt.Printf("   Progress: %d/%d rows (%.0f rows/sec), %d parts uploaded\n",
				i, *rows, rowsPerSec, sink.PartCount())
		}
	}

	// Finish the file and get statistics
	fmt.Println("\n⏳ Finalizing upload...")
	stats, err := writer.FinishFile()
	if err != nil {
		log.Fatalf("❌ Failed to finish file: %v", err)
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ SUCCESS! File uploaded to S3")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("S3 URL:        s3://%s/%s\n", *bucket, *key)
	fmt.Printf("Total rows:    %d\n", stats.TotalRows)
	fmt.Printf("Total sheets:  %d\n", stats.TotalSheets)
	fmt.Printf("Duration:      %.2f seconds\n", stats.Duration)
	fmt.Printf("Rows/second:   %.0f\n", stats.RowsPerSecond)
	fmt.Printf("File size:     %.2f MB\n", float64(sink.TotalBytes())/1024/1024)
	fmt.Printf("Parts:         %d\n", sink.PartCount())
	fmt.Println(strings.Repeat("=", 60))
}

// runLocalSimulation simulates the upload without actually uploading to S3
func runLocalSimulation(rows int) {
	tmpFile := fmt.Sprintf("dry_run_output_%s.xlsx", time.Now().Format("20060102_150405"))
	fmt.Printf("Creating local file: %s\n\n", tmpFile)

	sink, err := kolayxlsxstream.NewFileSink(tmpFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}

	writer := kolayxlsxstream.NewWriter(sink)
	writer.StartFile([]interface{}{"ID", "Product", "Quantity", "Price", "Total"})

	fmt.Printf("Generating %d rows...\n", rows)
	for i := 1; i <= rows; i++ {
		writer.WriteRow([]interface{}{
			i,
			fmt.Sprintf("Product %d", i),
			i % 100,
			float64(i) * 0.99,
			float64(i%100) * float64(i) * 0.99,
		})
		if i%10000 == 0 {
			fmt.Printf("  Progress: %d/%d rows\n", i, rows)
		}
	}

	stats, _ := writer.FinishFile()

	fileInfo, _ := os.Stat(tmpFile)
	fmt.Printf("\n✅ Dry run completed!\n")
	fmt.Printf("File: %s\n", tmpFile)
	fmt.Printf("Rows: %d\n", stats.TotalRows)
	fmt.Printf("Size: %.2f MB\n", float64(fileInfo.Size())/1024/1024)
	fmt.Printf("Duration: %.2f seconds\n", stats.Duration)
	fmt.Printf("Rows/second: %.0f\n", stats.RowsPerSecond)
}

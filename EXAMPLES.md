# Examples Guide

All examples are in the `examples/` directory. Each example is a standalone Go program.

## 📁 Available Examples

### 1. Basic Usage (`examples/basic/`)
Simple example showing the core API.

```bash
cd examples/basic
go run main.go
```

**What it does:**
- Creates a simple XLSX file with headers
- Writes a few rows of data
- Shows basic usage of Writer API

---

### 2. Large File Export (`examples/large/`)
Demonstrates streaming large datasets with automatic multi-sheet creation.

```bash
cd examples/large
go run main.go
```

**What it does:**
- Writes 250,000 rows
- Automatically creates 3 sheets (100k rows each)
- Shows batch writing for performance
- Displays progress updates

**Performance:** ~500,000 rows/second

---

### 3. S3 Streaming (`examples/s3/`)
Upload directly to AWS S3 with multipart upload.

```bash
# Test locally without AWS credentials
cd examples/s3
go run main.go -dry-run -rows 50000

# Upload to S3 (requires AWS credentials)
export S3_BUCKET=my-bucket
export AWS_REGION=us-east-1
go run main.go -rows 100000

# Or with command line flags
go run main.go -bucket my-bucket -key exports/report.xlsx -rows 100000
```

**Command line options:**
- `-bucket`: S3 bucket name (or set S3_BUCKET env var)
- `-key`: S3 object key (default: auto-generated with timestamp)
- `-region`: AWS region (default: us-east-1)
- `-rows`: Number of rows to generate (default: 100000)
- `-part-size`: Multipart part size in MB (default: 10)
- `-dry-run`: Test locally without S3

**What it does:**
- Streams data directly to S3 using multipart upload
- No temporary files created
- Tests S3 bucket access before starting
- Shows real-time progress with part counts
- Dry-run mode for testing without AWS

---

### 4. CSV to XLSX Converter (`examples/csv-to-xlsx/`)
Convert CSV files to XLSX format.

```bash
cd examples/csv-to-xlsx
go run main.go -input sample.csv -output converted.xlsx

# Or with custom options
go run main.go -input data.csv -output data.xlsx -headers=false
```

**Options:**
- `-input`: Input CSV file path (default: input.csv)
- `-output`: Output XLSX file path (default: output.xlsx)  
- `-headers`: First row contains headers (default: true)

**What it does:**
- Reads CSV file line by line (streaming)
- Converts to XLSX format
- Handles headers automatically
- Shows file size comparison

---

### 5. Database Export (`examples/database/`)
Stream database query results directly to XLSX.

```bash
cd examples/database
go run main.go -sample 100000 -output export.xlsx
```

**Options:**
- `-sample`: Number of sample records to create (default: 100000)
- `-output`: Output XLSX file path (default: database_export.xlsx)

**What it does:**
- Creates an in-memory SQLite database
- Populates with sample data
- Streams query results to XLSX
- No intermediate storage needed
- Shows real-time progress

**Use with your database:**
```go
// Query your database
rows, _ := db.Query("SELECT * FROM large_table")

// Stream to XLSX
for rows.Next() {
    var col1, col2, col3 string
    rows.Scan(&col1, &col2, &col3)
    writer.WriteRow([]interface{}{col1, col2, col3})
}
```

---

### 6. Memory Profiling (`examples/memory/`)
Demonstrates constant memory usage with large files.

```bash
cd examples/memory
go run main.go
```

**What it does:**
- Writes 1 million rows
- Monitors memory usage at intervals
- Generates CPU and memory profiles
- Proves O(1) memory complexity

**Output:**
- `memory_test.xlsx` - The generated file
- `cpu.prof` - CPU profile
- `mem.prof` - Memory profile

**Analyze profiles:**
```bash
go tool pprof cpu.prof
go tool pprof mem.prof
```

**Key observation:** Memory usage stays constant (~2-5 MB) regardless of file size!

---

## 🔧 Running All Examples

```bash
# Run all examples
for dir in examples/*/; do
    echo "Running example: $dir"
    (cd "$dir" && go run main.go 2>/dev/null || echo "Skipped (requires config)")
done
```

## 📊 Performance Comparison

| Example | Rows | Time | Rows/Sec | Memory |
|---------|------|------|----------|--------|
| Basic | 6 | 0.00s | ~3,600 | <1 MB |
| Large | 250,000 | 0.47s | ~528,000 | ~2 MB |
| Database | 100,000 | 0.66s | ~152,000 | ~2 MB |
| Memory Test | 1,000,000 | ~2.5s | ~400,000 | ~2-5 MB |

*Tested on Apple M4*

## 💡 Tips

1. **Start with `basic`** - Learn the core API
2. **Try `dry-run` for S3** - Test without AWS credentials
3. **Use `large` example** - See multi-sheet in action
4. **Run `memory` example** - Understand constant memory usage
5. **Adapt `database` example** - For your real database exports

## 🐛 Troubleshooting

### S3 Example

**Problem:** "Cannot access bucket"
```bash
# Check AWS credentials
aws s3 ls s3://your-bucket/

# Use dry-run to test locally
go run main.go -dry-run
```

### Database Example

**Problem:** "no such table"
- The example creates its own in-memory database
- To use your database, modify the connection string

### Memory Example

**Problem:** "too slow"
- Reduce sample size in code
- Lower compression level
- Use faster disk

## 📚 Further Reading

- [S3 Example README](s3/README.md) - Detailed S3 configuration
- [Main README](../README.md) - Full API documentation
- [Go Documentation](https://pkg.go.dev/github.com/turgutahmet/kolayxlsxstream)

# Comprehensive Benchmark Results

## Test Environment
- **CPU**: arm64
- **Go Version**: go1.25.3
- **OS**: darwin
- **Compression**: Level 1 (fastest)
- **S3 Part Size**: 32 MB

## Results

| Rows | Local Speed | Local Memory | Local Time | S3 Speed | S3 Memory | S3 Time | File Size |
|------|-------------|--------------|------------|----------|-----------|---------|----------|
| 100 | 48631 rows/s | 0 MB | 0.00s | 201 rows/s | 1 MB (±0) | 1.02s | 0.00 MB |
| 500 | 136626 rows/s | 0 MB | 0.00s | 822 rows/s | 1 MB (±0) | 1.12s | 0.02 MB |
| 1K | 198354 rows/s | 0 MB | 0.01s | 1570 rows/s | 1 MB (±1) | 1.14s | 0.03 MB |
| 5K | 313072 rows/s | 0 MB | 0.02s | 5289 rows/s | 1 MB (±2) | 1.45s | 0.15 MB |
| 10K | 479633 rows/s | 0 MB | 0.02s | 3041 rows/s | 1 MB (±4) | 3.80s | 0.30 MB |
| 25K | 559883 rows/s | 0 MB | 0.04s | 4782 rows/s | 2 MB (±5) | 5.74s | 0.75 MB |
| 50K | 588989 rows/s | 0 MB | 0.08s | 32805 rows/s | 3 MB (±9) | 2.04s | 1.50 MB |
| 100K | 607758 rows/s | 0 MB | 0.16s | 12355 rows/s | 5 MB (±15) | 8.58s | 3.00 MB |
| 250K | 598598 rows/s | 0 MB | 0.42s | 112810 rows/s | 9 MB (±26) | 2.77s | 7.54 MB |
| 500K | 598574 rows/s | 0 MB | 0.84s | 107857 rows/s | 17 MB (±49) | 5.14s | 15.10 MB |
| 750K | 599608 rows/s | 0 MB | 1.25s | 64397 rows/s | 33 MB (±96) | 12.13s | 22.67 MB |
| 1.0M | 595634 rows/s | 0 MB | 1.68s | 54525 rows/s | 33 MB (±96) | 18.83s | 30.21 MB |
| 1.5M | 577251 rows/s | 0 MB | 2.60s | 72855 rows/s | 33 MB (±96) | 21.10s | 45.38 MB |
| 2.0M | 576384 rows/s | 0 MB | 3.47s | 91518 rows/s | 33 MB (±96) | 22.37s | 60.58 MB |

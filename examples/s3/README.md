# S3 Streaming Example

This example demonstrates how to stream XLSX files directly to AWS S3 using multipart uploads.

## Quick Start (No AWS Credentials Needed)

Test locally without S3:

```bash
go run main.go -dry-run -rows 10000
```

## AWS Configuration

### Option 1: Environment Variables

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1
export S3_BUCKET=your-bucket-name

go run main.go -rows 100000
```

### Option 2: AWS CLI Configuration

```bash
aws configure
# Then run:
go run main.go -bucket your-bucket-name -rows 100000
```

### Option 3: Command Line Flags

```bash
go run main.go \
  -bucket my-bucket \
  -key exports/report.xlsx \
  -region us-east-1 \
  -rows 100000 \
  -part-size 10
```

## Command Line Options

- `-bucket`: S3 bucket name (required, or set S3_BUCKET env var)
- `-key`: S3 object key/path (default: auto-generated with timestamp)
- `-region`: AWS region (default: us-east-1, or set AWS_REGION env var)
- `-rows`: Number of rows to generate (default: 100000)
- `-part-size`: Multipart upload part size in MB (default: 10)
- `-dry-run`: Test locally without uploading to S3

## Examples

### Upload 1 million rows

```bash
go run main.go -bucket my-bucket -rows 1000000
```

### Custom part size (larger parts = faster upload)

```bash
go run main.go -bucket my-bucket -part-size 50 -rows 500000
```

### Specific S3 path

```bash
go run main.go -bucket my-bucket -key reports/monthly/2025-01.xlsx -rows 100000
```

## IAM Permissions Required

Your AWS credentials need these S3 permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:AbortMultipartUpload",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::your-bucket-name/*",
        "arn:aws:s3:::your-bucket-name"
      ]
    }
  ]
}
```

## Performance Tips

1. **Larger part sizes** (32-100MB) = faster uploads
2. **Lower compression** (1-3) = faster processing
3. **Batch writes** = better throughput
4. **Same region** = lower latency

## Troubleshooting

### "Failed to load AWS config"

Make sure you have AWS credentials configured. Try:
```bash
aws configure
```

### "Cannot access bucket"

Check:
1. Bucket exists in the correct region
2. Your credentials have the required permissions
3. Region is correct

### Testing without AWS

Use dry-run mode:
```bash
go run main.go -dry-run -rows 10000
```

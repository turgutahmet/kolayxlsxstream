package kolayxlsxstream

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Mock S3 client for testing
type mockS3Client struct {
	createMultipartUploadFunc func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	uploadPartFunc            func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	completeMultipartUpload   func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	abortMultipartUploadFunc  func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
}

func (m *mockS3Client) CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	if m.createMultipartUploadFunc != nil {
		return m.createMultipartUploadFunc(ctx, params, optFns...)
	}
	return &s3.CreateMultipartUploadOutput{
		UploadId: aws.String("test-upload-id"),
	}, nil
}

func (m *mockS3Client) UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	if m.uploadPartFunc != nil {
		return m.uploadPartFunc(ctx, params, optFns...)
	}
	return &s3.UploadPartOutput{
		ETag: aws.String("test-etag"),
	}, nil
}

func (m *mockS3Client) CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	if m.completeMultipartUpload != nil {
		return m.completeMultipartUpload(ctx, params, optFns...)
	}
	return &s3.CompleteMultipartUploadOutput{}, nil
}

func (m *mockS3Client) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	if m.abortMultipartUploadFunc != nil {
		return m.abortMultipartUploadFunc(ctx, params, optFns...)
	}
	return &s3.AbortMultipartUploadOutput{}, nil
}

func TestS3SinkPartSizeValidation(t *testing.T) {
	ctx := context.Background()
	client := &mockS3Client{}

	tests := []struct {
		name        string
		partSize    int64
		shouldError bool
	}{
		{"Valid 5MB", 5 * 1024 * 1024, false},
		{"Valid 10MB", 10 * 1024 * 1024, false},
		{"Valid 32MB", 32 * 1024 * 1024, false},
		{"Invalid 4MB", 4 * 1024 * 1024, true},
		{"Invalid 1MB", 1 * 1024 * 1024, true},
		{"Invalid 0", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &S3Options{
				PartSize:    tt.partSize,
				ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			}

			sink, err := NewS3Sink(ctx, client, "test-bucket", "test-key", opts)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for part size %d, got nil", tt.partSize)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for part size %d: %v", tt.partSize, err)
				}
				if sink != nil {
					_ = sink.Abort()
				}
			}
		})
	}
}

func TestS3SinkCreateMultipartUploadFailure(t *testing.T) {
	ctx := context.Background()
	client := &mockS3Client{
		createMultipartUploadFunc: func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	sink, err := NewS3Sink(ctx, client, "test-bucket", "test-key")
	if err == nil {
		t.Error("Expected error when CreateMultipartUpload fails")
	}
	if sink != nil {
		t.Error("Sink should be nil when initialization fails")
	}
}

func TestS3SinkUploadPartFailure(t *testing.T) {
	ctx := context.Background()

	uploadAttempts := 0
	client := &mockS3Client{
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			uploadAttempts++
			return nil, fmt.Errorf("network error")
		},
	}

	sink, err := NewS3Sink(ctx, client, "test-bucket", "test-key")
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}
	defer sink.Abort()

	// Write enough data to trigger a part upload (default is 32MB)
	data := bytes.Repeat([]byte("x"), 33*1024*1024)
	_, err = sink.Write(data)

	if err == nil {
		t.Error("Expected error when UploadPart fails")
	}
	if uploadAttempts == 0 {
		t.Error("UploadPart should have been called")
	}
}

func TestS3SinkCompleteMultipartUploadFailure(t *testing.T) {
	ctx := context.Background()
	client := &mockS3Client{
		completeMultipartUpload: func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
			return nil, fmt.Errorf("internal error")
		},
	}

	sink, err := NewS3Sink(ctx, client, "test-bucket", "test-key")
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	// Write some data
	_, err = sink.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Close should fail because CompleteMultipartUpload fails
	err = sink.Close()
	if err == nil {
		t.Error("Expected error when CompleteMultipartUpload fails")
	}
}

func TestS3SinkMultipartUploadFlow(t *testing.T) {
	ctx := context.Background()

	uploadedParts := 0
	completeCalled := false

	client := &mockS3Client{
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			uploadedParts++
			return &s3.UploadPartOutput{
				ETag: aws.String(fmt.Sprintf("etag-%d", uploadedParts)),
			}, nil
		},
		completeMultipartUpload: func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
			completeCalled = true
			if len(params.MultipartUpload.Parts) != uploadedParts {
				t.Errorf("Expected %d parts in complete request, got %d", uploadedParts, len(params.MultipartUpload.Parts))
			}
			return &s3.CompleteMultipartUploadOutput{}, nil
		},
	}

	opts := &S3Options{
		PartSize:    5 * 1024 * 1024, // 5MB parts
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	sink, err := NewS3Sink(ctx, client, "test-bucket", "test-key", opts)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	// Write 12MB of data (should create 2 parts of 5MB each, plus 2MB in buffer)
	totalBytes := 12 * 1024 * 1024
	written := 0
	chunkSize := 1024 * 1024 // Write 1MB at a time

	for written < totalBytes {
		toWrite := chunkSize
		if written+toWrite > totalBytes {
			toWrite = totalBytes - written
		}
		data := bytes.Repeat([]byte("x"), toWrite)
		n, err := sink.Write(data)
		if err != nil {
			t.Fatalf("Write failed at %d bytes: %v", written, err)
		}
		written += n
	}

	// Should have uploaded 2 parts so far
	if uploadedParts != 2 {
		t.Errorf("Expected 2 parts uploaded before close, got %d", uploadedParts)
	}

	// Close should upload the remaining 2MB and complete
	err = sink.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Should have uploaded 3 parts total
	if uploadedParts != 3 {
		t.Errorf("Expected 3 parts total, got %d", uploadedParts)
	}

	if !completeCalled {
		t.Error("CompleteMultipartUpload should have been called")
	}

	// Verify statistics
	if sink.TotalBytes() != int64(totalBytes) {
		t.Errorf("Expected %d total bytes, got %d", totalBytes, sink.TotalBytes())
	}

	if sink.PartCount() != 3 {
		t.Errorf("Expected 3 parts, got %d", sink.PartCount())
	}
}

func TestS3SinkAbort(t *testing.T) {
	ctx := context.Background()
	abortCalled := false

	client := &mockS3Client{
		abortMultipartUploadFunc: func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
			abortCalled = true
			return &s3.AbortMultipartUploadOutput{}, nil
		},
	}

	sink, err := NewS3Sink(ctx, client, "test-bucket", "test-key")
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}

	err = sink.Abort()
	if err != nil {
		t.Errorf("Abort failed: %v", err)
	}

	if !abortCalled {
		t.Error("AbortMultipartUpload should have been called")
	}
}

func TestS3SinkOptions(t *testing.T) {
	ctx := context.Background()
	client := &mockS3Client{}

	opts := &S3Options{
		PartSize:             10 * 1024 * 1024,
		ContentType:          "application/custom",
		ACL:                  types.ObjectCannedACLPublicRead,
		StorageClass:         types.StorageClassGlacier,
		ServerSideEncryption: types.ServerSideEncryptionAes256,
		Metadata: map[string]string{
			"custom-key": "custom-value",
		},
	}

	sink, err := NewS3Sink(ctx, client, "test-bucket", "test-key", opts)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}
	defer sink.Abort()

	if sink.options.PartSize != opts.PartSize {
		t.Errorf("PartSize not set correctly")
	}
	if sink.options.ContentType != opts.ContentType {
		t.Errorf("ContentType not set correctly")
	}
	if sink.options.ACL != opts.ACL {
		t.Errorf("ACL not set correctly")
	}
	if sink.options.StorageClass != opts.StorageClass {
		t.Errorf("StorageClass not set correctly")
	}
	if sink.options.ServerSideEncryption != opts.ServerSideEncryption {
		t.Errorf("ServerSideEncryption not set correctly")
	}
}

func TestS3SinkFromReader(t *testing.T) {
	ctx := context.Background()
	uploadedData := &bytes.Buffer{}

	client := &mockS3Client{
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			// Read the data being uploaded
			data, _ := io.ReadAll(params.Body)
			uploadedData.Write(data)
			return &s3.UploadPartOutput{
				ETag: aws.String("test-etag"),
			}, nil
		},
	}

	testData := []byte("This is test data for S3SinkFromReader")
	reader := bytes.NewReader(testData)

	err := S3SinkFromReader(ctx, client, "test-bucket", "test-key", reader)
	if err != nil {
		t.Fatalf("S3SinkFromReader failed: %v", err)
	}

	if !bytes.Equal(uploadedData.Bytes(), testData) {
		t.Errorf("Uploaded data doesn't match original data")
	}
}

func TestS3SinkFromReaderFailure(t *testing.T) {
	ctx := context.Background()
	client := &mockS3Client{
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			return nil, fmt.Errorf("upload failed")
		},
	}

	reader := bytes.NewReader([]byte("test data"))

	err := S3SinkFromReader(ctx, client, "test-bucket", "test-key", reader)
	if err == nil {
		t.Error("Expected error when upload fails in S3SinkFromReader")
	}
}

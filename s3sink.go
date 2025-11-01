package kolayxlsxstream

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Sink writes data to AWS S3 using multipart upload
type S3Sink struct {
	client  *s3.Client
	bucket  string
	key     string
	ctx     context.Context
	options *S3Options

	uploadID       *string
	buffer         *bytes.Buffer
	partNumber     int32
	completedParts []types.CompletedPart
	totalBytes     int64
}

// S3Options contains optional configuration for S3 uploads
type S3Options struct {
	// PartSize is the size of each multipart upload part in bytes (default: 32MB)
	// Must be at least 5MB (except for the last part)
	PartSize int64

	// ACL sets the canned ACL for the object (e.g., "private", "public-read")
	ACL types.ObjectCannedACL

	// ContentType sets the MIME type of the object (default: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ContentType string

	// Metadata sets custom metadata for the object
	Metadata map[string]string

	// StorageClass sets the storage class (e.g., STANDARD, INTELLIGENT_TIERING, GLACIER)
	StorageClass types.StorageClass

	// ServerSideEncryption sets the server-side encryption method (e.g., AES256, aws:kms)
	ServerSideEncryption types.ServerSideEncryption

	// SSEKMSKeyId sets the KMS key ID for server-side encryption with KMS
	SSEKMSKeyId *string
}

// DefaultS3Options returns the default S3 options
func DefaultS3Options() *S3Options {
	return &S3Options{
		PartSize:    32 * 1024 * 1024, // 32MB
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}
}

// NewS3Sink creates a new S3 sink that writes to the specified bucket and key
func NewS3Sink(ctx context.Context, client *s3.Client, bucket, key string, options ...*S3Options) (*S3Sink, error) {
	opts := DefaultS3Options()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}

	// Validate part size (minimum 5MB except for last part)
	if opts.PartSize < 5*1024*1024 {
		return nil, fmt.Errorf("part size must be at least 5MB")
	}

	sink := &S3Sink{
		client:     client,
		bucket:     bucket,
		key:        key,
		ctx:        ctx,
		options:    opts,
		buffer:     new(bytes.Buffer),
		partNumber: 1,
	}

	// Initiate multipart upload
	if err := sink.initiateMultipartUpload(); err != nil {
		return nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	return sink, nil
}

// Write implements io.Writer interface
func (s *S3Sink) Write(p []byte) (n int, err error) {
	n, err = s.buffer.Write(p)
	s.totalBytes += int64(n)

	// If buffer exceeds part size, upload the part
	if s.buffer.Len() >= int(s.options.PartSize) {
		if err := s.uploadPart(); err != nil {
			return n, fmt.Errorf("failed to upload part: %w", err)
		}
	}

	return n, err
}

// Close implements io.Closer interface and completes the multipart upload
func (s *S3Sink) Close() error {
	// Upload any remaining data in the buffer
	if s.buffer.Len() > 0 {
		if err := s.uploadPart(); err != nil {
			return fmt.Errorf("failed to upload final part: %w", err)
		}
	}

	// Complete the multipart upload
	if err := s.completeMultipartUpload(); err != nil {
		// If completion fails, abort the upload
		_ = s.abortMultipartUpload()
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// initiateMultipartUpload starts a new multipart upload
func (s *S3Sink) initiateMultipartUpload() error {
	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.key),
		ContentType: aws.String(s.options.ContentType),
	}

	// Add optional parameters
	if s.options.ACL != "" {
		input.ACL = s.options.ACL
	}
	if s.options.Metadata != nil {
		input.Metadata = s.options.Metadata
	}
	if s.options.StorageClass != "" {
		input.StorageClass = s.options.StorageClass
	}
	if s.options.ServerSideEncryption != "" {
		input.ServerSideEncryption = s.options.ServerSideEncryption
	}
	if s.options.SSEKMSKeyId != nil {
		input.SSEKMSKeyId = s.options.SSEKMSKeyId
	}

	result, err := s.client.CreateMultipartUpload(s.ctx, input)
	if err != nil {
		return err
	}

	s.uploadID = result.UploadId
	return nil
}

// uploadPart uploads the current buffer as a part
func (s *S3Sink) uploadPart() error {
	if s.buffer.Len() == 0 {
		return nil
	}

	data := s.buffer.Bytes()
	input := &s3.UploadPartInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(s.key),
		PartNumber: aws.Int32(s.partNumber),
		UploadId:   s.uploadID,
		Body:       bytes.NewReader(data),
	}

	result, err := s.client.UploadPart(s.ctx, input)
	if err != nil {
		return err
	}

	// Add to completed parts
	s.completedParts = append(s.completedParts, types.CompletedPart{
		ETag:       result.ETag,
		PartNumber: aws.Int32(s.partNumber),
	})

	// Reset buffer and increment part number
	s.buffer.Reset()
	s.partNumber++

	return nil
}

// completeMultipartUpload finalizes the multipart upload
func (s *S3Sink) completeMultipartUpload() error {
	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(s.key),
		UploadId: s.uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: s.completedParts,
		},
	}

	_, err := s.client.CompleteMultipartUpload(s.ctx, input)
	return err
}

// abortMultipartUpload cancels the multipart upload
func (s *S3Sink) abortMultipartUpload() error {
	if s.uploadID == nil {
		return nil
	}

	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(s.key),
		UploadId: s.uploadID,
	}

	_, err := s.client.AbortMultipartUpload(s.ctx, input)
	return err
}

// TotalBytes returns the total bytes written so far
func (s *S3Sink) TotalBytes() int64 {
	return s.totalBytes
}

// PartCount returns the number of parts uploaded
func (s *S3Sink) PartCount() int {
	return len(s.completedParts)
}

// Abort cancels the multipart upload (useful for error handling)
func (s *S3Sink) Abort() error {
	return s.abortMultipartUpload()
}

// S3SinkFromReader creates an S3Sink and copies data from a reader
// This is a convenience function for uploading existing data to S3
func S3SinkFromReader(ctx context.Context, client *s3.Client, bucket, key string, reader io.Reader, options ...*S3Options) error {
	sink, err := NewS3Sink(ctx, client, bucket, key, options...)
	if err != nil {
		return err
	}

	if _, err := io.Copy(sink, reader); err != nil {
		_ = sink.Abort()
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return sink.Close()
}

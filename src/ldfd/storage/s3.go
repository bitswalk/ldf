package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds the S3 storage configuration
type S3Config struct {
	// Endpoint is the S3-compatible endpoint URL (e.g., "https://s3.amazonaws.com" or "http://minio:9000")
	Endpoint string

	// Region is the S3 region (e.g., "us-east-1")
	Region string

	// Bucket is the default bucket name for storing artifacts
	Bucket string

	// AccessKeyID is the S3 access key
	AccessKeyID string

	// SecretAccessKey is the S3 secret key
	SecretAccessKey string

	// UsePathStyle enables path-style addressing (required for most S3-compatible storage)
	UsePathStyle bool
}

// S3Backend implements storage using S3-compatible object storage
type S3Backend struct {
	s3Client *s3.Client
	config   S3Config
}

// NewS3 creates a new S3 storage backend
func NewS3(cfg S3Config) (*S3Backend, error) {
	// Create custom resolver for S3-compatible endpoints
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		},
	)

	// Create AWS config with static credentials
	awsCfg := aws.Config{
		Region:                      cfg.Region,
		Credentials:                 credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		EndpointResolverWithOptions: customResolver,
	}

	// Create S3 client with path-style addressing option
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	backend := &S3Backend{
		s3Client: s3Client,
		config:   cfg,
	}

	return backend, nil
}

// EnsureBucket creates the bucket if it doesn't exist
func (b *S3Backend) EnsureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := b.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.config.Bucket),
	})
	if err == nil {
		return nil // Bucket exists
	}

	// Create bucket
	_, err = b.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(b.config.Bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", b.config.Bucket, err)
	}

	return nil
}

// Upload uploads data to S3
func (b *S3Backend) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(b.config.Bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentLength: aws.Int64(size),
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err := b.s3Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload object %s: %w", key, err)
	}

	return nil
}

// Download downloads an object from S3
func (b *S3Backend) Download(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	output, err := b.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download object %s: %w", key, err)
	}

	info := &ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(output.ContentLength),
		ContentType:  aws.ToString(output.ContentType),
		ETag:         aws.ToString(output.ETag),
		LastModified: aws.ToTime(output.LastModified),
	}

	return output.Body, info, nil
}

// Delete deletes an object from S3
func (b *S3Backend) Delete(ctx context.Context, key string) error {
	_, err := b.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}

	return nil
}

// Exists checks if an object exists in S3
func (b *S3Backend) Exists(ctx context.Context, key string) (bool, error) {
	_, err := b.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}
	return true, nil
}

// GetInfo retrieves metadata for an object
func (b *S3Backend) GetInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	output, err := b.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info for %s: %w", key, err)
	}

	return &ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(output.ContentLength),
		ContentType:  aws.ToString(output.ContentType),
		ETag:         aws.ToString(output.ETag),
		LastModified: aws.ToTime(output.LastModified),
	}, nil
}

// List lists objects with the given prefix
func (b *S3Backend) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo

	paginator := s3.NewListObjectsV2Paginator(b.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.config.Bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, ObjectInfo{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				ETag:         aws.ToString(obj.ETag),
				LastModified: aws.ToTime(obj.LastModified),
			})
		}
	}

	return objects, nil
}

// GetPresignedURL generates a presigned URL for downloading an object
func (b *S3Backend) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(b.s3Client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s: %w", key, err)
	}

	return request.URL, nil
}

// GetPresignedUploadURL generates a presigned URL for uploading an object
func (b *S3Backend) GetPresignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(b.s3Client)

	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL for %s: %w", key, err)
	}

	return request.URL, nil
}

// Ping checks if the S3 storage is accessible
func (b *S3Backend) Ping(ctx context.Context) error {
	_, err := b.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("failed to connect to S3 storage: %w", err)
	}
	return nil
}

// Type returns the storage backend type
func (b *S3Backend) Type() string {
	return "s3"
}

// Location returns the S3 endpoint and bucket
func (b *S3Backend) Location() string {
	return fmt.Sprintf("%s/%s", b.config.Endpoint, b.config.Bucket)
}

// Bucket returns the configured bucket name
func (b *S3Backend) Bucket() string {
	return b.config.Bucket
}

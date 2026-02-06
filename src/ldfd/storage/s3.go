package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Provider represents supported S3-compatible storage providers
type S3Provider string

const (
	// S3ProviderGarage is GarageHQ - uses api.{endpoint} for API and {bucket}.{endpoint} for web
	S3ProviderGarage S3Provider = "garage"
	// S3ProviderMinio is MinIO - uses {endpoint} for both API and web (path-style)
	S3ProviderMinio S3Provider = "minio"
	// S3ProviderAWS is Amazon S3 - uses s3.{region}.amazonaws.com for API
	S3ProviderAWS S3Provider = "aws"
	// S3ProviderOther is a generic S3-compatible provider (path-style)
	S3ProviderOther S3Provider = "other"
)

// S3Config holds the S3 storage configuration
type S3Config struct {
	// Provider is the S3 provider type (garage, minio, aws, other)
	Provider S3Provider

	// Endpoint is the base S3 domain (e.g., "s3.example.com")
	// The actual API and web URLs are constructed based on the provider
	Endpoint string

	// Region is the S3 region (e.g., "us-east-1", "emea-west")
	Region string

	// Bucket is the default bucket name for storing artifacts
	Bucket string

	// AccessKeyID is the S3 access key
	AccessKeyID string

	// SecretAccessKey is the S3 secret key
	SecretAccessKey string
}

// GetAPIEndpoint returns the full API endpoint URL based on the provider
func (c *S3Config) GetAPIEndpoint() string {
	endpoint := strings.TrimPrefix(c.Endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	switch c.Provider {
	case S3ProviderGarage:
		// GarageHQ: api.{endpoint}
		return fmt.Sprintf("https://api.%s", endpoint)
	case S3ProviderAWS:
		// AWS: s3.{region}.amazonaws.com
		return fmt.Sprintf("https://s3.%s.amazonaws.com", c.Region)
	case S3ProviderMinio, S3ProviderOther:
		// MinIO/Other: use endpoint directly
		return fmt.Sprintf("https://%s", endpoint)
	default:
		return fmt.Sprintf("https://%s", endpoint)
	}
}

// GetWebEndpoint returns the web endpoint URL for serving artifacts
func (c *S3Config) GetWebEndpoint() string {
	endpoint := strings.TrimPrefix(c.Endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	switch c.Provider {
	case S3ProviderGarage:
		// GarageHQ: {bucket}.{endpoint}
		return fmt.Sprintf("https://%s.%s", c.Bucket, endpoint)
	case S3ProviderAWS:
		// AWS: {bucket}.s3.{region}.amazonaws.com
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", c.Bucket, c.Region)
	case S3ProviderMinio, S3ProviderOther:
		// MinIO/Other: {endpoint}/{bucket} (path-style)
		return fmt.Sprintf("https://%s/%s", endpoint, c.Bucket)
	default:
		return fmt.Sprintf("https://%s/%s", endpoint, c.Bucket)
	}
}

// UsePathStyle returns whether to use path-style addressing for this provider
func (c *S3Config) UsePathStyle() bool {
	switch c.Provider {
	case S3ProviderGarage:
		// GarageHQ uses path-style for API operations
		return true
	case S3ProviderMinio, S3ProviderOther:
		// MinIO and generic providers use path-style
		return true
	case S3ProviderAWS:
		// AWS uses virtual-hosted style by default
		return false
	default:
		return true
	}
}

// S3Backend implements storage using S3-compatible object storage
type S3Backend struct {
	s3Client *s3.Client
	config   S3Config
}

// NewS3 creates a new S3 storage backend
func NewS3(cfg S3Config) (*S3Backend, error) {
	// Get the API endpoint based on provider
	apiEndpoint := cfg.GetAPIEndpoint()

	// For GarageHQ, use "garage" as the region if not AWS
	signingRegion := cfg.Region
	if cfg.Provider == S3ProviderGarage && signingRegion == "" {
		signingRegion = "garage"
	}

	// Create S3 client with custom endpoint configuration
	s3Client := s3.New(s3.Options{
		Region:       signingRegion,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		BaseEndpoint: aws.String(apiEndpoint),
		UsePathStyle: cfg.UsePathStyle(),
	})

	backend := &S3Backend{
		s3Client: s3Client,
		config:   cfg,
	}

	return backend, nil
}

// EnsureBucket checks if the bucket exists and is accessible.
// It does NOT attempt to create the bucket - bucket creation should be done
// through the storage provider's admin interface.
func (b *S3Backend) EnsureBucket(ctx context.Context) error {
	// Check if bucket exists and is accessible
	_, err := b.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.config.Bucket),
	})
	if err != nil {
		return fmt.Errorf("bucket %s is not accessible: %w", b.config.Bucket, err)
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

// Copy copies an object from srcKey to dstKey within the same S3 bucket
func (b *S3Backend) Copy(ctx context.Context, srcKey, dstKey string) error {
	_, err := b.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(b.config.Bucket),
		CopySource: aws.String(b.config.Bucket + "/" + srcKey),
		Key:        aws.String(dstKey),
	})
	if err != nil {
		return fmt.Errorf("failed to copy object %s to %s: %w", srcKey, dstKey, err)
	}
	return nil
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
	return fmt.Sprintf("%s/%s", b.config.GetAPIEndpoint(), b.config.Bucket)
}

// Bucket returns the configured bucket name
func (b *S3Backend) Bucket() string {
	return b.config.Bucket
}

// GetWebURL returns a direct web URL for accessing an artifact via the web gateway.
// The URL format is determined by the provider type.
func (b *S3Backend) GetWebURL(key string) string {
	return fmt.Sprintf("%s/%s", b.config.GetWebEndpoint(), key)
}

// Provider returns the configured S3 provider type
func (b *S3Backend) Provider() S3Provider {
	return b.config.Provider
}

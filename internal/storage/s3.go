package storage

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var _ Store = (*S3Store)(nil)

// S3Store implements Store backed by S3-compatible object storage.
type S3Store struct {
	client   *s3.Client
	presign  *s3.PresignClient
	bucket   string
	endpoint string
}

// NewS3Store creates a new S3-backed Store.
func NewS3Store(endpoint, bucket, accessKey, secretKey string) (*S3Store, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &S3Store{
		client:   client,
		presign:  s3.NewPresignClient(client),
		bucket:   bucket,
		endpoint: endpoint,
	}, nil
}

// EnsureBucket creates the bucket if it does not exist.
func (s *S3Store) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		return nil
	}

	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return err
	}
	return nil
}

// Put uploads an object to S3.
func (s *S3Store) Put(ctx context.Context, key string, reader io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	return err
}

// Get downloads an object from S3.
func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

// Delete removes an object from S3.
func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// Stat returns metadata about an object.
func (s *S3Store) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	info := &ObjectInfo{
		Key: key,
	}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil {
		info.ContentType = *out.ContentType
	}
	return info, nil
}

// List returns object keys matching the prefix.
func (s *S3Store) List(ctx context.Context, prefix string) ([]string, error) {
	out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(out.Contents))
	for _, obj := range out.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}
	return keys, nil
}

// PreSignedURL generates a presigned download URL.
func (s *S3Store) PreSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	out, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

// PreSignedUploadURL generates a presigned upload URL.
func (s *S3Store) PreSignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	out, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

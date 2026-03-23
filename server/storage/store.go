package storage

import (
	"context"
	"io"
	"time"
)

type ObjectInfo struct {
	Key         string
	Size        int64
	ContentType string
}

type Store interface {
	Put(ctx context.Context, key string, reader io.Reader, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (*ObjectInfo, error)
	List(ctx context.Context, prefix string) ([]string, error)
	PreSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	PreSignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

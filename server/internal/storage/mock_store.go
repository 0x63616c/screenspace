package storage

import (
	"context"
	"io"
	"time"
)

// MockStore is a test double for the Store interface.
// Set Fn fields for methods you need in each test scenario.
type MockStore struct {
	PutFn                func(ctx context.Context, key string, reader io.Reader, contentType string) error
	GetFn                func(ctx context.Context, key string) (io.ReadCloser, error)
	DeleteFn             func(ctx context.Context, key string) error
	StatFn               func(ctx context.Context, key string) (*ObjectInfo, error)
	ListFn               func(ctx context.Context, prefix string) ([]string, error)
	PreSignedURLFn       func(ctx context.Context, key string, expiry time.Duration) (string, error)
	PreSignedUploadURLFn func(ctx context.Context, key string, expiry time.Duration) (string, error)
}

func (m *MockStore) Put(ctx context.Context, key string, reader io.Reader, contentType string) error {
	if m.PutFn != nil {
		return m.PutFn(ctx, key, reader, contentType)
	}
	return nil
}

func (m *MockStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, key)
	}
	return nil, nil
}

func (m *MockStore) Delete(ctx context.Context, key string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, key)
	}
	return nil
}

func (m *MockStore) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	if m.StatFn != nil {
		return m.StatFn(ctx, key)
	}
	return nil, nil
}

func (m *MockStore) List(ctx context.Context, prefix string) ([]string, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, prefix)
	}
	return nil, nil
}

func (m *MockStore) PreSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.PreSignedURLFn != nil {
		return m.PreSignedURLFn(ctx, key, expiry)
	}
	return "https://mock-url/" + key, nil
}

func (m *MockStore) PreSignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.PreSignedUploadURLFn != nil {
		return m.PreSignedUploadURLFn(ctx, key, expiry)
	}
	return "https://mock-upload-url/" + key, nil
}

var _ Store = (*MockStore)(nil)

package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *S3Store {
	endpoint := os.Getenv("S3_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:9000"
	}
	store, err := NewS3Store(endpoint, "screenspace-test", "minioadmin", "minioadmin")
	if err != nil {
		t.Skipf("skipping, no S3: %v", err)
	}
	ctx := context.Background()
	if err := store.EnsureBucket(ctx); err != nil {
		t.Skipf("skipping, cannot create bucket: %v", err)
	}
	return store
}

func TestS3Store_PutGetDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := "test/hello.txt"
	content := []byte("hello world")

	err := store.Put(ctx, key, bytes.NewReader(content), "text/plain")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	reader, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", got)
	}

	info, err := store.Stat(ctx, key)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), info.Size)
	}

	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestS3Store_PreSignedURL(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	url, err := store.PreSignedURL(ctx, "test/file.mp4", 15*time.Minute)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestS3Store_PreSignedUploadURL(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	url, err := store.PreSignedUploadURL(ctx, "test/upload.mp4", 15*time.Minute)
	if err != nil {
		t.Fatalf("presign upload: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestS3Store_List(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Put two objects
	store.Put(ctx, "list-test/a.txt", bytes.NewReader([]byte("a")), "text/plain")
	store.Put(ctx, "list-test/b.txt", bytes.NewReader([]byte("b")), "text/plain")
	defer store.Delete(ctx, "list-test/a.txt")
	defer store.Delete(ctx, "list-test/b.txt")

	keys, err := store.List(ctx, "list-test/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestS3Store_EnsureBucket_Idempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	// Call twice - should not error
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
}

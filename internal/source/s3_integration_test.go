//go:build integration
// +build integration

package source

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestS3Source_Live(t *testing.T) {
	if os.Getenv("RUN_S3_TESTS") != "true" {
		t.Skip("Set RUN_S3_TESTS=true to run S3 integration tests")
	}

	ctx := context.Background()

	src := NewS3Source("test-bucket", "",
		WithS3Region("us-east-1"),
		WithS3Endpoint("http://localhost:9000"),
		WithS3Credentials("minioadmin", "minioadmin"))
	results, err := src.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var count int
	for r := range results {
		if r.Err != nil {
			t.Logf("error: %v", r.Err)
			continue
		}
		count++
		t.Logf("found: %s (%d bytes)", r.File.Path, r.File.Size)
	}

	if count == 0 {
		t.Error("expected files, got none")
	}
}

func TestS3Source_PresignedURL(t *testing.T) {
	if os.Getenv("RUN_S3_TESTS") != "true" {
		t.Skip("Set RUN_S3_TESTS=true to run S3 integration tests")
	}

	ctx := context.Background()

	url, err := GetS3PresignedURL(ctx, "test-bucket", "test.txt", 15*time.Minute,
		WithS3Region("us-east-1"),
		WithS3Endpoint("http://localhost:9000"),
		WithS3Credentials("minioadmin", "minioadmin"))
	if err != nil {
		t.Fatalf("Presign failed: %v", err)
	}

	t.Logf("Presigned URL: %s", url)

	if url == "" {
		t.Error("expected non-empty URL")
	}
}

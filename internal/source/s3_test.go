package source

import (
	"errors"
	"testing"
)

func TestParseS3Path(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantBucket string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "bucket only",
			path:       "s3://mybucket",
			wantBucket: "mybucket",
			wantPrefix: "",
			wantErr:    false,
		},
		{
			name:       "bucket with prefix",
			path:       "s3://mybucket/path/to/files",
			wantBucket: "mybucket",
			wantPrefix: "path/to/files",
			wantErr:    false,
		},
		{
			name:       "bucket with trailing slash",
			path:       "s3://mybucket/",
			wantBucket: "mybucket",
			wantPrefix: "",
			wantErr:    false,
		},
		{
			name:       "bucket with prefix trailing slash",
			path:       "s3://mybucket/path/",
			wantBucket: "mybucket",
			wantPrefix: "path/",
			wantErr:    false,
		},
		{
			name:    "empty path",
			path:    "s3://",
			wantErr: true,
		},
		{
			name:       "no s3 prefix treated as bucket",
			path:       "mybucket/path",
			wantBucket: "mybucket",
			wantPrefix: "path",
			wantErr:    false,
		},
		{
			name:    "empty string",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, prefix, err := parseS3Path(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if bucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", bucket, tt.wantBucket)
			}
			if prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tt.wantPrefix)
			}
		})
	}
}

func TestFormatS3Path(t *testing.T) {
	tests := []struct {
		bucket string
		key    string
		want   string
	}{
		{
			bucket: "mybucket",
			key:    "path/to/file.txt",
			want:   "s3://mybucket/path/to/file.txt",
		},
		{
			bucket: "mybucket",
			key:    "",
			want:   "s3://mybucket/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.bucket+"/"+tt.key, func(t *testing.T) {
			got := FormatS3Path(tt.bucket, tt.key)
			if got != tt.want {
				t.Errorf("FormatS3Path() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestS3Source_New(t *testing.T) {
	tests := []struct {
		name   string
		bucket string
		prefix string
		opts   []S3Option
	}{
		{
			name:   "bucket only",
			bucket: "test-bucket",
			prefix: "",
		},
		{
			name:   "bucket with prefix",
			bucket: "test-bucket",
			prefix: "logs/",
			opts:   []S3Option{WithS3Region("us-west-2")},
		},
		{
			name:   "with endpoint",
			bucket: "test-bucket",
			prefix: "",
			opts:   []S3Option{WithS3Endpoint("http://localhost:9000")},
		},
		{
			name:   "with credentials",
			bucket: "test-bucket",
			prefix: "",
			opts:   []S3Option{WithS3Credentials("access", "secret")},
		},
		{
			name:   "with workers",
			bucket: "test-bucket",
			prefix: "",
			opts:   []S3Option{WithS3Workers(16)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewS3Source(tt.bucket, tt.prefix, tt.opts...)
			if src == nil {
				t.Fatal("source is nil")
			}
			if src.bucket != tt.bucket {
				t.Errorf("bucket = %q, want %q", src.bucket, tt.bucket)
			}
			if src.prefix != tt.prefix {
				t.Errorf("prefix = %q, want %q", src.prefix, tt.prefix)
			}
		})
	}
}

func TestS3Source_EmptyBucket(t *testing.T) {
	src := NewS3Source("", "")
	_, err := src.List(nil)
	if !errors.Is(err, ErrS3BucketEmpty) {
		t.Errorf("expected ErrS3BucketEmpty, got %v", err)
	}
}

func TestOpenS3(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantBucket string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "valid path",
			path:       "mybucket/path/to/files",
			wantBucket: "mybucket",
			wantPrefix: "path/to/files",
			wantErr:    false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := openS3(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			s3src, ok := src.(*S3Source)
			if !ok {
				t.Fatal("expected *S3Source")
			}
			if s3src.bucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", s3src.bucket, tt.wantBucket)
			}
			if s3src.prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", s3src.prefix, tt.wantPrefix)
			}
		})
	}
}

package scanner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_New(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "no options uses defaults",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "custom workers",
			opts:    []Option{WithWorkers(16)},
			wantErr: false,
		},
		{
			name:    "zero workers uses default",
			opts:    []Option{WithWorkers(0)},
			wantErr: false,
		},
		{
			name:    "custom excludes",
			opts:    []Option{WithExcludes("foo", "bar")},
			wantErr: false,
		},
		{
			name:    "follow symlinks",
			opts:    []Option{WithFollowLinks(true)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.opts...)
			if s == nil {
				t.Fatal("scanner is nil")
			}
			if s.workers <= 0 {
				t.Error("workers should be positive")
			}
		})
	}
}

func TestScanner_Scan_NoPath(t *testing.T) {
	s := New()
	_, err := s.Scan()
	if !errors.Is(err, ErrNoPath) {
		t.Errorf("Scan() error = %v, want %v", err, ErrNoPath)
	}
}

func TestScanner_Scan_PathNotFound(t *testing.T) {
	s := New(WithPaths("/nonexistent/path/12345"))
	_, err := s.Scan()
	if !errors.Is(err, ErrPathNotFound) {
		t.Errorf("Scan() error = %v, want %v", err, ErrPathNotFound)
	}
}

func TestScanner_Scan_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithPaths(tmpDir), WithWorkers(4))
	results, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	var files []File
	for r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error: %v", r.Err)
			continue
		}
		files = append(files, r.File)
	}

	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
	if len(files) > 0 && files[0].Path != testFile {
		t.Errorf("path = %v, want %v", files[0].Path, testFile)
	}
}

func TestScanner_Scan_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 0; i < 10; i++ {
		testFile := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := New(WithPaths(tmpDir), WithWorkers(4))
	results, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	var files []File
	for r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error: %v", r.Err)
			continue
		}
		files = append(files, r.File)
	}

	if len(files) != 10 {
		t.Errorf("got %d files, want 10", len(files))
	}
}

func TestScanner_Scan_ExcludeDir(t *testing.T) {
	tmpDir := t.TempDir()

	normalFile := filepath.Join(tmpDir, "visible.txt")
	if err := os.WriteFile(normalFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	excludedDir := filepath.Join(tmpDir, "node_modules")
	if err := os.Mkdir(excludedDir, 0755); err != nil {
		t.Fatal(err)
	}
	excludedFile := filepath.Join(excludedDir, "hidden.txt")
	if err := os.WriteFile(excludedFile, []byte("hidden"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithPaths(tmpDir), WithWorkers(4))
	results, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	var files []File
	for r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error: %v", r.Err)
			continue
		}
		files = append(files, r.File)
	}

	if len(files) != 1 {
		t.Errorf("got %d files, want 1 (excluded node_modules)", len(files))
	}
	if len(files) > 0 && files[0].Path != normalFile {
		t.Errorf("path = %v, want %v", files[0].Path, normalFile)
	}
}

func TestScanner_Scan_NestedDirs(t *testing.T) {
	tmpDir := t.TempDir()

	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(nestedDir, "deep.txt")
	if err := os.WriteFile(testFile, []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithPaths(tmpDir), WithWorkers(4))
	results, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	var files []File
	for r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error: %v", r.Err)
			continue
		}
		files = append(files, r.File)
	}

	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestScanner_Scan_MultiplePaths(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	file1 := filepath.Join(tmpDir1, "file1.txt")
	file2 := filepath.Join(tmpDir2, "file2.txt")

	if err := os.WriteFile(file1, []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("two"), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithPaths(tmpDir1, tmpDir2), WithWorkers(4))
	results, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	var files []File
	for r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error: %v", r.Err)
			continue
		}
		files = append(files, r.File)
	}

	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}
}

func TestScanner_File_Fields(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithPaths(tmpDir))
	results, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	for r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error: %v", r.Err)
			continue
		}
		f := r.File

		if f.Path != testFile {
			t.Errorf("Path = %v, want %v", f.Path, testFile)
		}
		if f.Size != int64(len(content)) {
			t.Errorf("Size = %v, want %v", f.Size, len(content))
		}
		if f.ModTime == 0 {
			t.Error("ModTime is zero")
		}
		if f.IsDir {
			t.Error("IsDir should be false for file")
		}
		if f.IsSymlink {
			t.Error("IsSymlink should be false for regular file")
		}
	}
}

func TestScanner_DefaultExcludes(t *testing.T) {
	excludes := []string{".git", ".svn", ".hg", "node_modules", "__pycache__", ".idea", ".vscode", "target", "dist", "build", ".cache", "vendor"}

	for _, pattern := range excludes {
		found := false
		for _, d := range DefaultExcludes {
			if d == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultExcludes missing %s", pattern)
		}
	}
}

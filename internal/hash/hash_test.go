package hash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasher_SHA256(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHasher()
	result, err := h.HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}

	if result.Path != testFile {
		t.Errorf("Path = %v, want %v", result.Path, testFile)
	}
	if result.Algorithm != SHA256 {
		t.Errorf("Algorithm = %v, want %v", result.Algorithm, SHA256)
	}
	if result.Size != int64(len(content)) {
		t.Errorf("Size = %v, want %v", result.Size, len(content))
	}
	if result.Hash == "" {
		t.Error("Hash is empty")
	}

	wantHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if result.Hash != wantHash {
		t.Errorf("Hash = %v, want %v", result.Hash, wantHash)
	}
}

func TestHasher_SHA1(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHasher(WithAlgorithm(SHA1))
	result, err := h.HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}

	if result.Algorithm != SHA1 {
		t.Errorf("Algorithm = %v, want %v", result.Algorithm, SHA1)
	}

	wantHash := "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"
	if result.Hash != wantHash {
		t.Errorf("Hash = %v, want %v", result.Hash, wantHash)
	}
}

func TestHasher_SHA512(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHasher(WithAlgorithm(SHA512))
	result, err := h.HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}

	if result.Algorithm != SHA512 {
		t.Errorf("Algorithm = %v, want %v", result.Algorithm, SHA512)
	}

	if len(result.Hash) != 128 {
		t.Errorf("Hash length = %v, want 128", len(result.Hash))
	}
}

func TestHasher_NonExistentFile(t *testing.T) {
	h := NewHasher()
	_, err := h.HashFile("/nonexistent/file/path")
	if err == nil {
		t.Error("HashFile() should return error for non-existent file")
	}
}

func TestHasher_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.bin")

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}

	size := int64(10 * 1024 * 1024)
	buf := make([]byte, 1024)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	for written := int64(0); written < size; written += int64(len(buf)) {
		if _, err := file.Write(buf); err != nil {
			t.Fatal(err)
		}
	}
	file.Close()

	h := NewHasher(WithBufferSize(65536))
	result, err := h.HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}

	if result.Size != size {
		t.Errorf("Size = %v, want %v", result.Size, size)
	}
}

func TestFindDuplicates(t *testing.T) {
	tmpDir := t.TempDir()

	sameContent := []byte("same content everywhere")
	differentContent := []byte("different")

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")
	file4 := filepath.Join(tmpDir, "unique.txt")

	if err := os.WriteFile(file1, sameContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, sameContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, sameContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file4, differentContent, 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHasher()
	paths := []string{file1, file2, file3, file4}

	var results []Result
	for r := range h.HashFiles(paths) {
		results = append(results, r)
	}

	duplicates := FindDuplicates(results)

	if len(duplicates) != 1 {
		t.Fatalf("FindDuplicates() found %d groups, want 1", len(duplicates))
	}

	group := duplicates[0]
	if len(group.Files) != 3 {
		t.Errorf("Duplicate group has %d files, want 3", len(group.Files))
	}
}

func TestFindDuplicates_NoDuplicates(t *testing.T) {
	tmpDir := t.TempDir()

	var paths []string
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(path, []byte{byte(i)}, 0644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	h := NewHasher()
	var results []Result
	for r := range h.HashFiles(paths) {
		results = append(results, r)
	}

	duplicates := FindDuplicates(results)

	if len(duplicates) != 0 {
		t.Errorf("FindDuplicates() found %d groups, want 0", len(duplicates))
	}
}

func TestHashFiles_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()

	var paths []string
	for i := 0; i < 10; i++ {
		path := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	h := NewHasher()
	results := h.HashFiles(paths)

	count := 0
	for range results {
		count++
	}

	if count != 10 {
		t.Errorf("HashFiles() returned %d results, want 10", count)
	}
}

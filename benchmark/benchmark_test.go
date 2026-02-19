package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ripkitten-co/filehound/internal/matcher"
	"github.com/ripkitten-co/filehound/internal/scanner"
)

func createTestFiles(b *testing.B, count int) string {
	b.Helper()

	tmpDir := b.TempDir()

	for i := 0; i < count; i++ {
		name := filepath.Join(tmpDir, fmt.Sprintf("file%04d.txt", i))
		if err := os.WriteFile(name, []byte("test content"), 0644); err != nil {
			b.Fatal(err)
		}
	}

	return tmpDir
}

func BenchmarkScanner_100Files(b *testing.B) {
	tmpDir := createTestFiles(b, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := scanner.New(scanner.WithPaths(tmpDir), scanner.WithWorkers(4))
		results, _ := s.Scan()
		for range results {
		}
	}
}

func BenchmarkScanner_1000Files(b *testing.B) {
	tmpDir := createTestFiles(b, 1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := scanner.New(scanner.WithPaths(tmpDir), scanner.WithWorkers(8))
		results, _ := s.Scan()
		for range results {
		}
	}
}

func BenchmarkScanner_Parallel1(b *testing.B) {
	tmpDir := createTestFiles(b, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := scanner.New(scanner.WithPaths(tmpDir), scanner.WithWorkers(1))
		results, _ := s.Scan()
		for range results {
		}
	}
}

func BenchmarkScanner_Parallel8(b *testing.B) {
	tmpDir := createTestFiles(b, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := scanner.New(scanner.WithPaths(tmpDir), scanner.WithWorkers(8))
		results, _ := s.Scan()
		for range results {
		}
	}
}

func BenchmarkCalculateEntropy(b *testing.B) {
	data := []byte("xK9#mP2$vL5@nQ8&wR4*aB3%cD6^eF7!gH0+iJ4-kL2=mN5/oP9;qR1:tS8<uV3>wX7?yZ0{aC4}bD6|eF2~gH5")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.CalculateEntropy(data)
	}
}

func BenchmarkCalculateEntropy_1KB(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.CalculateEntropy(data)
	}
}

func BenchmarkCalculateEntropy_4KB(b *testing.B) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.CalculateEntropy(data)
	}
}

func BenchmarkRegexMatcher(b *testing.B) {
	m := matcher.MustRegex("test")
	f := scanner.File{Path: "test.txt"}

	tmpFile, err := os.CreateTemp("", "bench*.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.WriteString("test content")
	tmpFile.Close()
	f.Path = tmpFile.Name()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(f)
	}
}

func BenchmarkExtensionMatcher(b *testing.B) {
	m := matcher.NewExtensionMatcher([]string{".go", ".ts", ".js"})
	f := scanner.File{Path: "/path/to/file.go"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(f)
	}
}

func BenchmarkSizeMatcher(b *testing.B) {
	m := matcher.NewSizeMatcher(">", 1024)
	f := scanner.File{Size: 2048}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(f)
	}
}

func BenchmarkGlobMatcher(b *testing.B) {
	m := matcher.NewGlobMatcher("*.go")
	f := scanner.File{Path: "/path/to/file.go"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(f)
	}
}

func BenchmarkAllMatchers(b *testing.B) {
	all := matcher.All{
		matcher.NewExtensionMatcher([]string{".go"}),
		matcher.NewSizeMatcher(">", 100),
		matcher.NewGlobMatcher("*.go"),
	}
	f := scanner.File{Path: "/path/to/file.go", Size: 200}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		all.Match(f)
	}
}

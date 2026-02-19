package matcher

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ripkitten-co/filehound/internal/scanner"
)

func TestMatcherFunc(t *testing.T) {
	tests := []struct {
		name  string
		mf    MatcherFunc
		input scanner.File
		want  bool
	}{
		{
			name:  "always true",
			mf:    func(f scanner.File) bool { return true },
			input: scanner.File{},
			want:  true,
		},
		{
			name:  "always false",
			mf:    func(f scanner.File) bool { return false },
			input: scanner.File{},
			want:  false,
		},
		{
			name:  "check size",
			mf:    func(f scanner.File) bool { return f.Size > 100 },
			input: scanner.File{Size: 200},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mf.Match(tt.input); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		name     string
		matchers All
		input    scanner.File
		want     bool
	}{
		{
			name:     "empty all returns true",
			matchers: All{},
			input:    scanner.File{},
			want:     true,
		},
		{
			name:     "all true returns true",
			matchers: All{Always(), Always()},
			input:    scanner.File{},
			want:     true,
		},
		{
			name:     "one false returns false",
			matchers: All{Always(), Never()},
			input:    scanner.File{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matchers.Match(tt.input); got != tt.want {
				t.Errorf("All.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAny(t *testing.T) {
	tests := []struct {
		name     string
		matchers Any
		input    scanner.File
		want     bool
	}{
		{
			name:     "empty any returns false",
			matchers: Any{},
			input:    scanner.File{},
			want:     false,
		},
		{
			name:     "one true returns true",
			matchers: Any{Never(), Always()},
			input:    scanner.File{},
			want:     true,
		},
		{
			name:     "all false returns false",
			matchers: Any{Never(), Never()},
			input:    scanner.File{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matchers.Match(tt.input); got != tt.want {
				t.Errorf("Any.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNone(t *testing.T) {
	tests := []struct {
		name     string
		matchers None
		input    scanner.File
		want     bool
	}{
		{
			name:     "empty none returns true",
			matchers: None{},
			input:    scanner.File{},
			want:     true,
		},
		{
			name:     "all false returns true",
			matchers: None{Never(), Never()},
			input:    scanner.File{},
			want:     true,
		},
		{
			name:     "one true returns false",
			matchers: None{Never(), Always()},
			input:    scanner.File{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matchers.Match(tt.input); got != tt.want {
				t.Errorf("None.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegexMatcher(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		content string
		want    bool
		wantErr bool
	}{
		{
			name:    "match simple pattern",
			pattern: "hello",
			content: "hello world",
			want:    true,
		},
		{
			name:    "no match",
			pattern: "goodbye",
			content: "hello world",
			want:    false,
		},
		{
			name:    "case insensitive",
			pattern: "(?i)HELLO",
			content: "hello world",
			want:    true,
		},
		{
			name:    "invalid pattern",
			pattern: "[invalid",
			content: "test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewRegexMatcher(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewRegexMatcher() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			f := scanner.File{Path: tmpFile}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegexPathMatcher(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
		wantErr bool
	}{
		{
			name:    "match extension",
			pattern: `\.go$`,
			path:    "/path/to/file.go",
			want:    true,
		},
		{
			name:    "no match extension",
			pattern: `\.go$`,
			path:    "/path/to/file.txt",
			want:    false,
		},
		{
			name:    "match directory",
			pattern: `node_modules`,
			path:    "/project/node_modules/package/index.js",
			want:    true,
		},
		{
			name:    "invalid pattern",
			pattern: "[invalid",
			path:    "/test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewRegexPathMatcher(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewRegexPathMatcher() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			f := scanner.File{Path: tt.path}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContentMatcher(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		content string
		want    bool
	}{
		{
			name:    "match substring",
			pattern: "secret",
			content: "this is a secret value",
			want:    true,
		},
		{
			name:    "no match",
			pattern: "password",
			content: "this is a secret value",
			want:    false,
		},
		{
			name:    "case sensitive",
			pattern: "Secret",
			content: "this is a secret value",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewContentMatcher(tt.pattern, 0)

			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			f := scanner.File{Path: tmpFile}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		check func(float64) bool
	}{
		{
			name:  "empty data has zero entropy",
			data:  []byte{},
			check: func(e float64) bool { return e == 0 },
		},
		{
			name:  "single byte has zero entropy",
			data:  []byte("aaaa"),
			check: func(e float64) bool { return e == 0 },
		},
		{
			name:  "uniform distribution has max entropy",
			data:  []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			check: func(e float64) bool { return e > 3 && e < 5 },
		},
		{
			name:  "random-looking data has high entropy",
			data:  []byte("xK9#mP2$vL5@nQ8&wR4*"),
			check: func(e float64) bool { return e > 4 },
		},
		{
			name:  "repetitive data has low entropy",
			data:  []byte("aaaaaaaaaaaaaaaaaaaa"),
			check: func(e float64) bool { return e < 1 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := CalculateEntropy(tt.data)
			if !tt.check(entropy) {
				t.Errorf("CalculateEntropy() = %v, failed check", entropy)
			}
		})
	}
}

func TestEntropyMatcher(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		content   string
		want      bool
	}{
		{
			name:      "high entropy content matches",
			threshold: 4.0,
			content:   "xK9#mP2$vL5@nQ8&wR4*",
			want:      true,
		},
		{
			name:      "low entropy content doesn't match",
			threshold: 4.0,
			content:   "aaaaaaaaaaaaaaaaaaaa",
			want:      false,
		},
		{
			name:      "high entropy matches with lower threshold",
			threshold: 3.5,
			content:   "xK9mP2vL5nQ8wR4aB3cD6eF7gH0iJ4kL2mN5oP9",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewEntropyMatcher(WithEntropyThreshold(tt.threshold))

			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			f := scanner.File{Path: tmpFile}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtensionMatcher(t *testing.T) {
	tests := []struct {
		name       string
		extensions []string
		path       string
		want       bool
	}{
		{
			name:       "match single extension",
			extensions: []string{".go"},
			path:       "/path/to/file.go",
			want:       true,
		},
		{
			name:       "match multiple extensions",
			extensions: []string{".go", ".ts", ".js"},
			path:       "/path/to/file.ts",
			want:       true,
		},
		{
			name:       "no match",
			extensions: []string{".go"},
			path:       "/path/to/file.txt",
			want:       false,
		},
		{
			name:       "case insensitive",
			extensions: []string{".GO"},
			path:       "/path/to/file.go",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewExtensionMatcher(tt.extensions)
			f := scanner.File{Path: tt.path}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSizeMatcher(t *testing.T) {
	tests := []struct {
		name  string
		op    string
		size  int64
		input int64
		want  bool
	}{
		{
			name:  "greater than",
			op:    ">",
			size:  100,
			input: 200,
			want:  true,
		},
		{
			name:  "greater than fails",
			op:    ">",
			size:  100,
			input: 50,
			want:  false,
		},
		{
			name:  "less than",
			op:    "<",
			size:  100,
			input: 50,
			want:  true,
		},
		{
			name:  "equal",
			op:    "=",
			size:  100,
			input: 100,
			want:  true,
		},
		{
			name:  "greater or equal",
			op:    ">=",
			size:  100,
			input: 100,
			want:  true,
		},
		{
			name:  "less or equal",
			op:    "<=",
			size:  100,
			input: 100,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSizeMatcher(tt.op, tt.size)
			f := scanner.File{Size: tt.input}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:  "bytes",
			input: "100b",
			want:  100,
		},
		{
			name:  "kilobytes",
			input: "2kb",
			want:  2048,
		},
		{
			name:  "megabytes",
			input: "1mb",
			want:  1024 * 1024,
		},
		{
			name:  "gigabytes",
			input: "1gb",
			want:  1024 * 1024 * 1024,
		},
		{
			name:  "plain number",
			input: "512",
			want:  512,
		},
		{
			name:  "uppercase",
			input: "1MB",
			want:  1024 * 1024,
		},
		{
			name:    "invalid",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobMatcher(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{
			name:    "match all go files",
			pattern: "*.go",
			path:    "/path/to/file.go",
			want:    true,
		},
		{
			name:    "match specific prefix",
			pattern: "test_*",
			path:    "/path/to/test_main.go",
			want:    true,
		},
		{
			name:    "no match",
			pattern: "*.go",
			path:    "/path/to/file.txt",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewGlobMatcher(tt.pattern)
			f := scanner.File{Path: tt.path}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmptyMatcher(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want bool
	}{
		{
			name: "empty file matches",
			size: 0,
			want: true,
		},
		{
			name: "non-empty file doesn't match",
			size: 100,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewEmptyMatcher()
			f := scanner.File{Size: tt.size}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSymlinkMatcher(t *testing.T) {
	tests := []struct {
		name      string
		follow    bool
		isSymlink bool
		want      bool
	}{
		{
			name:      "follow true allows symlinks",
			follow:    true,
			isSymlink: true,
			want:      true,
		},
		{
			name:      "follow false rejects symlinks",
			follow:    false,
			isSymlink: true,
			want:      false,
		},
		{
			name:      "follow false allows regular files",
			follow:    false,
			isSymlink: false,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSymlinkMatcher(tt.follow)
			f := scanner.File{IsSymlink: tt.isSymlink}
			if got := m.Match(f); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func FuzzCalculateEntropy(f *testing.F) {
	seedCorpus := [][]byte{
		{},
		[]byte("a"),
		[]byte("aaaa"),
		[]byte("abcd"),
		[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	}

	for _, seed := range seedCorpus {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		entropy := CalculateEntropy(data)

		if entropy < 0 {
			t.Errorf("entropy should not be negative: %v", entropy)
		}

		if entropy > MaxEntropy {
			t.Errorf("entropy should not exceed MaxEntropy: %v > %v", entropy, MaxEntropy)
		}

		if len(data) == 0 && entropy != 0 {
			t.Errorf("empty data should have zero entropy: %v", entropy)
		}

		if !math.IsNaN(entropy) && math.IsInf(entropy, 0) {
			t.Errorf("entropy should not be infinite: %v", entropy)
		}
	})
}

func FuzzRegexMatcher(f *testing.F) {
	f.Add("test", "test content")
	f.Add("[a-z]+", "hello")
	f.Add("(?i)HELLO", "hello world")

	f.Fuzz(func(t *testing.T, pattern, content string) {
		m, err := NewRegexMatcher(pattern)
		if err != nil {
			return // Invalid pattern, skip
		}

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			return
		}

		f := scanner.File{Path: tmpFile}
		_ = m.Match(f) // Just verify it doesn't panic
	})
}

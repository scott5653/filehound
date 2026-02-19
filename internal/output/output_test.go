package output

import (
	"bytes"
	"testing"

	"github.com/ripkitten-co/filehound/internal/scanner"
)

func TestTableFormatter(t *testing.T) {
	tests := []struct {
		name         string
		files        []scanner.File
		noHeader     bool
		wantContains []string
	}{
		{
			name:         "single file with header",
			files:        []scanner.File{{Path: "/test.txt", Size: 100, ModTime: 1234567890}},
			noHeader:     false,
			wantContains: []string{"PATH", "SIZE", "MODIFIED", "/test.txt"},
		},
		{
			name:         "single file without header",
			files:        []scanner.File{{Path: "/test.txt", Size: 100, ModTime: 1234567890}},
			noHeader:     true,
			wantContains: []string{"/test.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewTableFormatter(&buf, tt.noHeader)

			if err := f.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}

			for _, file := range tt.files {
				if err := f.Write(file); err != nil {
					t.Fatalf("Write() error = %v", err)
				}
			}

			if err := f.End(); err != nil {
				t.Fatalf("End() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantContains {
				if !bytes.Contains([]byte(output), []byte(want)) {
					t.Errorf("output missing %q: %s", want, output)
				}
			}
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf, false)

	if err := f.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	file := scanner.File{Path: "/test.txt", Size: 100, ModTime: 1234567890}
	if err := f.Write(file); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := f.End(); err != nil {
		t.Fatalf("End() error = %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("/test.txt")) {
		t.Errorf("output missing path: %s", output)
	}
}

func TestCSVFormatter(t *testing.T) {
	tests := []struct {
		name      string
		files     []scanner.File
		noHeader  bool
		wantLines int
	}{
		{
			name:      "single file with header",
			files:     []scanner.File{{Path: "/test.txt", Size: 100, ModTime: 1234567890}},
			noHeader:  false,
			wantLines: 2,
		},
		{
			name:      "single file without header",
			files:     []scanner.File{{Path: "/test.txt", Size: 100, ModTime: 1234567890}},
			noHeader:  true,
			wantLines: 1,
		},
		{
			name:      "multiple files",
			files:     []scanner.File{{Path: "/a.txt", Size: 100}, {Path: "/b.txt", Size: 200}},
			noHeader:  false,
			wantLines: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewCSVFormatter(&buf, tt.noHeader)

			if err := f.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}

			for _, file := range tt.files {
				if err := f.Write(file); err != nil {
					t.Fatalf("Write() error = %v", err)
				}
			}

			if err := f.End(); err != nil {
				t.Fatalf("End() error = %v", err)
			}

			output := buf.String()
			lines := 0
			for _, c := range output {
				if c == '\n' {
					lines++
				}
			}

			if lines != tt.wantLines {
				t.Errorf("got %d lines, want %d", lines, tt.wantLines)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0B"},
		{100, "100B"},
		{1024, "1.00KB"},
		{1536, "1.50KB"},
		{1048576, "1.00MB"},
		{1073741824, "1.00GB"},
		{1099511627776, "1.00TB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSize(tt.size)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.size, got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		unix int64
		want string
	}{
		{0, "-"},
		{1234567890, "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatTime(tt.unix)
			if got != tt.want {
				t.Errorf("formatTime(%d) = %q, want %q", tt.unix, got, tt.want)
			}
		})
	}
}

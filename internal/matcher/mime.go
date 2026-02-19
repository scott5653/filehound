package matcher

import (
	"os"
	"strings"

	"github.com/h2non/filetype"
	"github.com/ripkitten-co/filehound/internal/scanner"
)

type MIMEMatcher struct {
	mimeTypes map[string]bool
	bufSize   int
}

type MIMEOption func(*MIMEMatcher)

func WithMIMEBufferSize(size int) MIMEOption {
	return func(m *MIMEMatcher) {
		if size > 0 {
			m.bufSize = size
		}
	}
}

func NewMIMEMatcher(mimeTypes []string, opts ...MIMEOption) *MIMEMatcher {
	m := &MIMEMatcher{
		mimeTypes: make(map[string]bool),
		bufSize:   512,
	}
	for _, mt := range mimeTypes {
		m.mimeTypes[strings.ToLower(strings.TrimSpace(mt))] = true
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *MIMEMatcher) Match(f scanner.File) bool {
	file, err := os.Open(f.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, m.bufSize)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	kind, _ := filetype.Match(buf[:n])
	if kind == filetype.Unknown {
		return false
	}

	mime := kind.MIME.Value
	if mime == "" {
		return false
	}

	for mt := range m.mimeTypes {
		if strings.HasPrefix(mime, mt) || mime == mt {
			return true
		}
		if strings.HasPrefix(mt, mime) {
			return true
		}
	}

	return false
}

type ExtensionMatcher struct {
	extensions map[string]bool
}

func NewExtensionMatcher(extensions []string) *ExtensionMatcher {
	e := &ExtensionMatcher{
		extensions: make(map[string]bool),
	}
	for _, ext := range extensions {
		e.extensions[strings.ToLower(strings.TrimSpace(ext))] = true
	}
	return e
}

func (e *ExtensionMatcher) Match(f scanner.File) bool {
	for ext := range e.extensions {
		if strings.HasSuffix(strings.ToLower(f.Path), ext) {
			return true
		}
	}
	return false
}

type FileTypeMatcher struct {
	types map[string]bool
}

func NewFileTypeMatcher(types []string) *FileTypeMatcher {
	t := &FileTypeMatcher{
		types: make(map[string]bool),
	}
	for _, ft := range types {
		t.types[strings.ToLower(strings.TrimSpace(ft))] = true
	}
	return t
}

func (t *FileTypeMatcher) Match(f scanner.File) bool {
	file, err := os.Open(f.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	kind, _ := filetype.Match(buf[:n])
	if kind == filetype.Unknown {
		return false
	}

	for ft := range t.types {
		if strings.ToLower(kind.Extension) == ft {
			return true
		}
	}

	return false
}

func IsBinary(f scanner.File) bool {
	file, err := os.Open(f.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}

package matcher

import (
	"bytes"
	"io"
	"os"
	"regexp"

	"github.com/ripkitten-co/filehound/internal/scanner"
)

type RegexMatcher struct {
	pattern   *regexp.Regexp
	bufSize   int
	matchPath bool
}

type RegexOption func(*RegexMatcher)

func WithBufferSize(size int) RegexOption {
	return func(r *RegexMatcher) {
		if size > 0 {
			r.bufSize = size
		}
	}
}

func WithMatchPath(match bool) RegexOption {
	return func(r *RegexMatcher) {
		r.matchPath = match
	}
}

func NewRegexMatcher(pattern string, opts ...RegexOption) (*RegexMatcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	r := &RegexMatcher{
		pattern: re,
		bufSize: 4096,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

func MustRegex(pattern string, opts ...RegexOption) *RegexMatcher {
	r, err := NewRegexMatcher(pattern, opts...)
	if err != nil {
		panic(err)
	}
	return r
}

func (r *RegexMatcher) Match(f scanner.File) bool {
	if r.matchPath {
		return r.pattern.MatchString(f.Path)
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, r.bufSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	return r.pattern.Match(buf[:n])
}

func (r *RegexMatcher) MatchBytes(data []byte) bool {
	return r.pattern.Match(data)
}

func (r *RegexMatcher) MatchString(s string) bool {
	return r.pattern.MatchString(s)
}

type RegexPathMatcher struct {
	pattern *regexp.Regexp
}

func NewRegexPathMatcher(pattern string) (*RegexPathMatcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexPathMatcher{pattern: re}, nil
}

func (r *RegexPathMatcher) Match(f scanner.File) bool {
	return r.pattern.MatchString(f.Path)
}

type ContentMatcher struct {
	pattern []byte
	bufSize int
}

func NewContentMatcher(pattern string, bufSize int) *ContentMatcher {
	if bufSize <= 0 {
		bufSize = 4096
	}
	return &ContentMatcher{
		pattern: []byte(pattern),
		bufSize: bufSize,
	}
}

func (c *ContentMatcher) Match(f scanner.File) bool {
	file, err := os.Open(f.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, c.bufSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	return bytes.Contains(buf[:n], c.pattern)
}

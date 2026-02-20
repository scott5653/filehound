package source

import (
	"context"
	"io/fs"
)

type File struct {
	Path      string
	Size      int64
	ModTime   int64
	Mode      fs.FileMode
	IsDir     bool
	IsSymlink bool
	Source    string
	GitCommit string
	GitAuthor string
	GitBranch string
}

type Result struct {
	File File
	Err  error
}

type Source interface {
	List(ctx context.Context) (<-chan Result, error)
	Read(ctx context.Context, path string) ([]byte, error)
	Close() error
}

type Opener func(path string) (Source, error)

var openers = make(map[string]Opener)

func Register(scheme string, opener Opener) {
	openers[scheme] = opener
}

func Open(path string) (Source, error) {
	scheme, rest := parseScheme(path)
	if scheme == "" {
		scheme = "file"
	}

	opener, ok := openers[scheme]
	if !ok {
		return nil, ErrUnsupportedSource
	}

	return opener(rest)
}

func parseScheme(path string) (scheme, rest string) {
	for i := 0; i < len(path); i++ {
		if path[i] == ':' {
			if i+1 < len(path) && path[i+1] == '/' && i+2 < len(path) && path[i+2] == '/' {
				return path[:i], path[i+3:]
			}
			if i+1 < len(path) && path[i+1] == '/' {
				return path[:i], path[i+1:]
			}
		}
	}
	return "", path
}

var (
	ErrUnsupportedSource = &SourceError{Msg: "unsupported source scheme"}
)

type SourceError struct {
	Msg string
	Err error
}

func (e *SourceError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

func (e *SourceError) Unwrap() error {
	return e.Err
}

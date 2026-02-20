package source

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrNoPath       = errors.New("no path specified")
	ErrPathNotFound = errors.New("path not found")
)

var DefaultExcludes = []string{
	".git",
	".svn",
	".hg",
	"node_modules",
	"__pycache__",
	".idea",
	".vscode",
	"target",
	"dist",
	"build",
	".cache",
	"vendor",
}

type LocalSource struct {
	paths       []string
	excludes    []string
	workers     int
	followLinks bool
}

type LocalOption func(*LocalSource)

func WithPaths(paths ...string) LocalOption {
	return func(s *LocalSource) {
		s.paths = append(s.paths, paths...)
	}
}

func WithExcludes(patterns ...string) LocalOption {
	return func(s *LocalSource) {
		s.excludes = append(s.excludes, patterns...)
	}
}

func WithWorkers(n int) LocalOption {
	return func(s *LocalSource) {
		if n > 0 {
			s.workers = n
		}
	}
}

func WithFollowLinks(follow bool) LocalOption {
	return func(s *LocalSource) {
		s.followLinks = follow
	}
}

func NewLocalSource(opts ...LocalOption) *LocalSource {
	s := &LocalSource{
		workers:     8,
		excludes:    DefaultExcludes,
		followLinks: false,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *LocalSource) List(ctx context.Context) (<-chan Result, error) {
	if len(s.paths) == 0 {
		return nil, ErrNoPath
	}

	for _, p := range s.paths {
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				return nil, ErrPathNotFound
			}
			return nil, err
		}
	}

	results := make(chan Result, s.workers*10)
	go s.walk(ctx, results)

	return results, nil
}

func (s *LocalSource) Read(ctx context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (s *LocalSource) Close() error {
	return nil
}

func (s *LocalSource) walk(ctx context.Context, results chan<- Result) {
	var wg sync.WaitGroup
	wg.Add(len(s.paths))

	for _, path := range s.paths {
		go func(p string) {
			defer wg.Done()
			s.walkPath(ctx, p, results)
		}(path)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

func (s *LocalSource) walkPath(ctx context.Context, root string, results chan<- Result) {
	sem := make(chan struct{}, s.workers)
	var wg sync.WaitGroup

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			select {
			case results <- Result{Err: err}:
			default:
			}
			return nil
		}

		if d.IsDir() && s.isExcluded(path, d.Name()) {
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			info, err := d.Info()
			if err != nil {
				results <- Result{Err: err}
				return
			}

			f := File{
				Path:   path,
				Size:   info.Size(),
				Mode:   info.Mode(),
				IsDir:  d.IsDir(),
				Source: "local",
			}

			if mt := info.ModTime(); !mt.IsZero() {
				f.ModTime = mt.Unix()
			}

			f.IsSymlink = info.Mode()&fs.ModeSymlink != 0

			results <- Result{File: f}
		}()

		return nil
	})

	wg.Wait()
}

func (s *LocalSource) isExcluded(path, name string) bool {
	for _, pattern := range s.excludes {
		if name == pattern {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		if strings.Contains(path, string(filepath.Separator)+pattern+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func init() {
	Register("file", func(path string) (Source, error) {
		if path == "" {
			path = "."
		}
		return NewLocalSource(WithPaths(path)), nil
	})
}

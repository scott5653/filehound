package scanner

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

var (
	ErrNoPath       = errors.New("no path specified")
	ErrPathNotFound = errors.New("path not found")
)

type File struct {
	Path      string
	Size      int64
	ModTime   int64
	Mode      fs.FileMode
	IsDir     bool
	IsSymlink bool
}

type Result struct {
	File File
	Err  error
}

type Scanner struct {
	paths       []string
	excludes    []string
	workers     int
	followLinks bool
}

type Option func(*Scanner)

func WithPaths(paths ...string) Option {
	return func(s *Scanner) {
		s.paths = append(s.paths, paths...)
	}
}

func WithExcludes(patterns ...string) Option {
	return func(s *Scanner) {
		s.excludes = append(s.excludes, patterns...)
	}
}

func WithWorkers(n int) Option {
	return func(s *Scanner) {
		if n > 0 {
			s.workers = n
		}
	}
}

func WithFollowLinks(follow bool) Option {
	return func(s *Scanner) {
		s.followLinks = follow
	}
}

func New(opts ...Option) *Scanner {
	s := &Scanner{
		workers:     8,
		excludes:    DefaultExcludes,
		followLinks: false,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

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

func (s *Scanner) Scan() (<-chan Result, error) {
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
	go s.walk(results)

	return results, nil
}

func (s *Scanner) walk(results chan<- Result) {
	var wg sync.WaitGroup
	wg.Add(len(s.paths))

	for _, path := range s.paths {
		go func(p string) {
			defer wg.Done()
			s.walkPath(p, results)
		}(path)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

func (s *Scanner) walkPath(root string, results chan<- Result) {
	sem := make(chan struct{}, s.workers)
	var wg sync.WaitGroup

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
				Path:  path,
				Size:  info.Size(),
				Mode:  info.Mode(),
				IsDir: d.IsDir(),
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

func (s *Scanner) isExcluded(path, name string) bool {
	for _, pattern := range s.excludes {
		if name == pattern {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

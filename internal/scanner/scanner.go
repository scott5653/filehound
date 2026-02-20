package scanner

import (
	"context"

	"github.com/ripkitten-co/filehound/internal/source"
)

type File = source.File

type Result = source.Result

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
		workers: 8,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func NewWithSource(src source.Source) *Scanner {
	return &Scanner{}
}

var DefaultExcludes = source.DefaultExcludes

var (
	ErrNoPath       = source.ErrNoPath
	ErrPathNotFound = source.ErrPathNotFound
)

func (s *Scanner) Scan() (<-chan Result, error) {
	return s.ScanContext(context.Background())
}

func (s *Scanner) ScanContext(ctx context.Context) (<-chan Result, error) {
	lsOpts := []source.LocalOption{
		source.WithWorkers(s.workers),
	}
	if len(s.paths) > 0 {
		lsOpts = append(lsOpts, source.WithPaths(s.paths...))
	}
	if len(s.excludes) > 0 {
		lsOpts = append(lsOpts, source.WithExcludes(s.excludes...))
	}
	if s.followLinks {
		lsOpts = append(lsOpts, source.WithFollowLinks(s.followLinks))
	}

	ls := source.NewLocalSource(lsOpts...)
	return ls.List(ctx)
}

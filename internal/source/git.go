package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	ErrInvalidGitPath = errors.New("invalid git path")
	ErrGitNotFound    = errors.New("not a git repository")
)

type GitMode string

const (
	GitModeWorking GitMode = "working"
	GitModeFull    GitMode = "full"
)

type GitSource struct {
	repoPath string
	mode     GitMode
	branch   string
	since    time.Time
	workers  int
}

type GitOption func(*GitSource)

func WithGitMode(mode GitMode) GitOption {
	return func(s *GitSource) {
		s.mode = mode
	}
}

func WithGitBranch(branch string) GitOption {
	return func(s *GitSource) {
		s.branch = branch
	}
}

func WithGitSince(since time.Time) GitOption {
	return func(s *GitSource) {
		s.since = since
	}
}

func WithGitWorkers(n int) GitOption {
	return func(s *GitSource) {
		if n > 0 {
			s.workers = n
		}
	}
}

func NewGitSource(repoPath string, opts ...GitOption) *GitSource {
	s := &GitSource{
		repoPath: repoPath,
		mode:     GitModeWorking,
		workers:  8,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *GitSource) List(ctx context.Context) (<-chan Result, error) {
	results := make(chan Result, s.workers*10)

	repo, err := git.PlainOpen(s.repoPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGitNotFound, err)
	}

	if s.mode == GitModeWorking {
		go s.listWorkingTree(ctx, repo, results)
	} else {
		go s.listFullHistory(ctx, repo, results)
	}

	return results, nil
}

func (s *GitSource) listWorkingTree(ctx context.Context, repo *git.Repository, results chan<- Result) {
	defer close(results)

	wt, err := repo.Worktree()
	if err != nil {
		results <- Result{Err: err}
		return
	}

	fs := wt.Filesystem

	err = walkFilesystem(fs, "", func(path string, info os.FileInfo) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		f := File{
			Path:      filepath.Join(s.repoPath, path),
			Size:      info.Size(),
			ModTime:   info.ModTime().Unix(),
			Source:    "git",
			GitBranch: "working",
		}

		results <- Result{File: f}
		return nil
	})

	if err != nil {
		results <- Result{Err: err}
	}
}

func walkFilesystem(fs billy.Filesystem, base string, fn func(string, os.FileInfo) error) error {
	entries, err := fs.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(base, entry.Name())

		info, err := fs.Stat(fullPath)
		if err != nil {
			continue
		}

		if err := fn(fullPath, info); err != nil {
			return err
		}

		if entry.IsDir() {
			if err := walkFilesystem(fs, fullPath, fn); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *GitSource) listFullHistory(ctx context.Context, repo *git.Repository, results chan<- Result) {
	defer close(results)

	branches, err := repo.Branches()
	if err != nil {
		results <- Result{Err: err}
		return
	}

	var branchNames []string
	branches.ForEach(func(ref *plumbing.Reference) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		branchNames = append(branchNames, ref.Name().Short())
		return nil
	})

	if s.branch != "" {
		branchNames = []string{s.branch}
	}

	for _, branchName := range branchNames {
		ref, err := repo.Reference(plumbing.NewBranchReferenceName(branchName), true)
		if err != nil {
			results <- Result{Err: fmt.Errorf("branch not found: %s", branchName)}
			continue
		}

		commitIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			results <- Result{Err: err}
			continue
		}

		err = commitIter.ForEach(func(commit *object.Commit) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if !s.since.IsZero() && commit.Author.When.Before(s.since) {
				return io.EOF
			}

			tree, err := commit.Tree()
			if err != nil {
				return nil
			}

			tree.Files().ForEach(func(f *object.File) error {
				select {
				case <-ctx.Done():
					return io.EOF
				default:
				}

				entry := File{
					Path:      f.Name,
					Size:      f.Size,
					ModTime:   commit.Author.When.Unix(),
					Source:    "git",
					GitCommit: commit.Hash.String()[:8],
					GitAuthor: commit.Author.Name,
					GitBranch: branchName,
				}

				results <- Result{File: entry}
				return nil
			})

			return nil
		})

		if err != nil && err != io.EOF {
			results <- Result{Err: err}
		}
	}
}

func (s *GitSource) Read(ctx context.Context, path string) ([]byte, error) {
	repo, err := git.PlainOpen(s.repoPath)
	if err != nil {
		return nil, err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	relPath, err := filepath.Rel(s.repoPath, path)
	if err != nil {
		return nil, err
	}

	file, err := wt.Filesystem.Open(relPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

func (s *GitSource) Close() error {
	return nil
}

func parseGitPath(path string) (repoPath string, err error) {
	path = strings.TrimPrefix(path, "git://")
	path = strings.TrimPrefix(path, "file://")

	if path == "" {
		return "", ErrInvalidGitPath
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(filepath.Join(absPath, ".git")); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", ErrGitNotFound, absPath)
	}

	return absPath, nil
}

func openGit(path string) (Source, error) {
	repoPath, err := parseGitPath(path)
	if err != nil {
		return nil, err
	}
	return NewGitSource(repoPath), nil
}

func init() {
	Register("git", openGit)
}

func IsGitRepo(path string) bool {
	_, err := parseGitPath(path)
	return err == nil
}

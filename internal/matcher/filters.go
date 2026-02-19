package matcher

import (
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ripkitten-co/filehound/internal/scanner"
)

type SizeMatcher struct {
	min int64
	max int64
}

func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, nil
	}

	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"tb", 1024 * 1024 * 1024 * 1024},
		{"gb", 1024 * 1024 * 1024},
		{"mb", 1024 * 1024},
		{"kb", 1024},
		{"b", 1},
	}

	for _, sm := range suffixes {
		if strings.HasSuffix(s, sm.suffix) {
			numStr := strings.TrimSuffix(s, sm.suffix)
			num, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, err
			}
			return num * sm.mult, nil
		}
	}

	return strconv.ParseInt(s, 10, 64)
}

func NewSizeMatcher(op string, size int64) *SizeMatcher {
	s := &SizeMatcher{}
	switch {
	case strings.HasPrefix(op, ">="):
		s.min = size
		s.max = -1
	case strings.HasPrefix(op, "<="):
		s.min = 0
		s.max = size
	case strings.HasPrefix(op, ">"):
		s.min = size + 1
		s.max = -1
	case strings.HasPrefix(op, "<"):
		s.min = 0
		s.max = size - 1
	case strings.HasPrefix(op, "==") || op == "=":
		s.min = size
		s.max = size
	default:
		s.min = size
		s.max = -1
	}
	return s
}

func NewSizeRangeMatcher(min, max int64) *SizeMatcher {
	return &SizeMatcher{min: min, max: max}
}

func (s *SizeMatcher) Match(f scanner.File) bool {
	if s.min > 0 && f.Size < s.min {
		return false
	}
	if s.max > 0 && f.Size > s.max {
		return false
	}
	return true
}

type TimeMatcher struct {
	before int64
	after  int64
}

func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	multipliers := map[string]time.Duration{
		"s":  time.Second,
		"m":  time.Minute,
		"h":  time.Hour,
		"d":  24 * time.Hour,
		"w":  7 * 24 * time.Hour,
		"mo": 30 * 24 * time.Hour,
		"y":  365 * 24 * time.Hour,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			num, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, err
			}
			return time.Duration(num) * mult, nil
		}
	}

	return time.ParseDuration(s)
}

func NewModifiedMatcher(op string, duration time.Duration) *TimeMatcher {
	now := time.Now()
	t := &TimeMatcher{}

	switch {
	case strings.HasPrefix(op, "<"):
		t.before = now.Add(-duration).Unix()
	case strings.HasPrefix(op, ">"):
		t.after = now.Add(-duration).Unix()
	}
	return t
}

func NewTimeRangeMatcher(before, after time.Time) *TimeMatcher {
	return &TimeMatcher{
		before: before.Unix(),
		after:  after.Unix(),
	}
}

func (t *TimeMatcher) Match(f scanner.File) bool {
	if t.before > 0 && f.ModTime > t.before {
		return false
	}
	if t.after > 0 && f.ModTime < t.after {
		return false
	}
	return true
}

type OwnerMatcher struct {
	uid int
	gid int
}

func ParseOwner(owner string) (int, int, error) {
	parts := strings.Split(owner, ":")
	if len(parts) == 0 {
		return -1, -1, nil
	}

	uid, err := strconv.Atoi(parts[0])
	if err != nil {
		if u, err := user.Lookup(parts[0]); err == nil {
			uid, _ = strconv.Atoi(u.Uid)
		}
	}

	gid := -1
	if len(parts) > 1 {
		gid, err = strconv.Atoi(parts[1])
		if err != nil {
			if g, err := user.LookupGroup(parts[1]); err == nil {
				gid, _ = strconv.Atoi(g.Gid)
			}
		}
	}

	return uid, gid, nil
}

func NewOwnerMatcher(uid, gid int) *OwnerMatcher {
	return &OwnerMatcher{uid: uid, gid: gid}
}

func (o *OwnerMatcher) Match(f scanner.File) bool {
	return getOwner(f.Path, o.uid, o.gid)
}

func getOwner(path string, wantUID, wantGID int) bool {
	return true
}

type GlobMatcher struct {
	pattern string
}

func NewGlobMatcher(pattern string) *GlobMatcher {
	return &GlobMatcher{pattern: pattern}
}

func (g *GlobMatcher) Match(f scanner.File) bool {
	matched, err := filepath.Match(g.pattern, filepath.Base(f.Path))
	if err != nil {
		return false
	}
	return matched
}

type PathMatcher struct {
	pattern string
}

func NewPathMatcher(pattern string) *PathMatcher {
	return &PathMatcher{pattern: pattern}
}

func (p *PathMatcher) Match(f scanner.File) bool {
	matched, err := filepath.Match(p.pattern, f.Path)
	if err != nil {
		return false
	}
	return matched
}

type SymlinkMatcher struct {
	follow bool
}

func NewSymlinkMatcher(follow bool) *SymlinkMatcher {
	return &SymlinkMatcher{follow: follow}
}

func (s *SymlinkMatcher) Match(f scanner.File) bool {
	if s.follow {
		return true
	}
	return !f.IsSymlink
}

type EmptyMatcher struct{}

func NewEmptyMatcher() *EmptyMatcher {
	return &EmptyMatcher{}
}

func (e *EmptyMatcher) Match(f scanner.File) bool {
	return f.Size == 0
}

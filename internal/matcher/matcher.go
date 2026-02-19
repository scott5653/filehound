package matcher

import (
	"github.com/ripkitten-co/filehound/internal/scanner"
)

type Matcher interface {
	Match(f scanner.File) bool
}

type MatcherFunc func(f scanner.File) bool

func (mf MatcherFunc) Match(f scanner.File) bool {
	return mf(f)
}

type All []Matcher

func (a All) Match(f scanner.File) bool {
	for _, m := range a {
		if !m.Match(f) {
			return false
		}
	}
	return true
}

type Any []Matcher

func (a Any) Match(f scanner.File) bool {
	for _, m := range a {
		if m.Match(f) {
			return true
		}
	}
	return false
}

type None []Matcher

func (n None) Match(f scanner.File) bool {
	for _, m := range n {
		if m.Match(f) {
			return false
		}
	}
	return true
}

func Always() Matcher {
	return MatcherFunc(func(f scanner.File) bool {
		return true
	})
}

func Never() Matcher {
	return MatcherFunc(func(f scanner.File) bool {
		return false
	})
}

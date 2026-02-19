package matcher

import (
	"math"
	"os"

	"github.com/ripkitten-co/filehound/internal/scanner"
)

const (
	DefaultEntropyThreshold = 7.5
	MaxEntropy              = 8.0
)

type EntropyMatcher struct {
	threshold float64
	bufSize   int
}

type EntropyOption func(*EntropyMatcher)

func WithEntropyThreshold(threshold float64) EntropyOption {
	return func(e *EntropyMatcher) {
		if threshold > 0 && threshold <= MaxEntropy {
			e.threshold = threshold
		}
	}
}

func WithEntropyBufferSize(size int) EntropyOption {
	return func(e *EntropyMatcher) {
		if size > 0 {
			e.bufSize = size
		}
	}
}

func NewEntropyMatcher(opts ...EntropyOption) *EntropyMatcher {
	e := &EntropyMatcher{
		threshold: DefaultEntropyThreshold,
		bufSize:   4096,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *EntropyMatcher) Match(f scanner.File) bool {
	file, err := os.Open(f.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, e.bufSize)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	entropy := CalculateEntropy(buf[:n])
	return entropy >= e.threshold
}

func CalculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	freq := make(map[byte]int, 256)
	for _, b := range data {
		freq[b]++
	}

	var entropy float64
	dataLen := float64(len(data))

	for _, count := range freq {
		if count == 0 {
			continue
		}
		p := float64(count) / dataLen
		entropy -= p * math.Log2(p)
	}

	return entropy
}

type EntropyRangeMatcher struct {
	minThreshold float64
	maxThreshold float64
	bufSize      int
}

func NewEntropyRangeMatcher(min, max float64, bufSize int) *EntropyRangeMatcher {
	if bufSize <= 0 {
		bufSize = 4096
	}
	return &EntropyRangeMatcher{
		minThreshold: min,
		maxThreshold: max,
		bufSize:      bufSize,
	}
}

func (e *EntropyRangeMatcher) Match(f scanner.File) bool {
	file, err := os.Open(f.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, e.bufSize)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	entropy := CalculateEntropy(buf[:n])
	return entropy >= e.minThreshold && entropy <= e.maxThreshold
}

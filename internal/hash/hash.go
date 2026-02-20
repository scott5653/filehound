package hash

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

type Algorithm string

const (
	SHA1   Algorithm = "sha1"
	SHA256 Algorithm = "sha256"
	SHA512 Algorithm = "sha512"
)

type Result struct {
	Path      string
	Hash      string
	Algorithm Algorithm
	Size      int64
}

type Hasher struct {
	algorithm  Algorithm
	bufferSize int
}

type Option func(*Hasher)

func WithAlgorithm(alg Algorithm) Option {
	return func(h *Hasher) {
		h.algorithm = alg
	}
}

func WithBufferSize(size int) Option {
	return func(h *Hasher) {
		if size > 0 {
			h.bufferSize = size
		}
	}
}

func NewHasher(opts ...Option) *Hasher {
	h := &Hasher{
		algorithm:  SHA256,
		bufferSize: 65536,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *Hasher) HashFile(path string) (*Result, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	var hasher hash.Hash
	switch h.algorithm {
	case SHA1:
		hasher = sha1.New()
	case SHA256:
		hasher = sha256.New()
	case SHA512:
		hasher = sha512.New()
	default:
		hasher = sha256.New()
	}

	buf := make([]byte, h.bufferSize)
	if _, err := io.CopyBuffer(hasher, file, buf); err != nil {
		return nil, err
	}

	return &Result{
		Path:      path,
		Hash:      hex.EncodeToString(hasher.Sum(nil)),
		Algorithm: h.algorithm,
		Size:      stat.Size(),
	}, nil
}

func (h *Hasher) HashFiles(paths []string) <-chan Result {
	results := make(chan Result, len(paths))

	go func() {
		defer close(results)
		for _, path := range paths {
			result, err := h.HashFile(path)
			if err != nil {
				continue
			}
			results <- *result
		}
	}()

	return results
}

type DuplicateGroup struct {
	Hash  string
	Size  int64
	Files []string
}

func FindDuplicates(results []Result) []DuplicateGroup {
	hashMap := make(map[string]*DuplicateGroup)

	for _, r := range results {
		if _, exists := hashMap[r.Hash]; !exists {
			hashMap[r.Hash] = &DuplicateGroup{
				Hash:  r.Hash,
				Size:  r.Size,
				Files: []string{r.Path},
			}
		} else {
			hashMap[r.Hash].Files = append(hashMap[r.Hash].Files, r.Path)
		}
	}

	var duplicates []DuplicateGroup
	for _, group := range hashMap {
		if len(group.Files) > 1 {
			duplicates = append(duplicates, *group)
		}
	}

	return duplicates
}

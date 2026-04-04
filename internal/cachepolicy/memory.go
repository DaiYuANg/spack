// Package cachepolicy centralizes runtime cache policy decisions.
package cachepolicy

import "github.com/daiyuang/spack/internal/config"

// MemoryPolicy decides when an asset should be served from the in-memory body cache.
type MemoryPolicy interface {
	ShouldServe(size int64, rangeRequested bool) bool
}

// StaticMemoryPolicy applies size and range-request admission rules from static config.
type StaticMemoryPolicy struct {
	maxFileSize int64
}

// NewMemoryPolicy builds a memory-cache admission policy from HTTP config.
func NewMemoryPolicy(cfg config.MemoryCache) MemoryPolicy {
	return StaticMemoryPolicy{maxFileSize: cfg.MaxFileSize}
}

func (p StaticMemoryPolicy) ShouldServe(size int64, rangeRequested bool) bool {
	if rangeRequested {
		return false
	}
	return size >= 0 && size <= p.maxFileSize
}

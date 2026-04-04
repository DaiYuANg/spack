package cachepolicy

import (
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/config"
)

// ArtifactPolicy decides artifact retention and eviction for generated variants.
type ArtifactPolicy interface {
	Enabled() bool
	DefaultMaxAge() time.Duration
	MaxAge(namespace string) time.Duration
	MaxCacheBytes() int64
	ShouldRemoveExpired(namespace string, lastUsed, now time.Time) bool
}

// CompressionArtifactPolicy applies compression/image artifact retention from config.
type CompressionArtifactPolicy struct {
	defaultMaxAge   time.Duration
	namespaceMaxAge collectionx.Map[string, time.Duration]
	maxCacheBytes   int64
}

// NewArtifactPolicy builds an artifact retention policy from compression config.
func NewArtifactPolicy(cfg *config.Compression) ArtifactPolicy {
	if cfg == nil {
		return CompressionArtifactPolicy{
			namespaceMaxAge: emptyNamespaceMaxAges(),
		}
	}

	return CompressionArtifactPolicy{
		defaultMaxAge:   cfg.ParsedMaxAge(),
		namespaceMaxAge: cfg.NamespaceMaxAges(),
		maxCacheBytes:   cfg.MaxCacheBytes,
	}
}

func (p CompressionArtifactPolicy) Enabled() bool {
	return p.defaultMaxAge > 0 || p.namespaceMaxAge.Len() > 0 || p.maxCacheBytes > 0
}

func (p CompressionArtifactPolicy) DefaultMaxAge() time.Duration {
	return p.defaultMaxAge
}

func (p CompressionArtifactPolicy) MaxAge(namespace string) time.Duration {
	return resolveNamespaceMaxAge(p.namespaceMaxAge, namespace, p.defaultMaxAge)
}

func (p CompressionArtifactPolicy) MaxCacheBytes() int64 {
	return p.maxCacheBytes
}

func (p CompressionArtifactPolicy) ShouldRemoveExpired(namespace string, lastUsed, now time.Time) bool {
	maxAge := p.MaxAge(namespace)
	return maxAge > 0 && now.Sub(lastUsed) > maxAge
}

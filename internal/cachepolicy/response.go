package cachepolicy

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
)

// RevalidateCacheControl is the default response policy for original assets.
const RevalidateCacheControl = "public, max-age=0, must-revalidate"

// ResponsePolicy decides the cache headers emitted for resolved assets.
type ResponsePolicy interface {
	CacheControl(result *resolver.Result) string
	ExpiresAt(cacheControl string, lastModified time.Time, hasLastModified bool) (time.Time, bool)
}

// StaticResponsePolicy applies cache-header rules derived from compression config.
type StaticResponsePolicy struct {
	defaultMaxAge   time.Duration
	namespaceMaxAge collectionx.Map[string, time.Duration]
}

// NewResponsePolicy builds a response cache policy from compression config.
func NewResponsePolicy(cfg *config.Compression) ResponsePolicy {
	if cfg == nil {
		return StaticResponsePolicy{namespaceMaxAge: emptyNamespaceMaxAges()}
	}
	return StaticResponsePolicy{
		defaultMaxAge:   cfg.ParsedMaxAge(),
		namespaceMaxAge: cfg.NamespaceMaxAges(),
	}
}

func (p StaticResponsePolicy) CacheControl(result *resolver.Result) string {
	if result == nil || result.Variant == nil {
		return RevalidateCacheControl
	}

	maxAge := p.variantMaxAge(result.Variant)
	if maxAge <= 0 {
		return RevalidateCacheControl
	}
	return fmt.Sprintf("public, max-age=%d, immutable", int(maxAge.Seconds()))
}

func (p StaticResponsePolicy) ExpiresAt(cacheControl string, lastModified time.Time, hasLastModified bool) (time.Time, bool) {
	_ = lastModified
	_ = hasLastModified

	maxAge, ok := cacheControlMaxAge(cacheControl)
	if !ok {
		return time.Time{}, false
	}
	return time.Now().UTC().Add(maxAge), true
}

func (p StaticResponsePolicy) variantMaxAge(variant *catalog.Variant) time.Duration {
	switch variantNamespace(variant) {
	case artifactNamespaceEncoding:
		return resolveNamespaceMaxAge(p.namespaceMaxAge, artifactNamespaceEncoding, p.defaultMaxAge)
	case artifactNamespaceImage:
		return resolveNamespaceMaxAge(p.namespaceMaxAge, artifactNamespaceImage, 0)
	default:
		return 0
	}
}

func cacheControlMaxAge(cacheControl string) (time.Duration, bool) {
	remaining := cacheControl
	for {
		part, rest, found := strings.Cut(remaining, ",")
		directive := strings.TrimSpace(part)
		key, value, hasValue := strings.Cut(directive, "=")
		if hasValue && strings.EqualFold(strings.TrimSpace(key), "max-age") {
			seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil || seconds < 0 {
				return 0, false
			}
			return time.Duration(seconds) * time.Second, true
		}
		if !found {
			return 0, false
		}
		remaining = rest
	}
}

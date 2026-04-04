package cachepolicy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/pquerna/cachecontrol/cacheobject"
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
	headers := make(http.Header, 2)
	headers.Set("Cache-Control", cacheControl)
	if hasLastModified {
		headers.Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
	}

	_, expiresAt, err := cacheobject.UsingRequestResponse(nil, http.StatusOK, headers, false)
	if err != nil || expiresAt.IsZero() {
		return time.Time{}, false
	}
	return expiresAt, true
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

package cachepolicy_test

import (
	"testing"

	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
)

func TestResponsePolicyUsesDefaultMaxAgeForEncodingVariants(t *testing.T) {
	policy := cachepolicy.NewResponsePolicy(&config.Compression{
		MaxAge: "168h",
	})

	cacheControl := policy.CacheControl(&resolver.Result{
		Variant: &catalog.Variant{Encoding: "br"},
	})
	if cacheControl != "public, max-age=604800, immutable" {
		t.Fatalf("unexpected cache-control %q", cacheControl)
	}
}

func TestResponsePolicyDoesNotFallbackImageVariantsToDefaultMaxAge(t *testing.T) {
	policy := cachepolicy.NewResponsePolicy(&config.Compression{
		MaxAge: "168h",
	})

	cacheControl := policy.CacheControl(&resolver.Result{
		Variant: &catalog.Variant{Width: 640},
	})
	if cacheControl != cachepolicy.RevalidateCacheControl {
		t.Fatalf("expected revalidate cache-control, got %q", cacheControl)
	}
}

func TestResponsePolicyUsesImageNamespaceMaxAgeForImageVariants(t *testing.T) {
	policy := cachepolicy.NewResponsePolicy(&config.Compression{
		MaxAge:      "168h",
		ImageMaxAge: "336h",
	})

	cacheControl := policy.CacheControl(&resolver.Result{
		Variant: &catalog.Variant{Width: 640},
	})
	if cacheControl != "public, max-age=1209600, immutable" {
		t.Fatalf("unexpected cache-control %q", cacheControl)
	}
}

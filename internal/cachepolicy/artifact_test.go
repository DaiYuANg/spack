package cachepolicy_test

import (
	"testing"
	"time"

	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/config"
)

func TestArtifactPolicyFallsBackToDefaultMaxAge(t *testing.T) {
	policy := cachepolicy.NewArtifactPolicy(&config.Compression{
		MaxAge: "168h",
	})

	if got := policy.MaxAge("encoding"); got != 168*time.Hour {
		t.Fatalf("expected default max-age, got %s", got)
	}
}

func TestArtifactPolicyUsesNamespaceMaxAgeWhenPresent(t *testing.T) {
	policy := cachepolicy.NewArtifactPolicy(&config.Compression{
		MaxAge:         "168h",
		EncodingMaxAge: "24h",
	})

	if got := policy.MaxAge("encoding"); got != 24*time.Hour {
		t.Fatalf("expected namespace max-age, got %s", got)
	}
}

package workerpool

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/panjf2000/ants/v2"
)

// NewSettingsForTest exposes settings construction for external tests.
func NewSettingsForTest(cfg *config.Async) *Settings {
	return newSettings(cfg)
}

// NewPoolForTest exposes shared pool construction for external tests.
func NewPoolForTest(settings *Settings) (*ants.Pool, error) {
	return newPool(settings)
}

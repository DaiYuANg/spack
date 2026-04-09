package cmd

import (
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/config"
)

// CreateContainerForTest exposes container construction for external tests.
func CreateContainerForTest(loadOptions config.LoadOptions, userModules ...dix.Module) (*dix.App, error) {
	return createContainer(loadOptions, userModules...)
}

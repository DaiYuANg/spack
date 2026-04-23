package cmd

import (
	"github.com/arcgolabs/dix"
	"github.com/daiyuang/spack/internal/config"
)

// CreateContainerForTest exposes container construction for external tests.
func CreateContainerForTest(loadOptions config.LoadOptions, userModules ...dix.Module) (*dix.App, error) {
	return createContainer(loadOptions, userModules...)
}

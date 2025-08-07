package preprocessor

import "github.com/daiyuang/spack/internal/registry"

type Preprocessor interface {
	Name() string
	Order() int
	CanProcess(info *registry.OriginalFileInfo) bool
	Process(info *registry.OriginalFileInfo) error
}

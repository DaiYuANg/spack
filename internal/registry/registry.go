package registry

import (
	"errors"
	"sync/atomic"

	"github.com/daiyuang/spack/internal/constant"
)

var ErrFrozen = errors.New("registry is frozen")

type OriginalFileInfo struct {
	Path     string
	Size     int64
	Hash     string
	Ext      string
	Mimetype string
	Metrics  *Metrics
}

type VariantFileInfo struct {
	Path        string
	Ext         string
	VariantType constant.VariantType
	Size        int64
	Metrics     *Metrics
}

var (
	ErrNotFound = errors.New("not found")
)

type ViewData struct {
	Originals []*OriginalFileInfo
	Variants  []*VariantView
}

type VariantView struct {
	OriginalPath string
	*VariantFileInfo
}

type Registry interface {
	Writer() Writer

	//READ ONLY
	GetOriginal(path string) (*OriginalFileInfo, error)
	GetVariants(originalPath string) ([]*VariantFileInfo, error)
	HasVariants(originalPath string) bool

	CountOriginals() int
	CountVariants(originalPath string) int
	ListOriginals() []*OriginalFileInfo

	ViewData() *ViewData

	Freeze() error
	IsFrozen() bool

	Json() (string, error)
}

type Writer interface {
	RegisterOriginal(info *OriginalFileInfo) error
	AddVariant(path string, v *VariantFileInfo) error
}
type Metrics struct {
	AccessCount atomic.Int64
	BytesSent   atomic.Int64
}

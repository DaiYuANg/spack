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
	Mimetype constant.MimeType
	Metrics  *Metrics
}

var (
	ErrNotFound = errors.New("not found")
)

type ViewData struct {
	Originals []*OriginalFileInfo
}

type Registry interface {
	Writer() Writer

	//READ ONLY
	GetOriginal(path string) (*OriginalFileInfo, error)
	CountOriginals() int
	ListOriginals() []*OriginalFileInfo

	ViewData() *ViewData

	Freeze() error
	IsFrozen() bool
}

type Writer interface {
	RegisterOriginal(info *OriginalFileInfo) error
}
type Metrics struct {
	AccessCount atomic.Int64
	BytesSent   atomic.Int64
}

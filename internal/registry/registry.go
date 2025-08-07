package registry

import (
	"errors"

	"github.com/daiyuang/spack/internal/constant"
)

type OriginalFileInfo struct {
	Path        string
	Size        int64
	Hash        string
	Ext         string
	Mimetype    string
	AccessCount int64
}

type VariantFileInfo struct {
	Path        string
	Ext         string
	VariantType constant.VariantType
	Size        int64
	AccessCount int64
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
	// RegisterOriginal 注册一个原始文件
	RegisterOriginal(info *OriginalFileInfo) error

	// GetOriginal 查询原始文件信息
	GetOriginal(path string) (*OriginalFileInfo, error)

	// AddVariant 增加一个变体文件
	AddVariant(originalPath string, variant *VariantFileInfo)

	// BatchAddVariants 批量添加多个变体文件
	BatchAddVariants(originalPath string, variants []*VariantFileInfo)

	// GetVariants 获取某个原始文件的所有变体
	GetVariants(originalPath string) ([]*VariantFileInfo, error)

	// HasVariants 判断原始文件是否存在变体
	HasVariants(originalPath string) bool

	// CountOriginals 获取注册的原始文件数量
	CountOriginals() int

	// CountVariants 获取某个原始文件变体数量
	CountVariants(originalPath string) int

	ListOriginal() []*OriginalFileInfo

	ViewData() *ViewData
}

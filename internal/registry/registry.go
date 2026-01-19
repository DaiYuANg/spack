package registry

import (
	"errors"
	"sync/atomic"

	"github.com/daiyuang/spack/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
)

type ViewData struct {
	Objects []*model.ObjectInfo
}

type Registry interface {
	// Register 核心注册
	Register(info *model.ObjectInfo) error

	// RegisterParents 关联关系
	RegisterParents(info *model.ObjectInfo, parents ...*model.ObjectInfo) error
	RegisterChildren(info *model.ObjectInfo, children ...*model.ObjectInfo) error

	// FindByKey 查找
	FindByKey(key string) (*model.ObjectInfo, error)
	FindByPath(path string) (*model.ObjectInfo, error)
	// ListChildren 查找指定 key 的所有子节点（可用于查找压缩文件）
	ListChildren(key string) ([]*model.ObjectInfo, error)

	// ListParents 查找指定 key 的所有父节点（可用于回溯原始文件）
	ListParents(key string) ([]*model.ObjectInfo, error)
	// Count 遍历
	Count() int
	List() []*model.ObjectInfo
	ViewData() *ViewData

	// Metrics 统计
	Metrics() *Metrics

	Json() (string, error)
}

type Metrics struct {
	AccessCount atomic.Int64
	BytesSent   atomic.Int64
}

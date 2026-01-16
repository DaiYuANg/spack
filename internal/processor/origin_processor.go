package processor

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/daiyuang/spack/internal/model"
	"github.com/samber/oops"
)

// OriginProcessor 用于收集原始文件信息并注册到 registry，同时维护 DAG
type OriginProcessor struct {
	roots map[string]*model.ObjectInfo
}

// NewOriginProcessor 构造函数
func NewOriginProcessor() *OriginProcessor {
	return &OriginProcessor{
		roots: make(map[string]*model.ObjectInfo),
	}
}

// Name 唯一标识
func (p *OriginProcessor) Name() string {
	return "OriginProcessor"
}

// Match 始终处理所有 original
func (p *OriginProcessor) Match(obj *model.ObjectInfo) bool {
	return true
}

// Run 扫描文件内容，并注册到 registry，同时维护 parent-child 关系
func (p *OriginProcessor) Run(ctx Context) (int64, error) {
	var size int64

	// 计算文件大小
	if ctx.Open != nil {
		r, err := ctx.Open()
		if err != nil {
			return 0, oops.Wrap(err)
		}
		defer r.Close()

		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			size += int64(n)
			if err != nil {
				if err == io.EOF {
					break
				}
				return size, oops.Wrap(err)
			}
		}
	}

	// 判断 rootKey（去掉压缩后缀，如 .gz/.br/.xz 等）
	rootKey := rootKey(ctx.Obj.Key)

	var parentRoot *model.ObjectInfo

	// 查找已有 root
	if root, ok := p.roots[rootKey]; ok {
		parentRoot = root
	}

	// 注册当前文件
	if err := ctx.Registry.Register(ctx.Obj); err != nil {
		return size, oops.Wrap(err)
	}

	// 如果找到了 root，注册为 child
	if parentRoot != nil && parentRoot != ctx.Obj {
		if err := ctx.Registry.RegisterChildren(parentRoot, ctx.Obj); err != nil {
			return size, oops.Wrap(err)
		}
		if err := ctx.Registry.RegisterParents(ctx.Obj, parentRoot); err != nil {
			return size, oops.Wrap(err)
		}
	} else {
		// 否则保存为 root
		p.roots[rootKey] = ctx.Obj
	}

	return size, nil
}

// rootKey 根据文件名去掉压缩后缀，得到原始 root key
func rootKey(key string) string {
	ext := filepath.Ext(key)
	base := strings.TrimSuffix(key, ext)

	// 支持常见压缩后缀
	switch ext {
	case ".gz", ".br", ".xz", ".zip", ".lz4":
		return base
	default:
		return key
	}
}

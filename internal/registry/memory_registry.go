package registry

import (
	"github.com/daiyuang/spack/internal/model"
	. "github.com/goradd/maps"
	"github.com/samber/lo"
)

// InMemoryRegistry 内存版实现
type InMemoryRegistry struct {
	nodes   *SafeMap[string, *model.ObjectNode] // key -> node
	metrics *Metrics
}

func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		nodes:   NewSafeMap[string, *model.ObjectNode](),
		metrics: &Metrics{},
	}
}

// GetOriginal 在运行时读取原始文件信息
// Register 注册单个 ObjectInfo
func (r *InMemoryRegistry) Register(info *model.ObjectInfo) error {
	if node, exists := r.nodes.Load(info.Key); exists {
		// 允许在运行时刷新节点信息（例如同 key 的压缩变体更新）。
		node.Info = info
		return nil
	}

	r.nodes.Set(info.Key, &model.ObjectNode{
		Info:     info,
		Parents:  NewSafeMap[string, *model.ObjectNode](),
		Children: NewSafeMap[string, *model.ObjectNode](),
	})

	return nil
}

// RegisterParents 将 info 与 parents 建立父子关系
func (r *InMemoryRegistry) RegisterParents(info *model.ObjectInfo, parents ...*model.ObjectInfo) error {
	node, ok := r.nodes.Load(info.Key)
	if !ok {
		return ErrNotFound
	}

	lo.ForEach(parents, func(p *model.ObjectInfo, index int) {
		parentNode, ok := r.nodes.Load(p.Key)
		if !ok {
			// 自动注册 parent
			parentNode = &model.ObjectNode{
				Info:     p,
				Parents:  NewSafeMap[string, *model.ObjectNode](),
				Children: NewSafeMap[string, *model.ObjectNode](),
			}
			r.nodes.Set(p.Key, parentNode)
		}

		node.Parents.Set(p.Key, parentNode)
		parentNode.Children.Set(info.Key, node)
	})

	return nil
}

// RegisterChildren 将 info 与 children 建立父子关系
func (r *InMemoryRegistry) RegisterChildren(info *model.ObjectInfo, children ...*model.ObjectInfo) error {
	node, ok := r.nodes.Load(info.Key)
	if !ok {
		return ErrNotFound
	}

	lo.ForEach(children, func(c *model.ObjectInfo, index int) {
		childNode, ok := r.nodes.Load(c.Key)
		if !ok {
			childNode = &model.ObjectNode{
				Info:     c,
				Parents:  NewSafeMap[string, *model.ObjectNode](),
				Children: NewSafeMap[string, *model.ObjectNode](),
			}
			r.nodes.Set(c.Key, childNode)
		}
		node.Children.Set(c.Key, childNode) // info 的子节点列表添加 child
		childNode.Parents.Set(info.Key, node)
	})
	return nil
}
func (r *InMemoryRegistry) FindByKey(key string) (*model.ObjectInfo, error) {
	node, ok := r.nodes.Load(key)
	if !ok {
		return nil, ErrNotFound
	}
	// ---- 更新 Metrics ----
	r.metrics.AccessCount.Add(1)
	r.metrics.BytesSent.Add(node.Info.Size)

	return node.Info, nil
}

// FindByPath 与 FindByKey 逻辑一样（如果 Key=Path 可以共用）
func (r *InMemoryRegistry) FindByPath(path string) (*model.ObjectInfo, error) {
	return r.FindByKey(path)
}

// Count 返回注册的对象数量
func (r *InMemoryRegistry) Count() int {
	return r.nodes.Len()
}

// List 返回所有 ObjectInfo
func (r *InMemoryRegistry) List() []*model.ObjectInfo {
	out := make([]*model.ObjectInfo, 0, r.nodes.Len())
	for _, n := range r.nodes.All() {
		out = append(out, n.Info)
	}
	return out
}

// ViewData 返回对象快照
func (r *InMemoryRegistry) ViewData() *ViewData {
	return &ViewData{
		Objects: r.List(),
	}
}

// Metrics 返回访问统计
func (r *InMemoryRegistry) Metrics() *Metrics {
	return r.metrics
}

func (r *InMemoryRegistry) ListChildren(key string) ([]*model.ObjectInfo, error) {
	node, ok := r.nodes.Load(key)
	if !ok {
		return nil, ErrNotFound
	}

	children := make([]*model.ObjectInfo, 0, node.Children.Len())
	for _, childNode := range node.Children.All() {
		children = append(children, childNode.Info)
	}
	return children, nil
}

func (r *InMemoryRegistry) ListParents(key string) ([]*model.ObjectInfo, error) {
	node, ok := r.nodes.Load(key)
	if !ok {
		return nil, ErrNotFound
	}

	parents := make([]*model.ObjectInfo, 0, node.Parents.Len())
	for _, parentNode := range node.Parents.All() {
		parents = append(parents, parentNode.Info)
	}
	return parents, nil
}

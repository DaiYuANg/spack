package registry

import (
	"sync"

	"github.com/daiyuang/spack/internal/model"
)

// InMemoryRegistry 内存版实现
type InMemoryRegistry struct {
	mu      sync.RWMutex
	nodes   map[string]*model.ObjectNode // key -> node
	metrics *Metrics
}

func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		nodes:   make(map[string]*model.ObjectNode),
		metrics: &Metrics{},
	}
}

// GetOriginal 在运行时读取原始文件信息
// Register 注册单个 ObjectInfo
func (r *InMemoryRegistry) Register(info *model.ObjectInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[info.Key]; exists {
		return nil // 已存在则忽略
	}

	r.nodes[info.Key] = &model.ObjectNode{
		Info:     info,
		Parents:  make(map[string]*model.ObjectNode),
		Children: make(map[string]*model.ObjectNode),
	}

	return nil
}

// RegisterParents 将 info 与 parents 建立父子关系
func (r *InMemoryRegistry) RegisterParents(info *model.ObjectInfo, parents ...*model.ObjectInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, ok := r.nodes[info.Key]
	if !ok {
		return ErrNotFound
	}

	for _, p := range parents {
		parentNode, ok := r.nodes[p.Key]
		if !ok {
			// 自动注册 parent
			parentNode = &model.ObjectNode{
				Info:     p,
				Parents:  make(map[string]*model.ObjectNode),
				Children: make(map[string]*model.ObjectNode),
			}
			r.nodes[p.Key] = parentNode
		}

		node.Parents[p.Key] = parentNode
		parentNode.Children[info.Key] = node
	}

	return nil
}

// RegisterChildren 将 info 与 children 建立父子关系
func (r *InMemoryRegistry) RegisterChildren(info *model.ObjectInfo, children ...*model.ObjectInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, ok := r.nodes[info.Key]
	if !ok {
		return ErrNotFound
	}

	for _, c := range children {
		childNode, ok := r.nodes[c.Key]
		if !ok {
			childNode = &model.ObjectNode{
				Info:     c,
				Parents:  make(map[string]*model.ObjectNode),
				Children: make(map[string]*model.ObjectNode),
			}
			r.nodes[c.Key] = childNode
		}

		node.Children[c.Key] = childNode
		childNode.Parents[info.Key] = node
	}

	return nil
}
func (r *InMemoryRegistry) FindByKey(key string) (*model.ObjectInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, ok := r.nodes[key]
	if !ok {
		return nil, ErrNotFound
	}

	return node.Info, nil
}

// FindByPath 与 FindByKey 逻辑一样（如果 Key=Path 可以共用）
func (r *InMemoryRegistry) FindByPath(path string) (*model.ObjectInfo, error) {
	return r.FindByKey(path)
}

// Count 返回注册的对象数量
func (r *InMemoryRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// List 返回所有 ObjectInfo
func (r *InMemoryRegistry) List() []*model.ObjectInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*model.ObjectInfo, 0, len(r.nodes))
	for _, n := range r.nodes {
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

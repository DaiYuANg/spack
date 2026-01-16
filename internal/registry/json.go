package registry

import (
	"encoding/json"
	"errors"

	"github.com/daiyuang/spack/internal/model"
	"github.com/samber/lo"
)

// 可序列化的 ObjectInfo 表示（不包含 Reader）
type objectInfoJSON struct {
	Key      string            `json:"key"`
	FullPath string            `json:"fullPath"`
	Mimetype string            `json:"mimetype"`
	Size     int64             `json:"size"`
	IsDir    bool              `json:"isDir"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// 扁平化节点表示：完整 info + 父/子 key 列表
type nodeFlatJSON struct {
	Info     objectInfoJSON `json:"info"`
	Parents  []string       `json:"parents,omitempty"`
	Children []string       `json:"children,omitempty"`
}

// 嵌套树形节点表示：Info + 嵌套 children（用于从 roots 展开）
type nodeTreeJSON struct {
	Info     objectInfoJSON  `json:"info"`
	Children []*nodeTreeJSON `json:"children,omitempty"`
}

// Json 返回扁平化的 JSON（每个节点的完整 info + 父/子 key 列表）
func (r *InMemoryRegistry) Json() (string, error) {
	if r == nil {
		return "", errors.New("registry is nil")
	}

	out := make([]nodeFlatJSON, 0, r.nodes.Len())
	for _, n := range r.nodes.All() {
		// prepare info
		info := objectInfoJSON{
			Key:      n.Info.Key,
			FullPath: n.Info.FullPath,
			Mimetype: n.Info.MimeString(),
			Size:     n.Info.Size,
			IsDir:    n.Info.IsDir,
			Metadata: n.Info.Metadata,
		}

		// parents keys
		parents := make([]string, 0, n.Parents.Len())
		for k := range n.Parents.All() {
			parents = append(parents, k)
		}

		// children keys
		children := make([]string, 0, n.Children.Len())
		for k := range n.Children.All() {
			children = append(children, k)
		}

		out = append(out, nodeFlatJSON{
			Info:     info,
			Parents:  parents,
			Children: children,
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JsonTree 从所有 root（无 parents 的节点）开始，递归构建嵌套结构并返回 JSON。
// 为避免无限循环，使用 visited map（按 key）。
func (r *InMemoryRegistry) JsonTree() (string, error) {
	if r == nil {
		return "", errors.New("registry is nil")
	}

	// 找到 roots（无 parents 的节点）
	roots := make([]*model.ObjectNode, 0)
	for _, n := range r.nodes.All() {
		if n.Parents.Len() == 0 {
			roots = append(roots, n)
		}
	}

	visited := make(map[string]*nodeTreeJSON)

	var build func(n *model.ObjectNode) *nodeTreeJSON
	build = func(n *model.ObjectNode) *nodeTreeJSON {
		if existing, ok := visited[n.Info.Key]; ok {
			return existing
		}
		info := objectInfoJSON{
			Key:      n.Info.Key,
			FullPath: n.Info.FullPath,
			Mimetype: n.Info.MimeString(),
			Size:     n.Info.Size,
			IsDir:    n.Info.IsDir,
			Metadata: n.Info.Metadata,
		}
		node := &nodeTreeJSON{Info: info}
		visited[n.Info.Key] = node

		for _, child := range n.Children.All() {
			node.Children = append(node.Children, build(child))
		}
		return node
	}

	out := make([]*nodeTreeJSON, 0, len(roots))
	lo.ForEach(roots, func(item *model.ObjectNode, index int) {
		out = append(out, build(item))
	})

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

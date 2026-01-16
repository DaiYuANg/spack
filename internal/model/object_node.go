package model

// ObjectNode 内部节点，用于表示 DAG 结构
type ObjectNode struct {
	Info     *ObjectInfo
	Parents  map[string]*ObjectNode
	Children map[string]*ObjectNode
}

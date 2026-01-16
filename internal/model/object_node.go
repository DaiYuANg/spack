package model

import . "github.com/goradd/maps"

// ObjectNode 内部节点，用于表示 DAG 结构
type ObjectNode struct {
	Info     *ObjectInfo
	Parents  *SafeMap[string, *ObjectNode]
	Children *SafeMap[string, *ObjectNode]
}

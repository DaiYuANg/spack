package scanner

import "github.com/daiyuang/spack/internal/model"

// Backend 是抽象存储，Scanner 通过它遍历和读取对象
type Backend interface {
	// Walk 遍历所有对象（目录/文件）
	// walkFn 对每个对象调用，非 error 时继续
	Walk(walkFn func(obj *model.ObjectInfo) error) error

	// Stat 获取具体路径的对象元信息
	Stat(key string) (*model.ObjectInfo, error)
}

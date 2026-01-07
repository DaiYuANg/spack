package storage

import "io"

type Key string

type Storage interface {
	// Open 以只读方式打开 blob
	Open(key Key) (io.ReadCloser, error)

	// Put 写入一个 blob，返回最终 key
	Put(r io.Reader) (Key, int64, error)

	// Exists 判断 blob 是否存在
	Exists(key Key) bool
}

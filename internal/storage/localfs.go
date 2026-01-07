package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/samber/oops"
	"github.com/spf13/afero"
)

type LocalFS struct {
	fs     afero.Fs
	root   string
	logger *slog.Logger

	// 并发安全的 key index（Key -> struct{}）
	idx sync.Map
}

func NewLocalFS(fs afero.Fs, root string, logger *slog.Logger) (*LocalFS, error) {
	if err := fs.MkdirAll(root, 0o755); err != nil {
		return nil, oops.Wrap(err)
	}

	l := &LocalFS{
		fs:     fs,
		root:   root,
		logger: logger,
	}

	// 启动时 preload 已有 blob
	if err := l.preload(); err != nil {
		return nil, oops.Wrap(err)
	}

	return l, nil
}

func (l *LocalFS) preload() error {
	return afero.Walk(l.fs, l.root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		key := Key(filepath.Base(p))
		l.idx.Store(key, struct{}{})
		return nil
	})
}

func (l *LocalFS) blobPath(key Key) string {
	return filepath.Join(l.root, string(key))
}

/*
Exists
- 先查 idx
- miss 再查 fs（防止 idx 不一致）
*/
func (l *LocalFS) Exists(key Key) bool {
	_, ok := l.idx.Load(key)
	return ok
}
func (l *LocalFS) Open(key Key) (io.ReadCloser, error) {
	return l.fs.Open(l.blobPath(key))
}

/*
Put
- 流式写入
- 计算 hash
- 原子 rename
- 天然去重
*/
func (l *LocalFS) Put(r io.Reader) (Key, int64, error) {
	// 1. 写入临时文件并计算 hash
	tmp, err := afero.TempFile(l.fs, l.root, ".tmp-*")
	if err != nil {
		return "", 0, oops.Wrap(err)
	}
	defer tmp.Close()

	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hasher), r)
	if err != nil {
		_ = l.fs.Remove(tmp.Name())
		return "", n, oops.Wrap(err)
	}

	key := Key(hex.EncodeToString(hasher.Sum(nil)))
	finalPath := l.blobPath(key)

	// 2. CAS：原子检查 + 标记
	if _, loaded := l.idx.LoadOrStore(key, struct{}{}); loaded {
		// 已存在（可能是并发 Put 或 preload）
		_ = l.fs.Remove(tmp.Name())
		return key, n, nil
	}

	// 3. 落盘
	if err := l.fs.Rename(tmp.Name(), finalPath); err != nil {
		// 回滚 idx（非常重要）
		l.idx.Delete(key)
		_ = l.fs.Remove(tmp.Name())
		return "", n, oops.Wrap(err)
	}

	l.logger.Debug(
		"blob stored",
		"key", key,
		"bytes", n,
	)

	return key, n, nil
}

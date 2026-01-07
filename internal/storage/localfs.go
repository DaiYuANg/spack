package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/samber/oops"
	"github.com/spf13/afero"
)

type LocalFS struct {
	fs     afero.Fs
	root   string
	logger *slog.Logger

	mu  sync.RWMutex
	idx map[Key]string // 可选 cache，加速 Exists / 列表
}

func NewLocalFS(
	fs afero.Fs,
	root string,
	logger *slog.Logger,
) (*LocalFS, error) {
	if err := fs.MkdirAll(root, 0o755); err != nil {
		return nil, oops.Wrap(err)
	}

	l := &LocalFS{
		fs:     fs,
		root:   root,
		logger: logger,
		idx:    make(map[Key]string),
	}

	// 启动时构建 idx（可选，但推荐）
	if err := l.buildIndex(); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *LocalFS) buildIndex() error {
	entries, err := afero.ReadDir(l.fs, l.root)
	if err != nil {
		return oops.Wrap(err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		l.idx[Key(e.Name())] = ""
	}

	l.logger.Info(
		"localfs index built",
		"count", len(l.idx),
	)

	return nil
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
	l.mu.RLock()
	_, ok := l.idx[key]
	l.mu.RUnlock()
	if ok {
		return true
	}
	path := l.blobPath(key)
	_, err := l.fs.Stat(path)
	if err == nil {
		l.mu.Lock()
		l.idx[key] = path
		l.mu.Unlock()
		return true
	}

	return false
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

	// fast-path: 已存在
	if l.Exists(key) {
		_ = l.fs.Remove(tmp.Name())
		return key, n, nil
	}

	// 原子提交
	if err := l.fs.Rename(tmp.Name(), finalPath); err != nil {
		// 并发场景下，可能是另一个 writer 已经写入
		if l.Exists(key) {
			_ = l.fs.Remove(tmp.Name())
			return key, n, nil
		}
		_ = l.fs.Remove(tmp.Name())
		return "", n, oops.Wrap(err)
	}

	l.mu.Lock()
	l.idx[key] = finalPath
	l.mu.Unlock()

	l.logger.Debug(
		"blob stored",
		"key", key,
		"bytes", n,
	)

	return key, n, nil
}

func (l *LocalFS) IndexSnapshot() map[Key]string {
	out := make(map[Key]string)

	for key := range l.idx {
		out[key] = l.idx[key]
	}

	return out
}

package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/daiyuang/spack/internal/model"
	"github.com/samber/oops"
)

// FileProcessor 定义处理单个文件的回调
type FileProcessor func(obj *model.ObjectInfo, hash string) error

type Scanner struct {
	backend Backend
}

// NewScanner 创建 Scanner
func NewScanner(backend Backend) *Scanner {
	return &Scanner{backend: backend}
}

// Scan 遍历 backend 并处理文件
func (s *Scanner) Scan(process FileProcessor) error {
	return s.backend.Walk(func(obj *model.ObjectInfo) error {
		if obj.IsDir {
			// 跳过目录
			return nil
		}

		// 计算 Hash（按需）
		h, err := calcHash(obj)
		if err != nil {
			return oops.Wrap(err)
		}

		return process(obj, h)
	})
}

// calcHash 计算对象内容 sha256
func calcHash(obj *model.ObjectInfo) (string, error) {
	r, err := obj.Reader()
	if err != nil {
		return "", err
	}
	defer func(r io.ReadCloser) {
		err := r.Close()
		if err != nil {
			panic(oops.In("scanner calchash").Wrap(err))
		}
	}(r)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

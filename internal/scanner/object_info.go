package scanner

import (
	"io"
	"os"
	"path/filepath"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/oops"
)

// ObjectInfo 代表一个可扫描对象的元信息
type ObjectInfo struct {
	Key      string // 相对路径或对象 key
	Size     int64
	FullPath string
	Reader   func() (io.ReadCloser, error) // 延迟打开 Reader
	IsDir    bool
	Metadata map[string]string // 可选字段（MIME、etag 等）
	Mimetype constant.MimeType
}

func newObjectInfo(root, fullPath string, info os.FileInfo) (*ObjectInfo, error) {
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return nil, oops.Wrap(err)
	}
	key := filepath.ToSlash(rel)
	mimetype := pkg.DetectMIME(fullPath)
	return &ObjectInfo{
		Key:      key,
		Size:     info.Size(),
		IsDir:    info.IsDir(),
		FullPath: fullPath,
		Reader: func() (io.ReadCloser, error) {
			if info.IsDir() {
				return nil, nil
			}
			return os.Open(fullPath)
		},
		Mimetype: mimetype,
	}, nil
}

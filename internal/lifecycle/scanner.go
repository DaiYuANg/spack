package lifecycle

import (
	"mime"
	"path"
	"path/filepath"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/gabriel-vasile/mimetype"
	"github.com/samber/oops"
)

func scan(scannerInstance *scanner.Scanner, reg registry.Registry) error {
	err := scannerInstance.Scan(func(obj *scanner.ObjectInfo, hash string) error {
		// 注册 original
		info := &registry.OriginalFileInfo{
			Path:     obj.Key,
			Size:     obj.Size,
			Hash:     hash,
			Ext:      filepath.Ext(obj.Key),
			Mimetype: detectMIME(obj),
		}

		err := reg.Writer().RegisterOriginal(info)
		if err != nil {
			return err
		}

		// 这里可把 obj.Reader() 内容送给变体 worker 池
		return nil
	})
	if err != nil {
		return oops.Wrap(err)
	}
	err = reg.Freeze()
	if err != nil {
		return oops.Wrap(err)
	}
	return nil
}

func detectMIME(obj *scanner.ObjectInfo) string {
	r, err := obj.Reader()
	if err == nil && r != nil {
		defer r.Close()
		mtype, _ := mimetype.DetectReader(r)
		return mtype.String()
	}
	// 失败 fallback to extension
	return mime.TypeByExtension(path.Ext(obj.Key))
}

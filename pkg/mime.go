package pkg

import (
	"mime"
	"os"
	"path"

	"github.com/gabriel-vasile/mimetype"
)

func DetectMIME(filePath string) string {
	// 尝试通过内容检测
	f, err := os.Open(filePath)
	if err == nil {
		defer f.Close()
		mtype, _ := mimetype.DetectReader(f)
		if mtype != nil && mtype.String() != "" {
			return mtype.String()
		}
	}

	// fallback 到扩展名
	mtype := mime.TypeByExtension(path.Ext(filePath))
	if mtype == "" {
		return "application/octet-stream"
	}
	return mtype
}

package pkg

import (
	"mime"
	"os"
	"path"
	"strings"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/gabriel-vasile/mimetype"
)

func DetectMIME(filePath string) constant.MimeType {
	ext := strings.ToLower(path.Ext(filePath))

	// 1. Web 语义优先修正（强规则）
	switch ext {
	case ".js":
		return constant.ApplicationJavascript
	case ".css":
		return constant.Css
	case ".html", ".htm":
		return constant.Html
	case ".json":
		return constant.Json
	case ".svg":
		return constant.Svg
	}

	// 2. 内容嗅探（主要用于二进制类型）
	if f, err := os.Open(filePath); err == nil {
		defer f.Close()

		if mtype, err := mimetype.DetectReader(f); err == nil && mtype != nil {
			mt := mtype.String()

			// 去掉 charset 等参数
			if idx := strings.Index(mt, ";"); idx > 0 {
				mt = mt[:idx]
			}

			switch constant.MimeType(mt) {
			case constant.Png:
				return constant.Png
			case constant.Jpeg:
				return constant.Jpeg
			case constant.Svg:
				return constant.Svg
			}
		}
	}

	// 3. fallback：Go 标准库（再 normalize 一次）
	if mt := mime.TypeByExtension(ext); mt != "" {
		if idx := strings.Index(mt, ";"); idx > 0 {
			mt = mt[:idx]
		}
		return constant.MimeType(mt)
	}

	// 4. 兜底
	return constant.OctetStream
}

package finder

import (
	"os"
	"strings"

	"log/slog"

	"github.com/daiyuang/spack/internal/model"
)

// Registry 你已有的接口
type Registry interface {
	FindByPath(path string) (*model.ObjectInfo, error)
	ListChildren(key string) ([]*model.ObjectInfo, error)
}

// Lookup 查找文件，同时处理压缩变体
func (p *Finder) Lookup(ctx LookupOption) (*Result, error) {
	p.Debug("Lookup start",
		slog.String("path", ctx.Path),
		slog.String("acceptEncoding", ctx.AcceptEncoding),
	)

	// 1️⃣ 查找原文件
	orig, err := p.registry.FindByPath(ctx.Path)
	if err != nil {
		p.Debug("Original file not found",
			slog.String("path", ctx.Path),
			slog.String("err", err.Error()),
		)
		return nil, err
	}
	p.Debug("Original file found",
		slog.String("key", orig.Key),
		slog.String("fullPath", orig.FullPath),
		slog.String("mime", string(orig.Mimetype)),
	)

	selected := orig
	selectedEncoding := ""

	// 2️⃣ 如果客户端提供 Accept-Encoding，尝试匹配子节点（压缩文件）
	if ctx.AcceptEncoding != "" {
		children, _ := p.registry.ListChildren(orig.Key)
		p.Debug("Found children", slog.Int("count", len(children)))

		encodings := p.parseAcceptEncoding(ctx.AcceptEncoding)
		p.Debug("Parsed Accept-Encoding", slog.Any("encodings", encodings))

	loopEnc:
		for _, enc := range encodings {
			for _, child := range children {
				if p.matchesEncoding(enc, child.Key, orig.Key) {
					selected = child
					selectedEncoding = enc
					p.Debug("Matched child encoding",
						slog.String("childKey", child.Key),
						slog.String("encoding", enc),
					)
					break loopEnc
				} else {
					p.Debug("Child not matched",
						slog.String("childKey", child.Key),
						slog.String("expectedEncoding", enc),
					)
				}
			}
		}
	}

	// 3️⃣ 读取文件内容
	content, err := os.ReadFile(selected.FullPath)
	if err != nil {
		p.Debug("Failed to read file",
			slog.String("path", selected.FullPath),
			slog.String("err", err.Error()),
		)
		return nil, err
	}
	p.Debug("File read successfully",
		slog.String("path", selected.FullPath),
		slog.String("encoding", selectedEncoding),
	)

	return &Result{
		Key:       orig.Key,
		Data:      content,
		MediaType: orig.Mimetype,    // 始终使用原始 MIME
		Encoding:  selectedEncoding, // 实际使用的编码
	}, nil
}

// parseAcceptEncoding 简单解析 Accept-Encoding
func (p *Finder) parseAcceptEncoding(header string) []string {
	if header == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		enc := strings.TrimSpace(strings.Split(part, ";")[0])
		if enc != "" {
			out = append(out, enc)
		}
	}
	return out
}

// matchesEncoding 判断 childKey 是否是 parentKey 的指定编码版本
func (p *Finder) matchesEncoding(enc, childKey, parentKey string) bool {
	base := childKey
	matched := false

	switch enc {
	case "gzip":
		if strings.HasSuffix(childKey, ".gz") {
			base = strings.TrimSuffix(childKey, ".gz")
			matched = base == parentKey
		}
	case "br":
		if strings.HasSuffix(childKey, ".br") {
			base = strings.TrimSuffix(childKey, ".br")
			matched = base == parentKey
		}
	}

	p.Debug("Encoding match check",
		slog.String("childKey", childKey),
		slog.String("parentKey", parentKey),
		slog.String("encoding", enc),
		slog.String("base", base),
		slog.Bool("matched", matched),
	)

	return matched
}

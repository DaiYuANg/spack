package finder

import (
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/daiyuang/spack/internal/model"
	"github.com/samber/lo"
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

	requestedEncodings := p.parseAcceptEncoding(ctx.AcceptEncoding)
	selected := orig
	selectedEncoding := ""

	// 2️⃣ 如果客户端提供 Accept-Encoding，尝试匹配子节点（压缩文件）
	if len(requestedEncodings) > 0 {
		children, err := p.registry.ListChildren(orig.Key)
		if err != nil {
			p.Debug("List children failed",
				slog.String("key", orig.Key),
				slog.String("err", err.Error()),
			)
			children = nil
		}
		p.Debug("Found children", slog.Int("count", len(children)))
		p.Debug("Parsed Accept-Encoding", slog.Any("encodings", requestedEncodings))

	loopEnc:
		for _, enc := range requestedEncodings {
			for _, child := range children {
				if p.isUsableVariant(orig, child, enc) {
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
	etag := ""
	if selected.Metadata != nil {
		etag = selected.Metadata["etag"]
	}
	if etag == "" && orig.Metadata != nil {
		etag = orig.Metadata["etag"]
	}

	return &Result{
		Key:            orig.Key,
		Data:           content,
		MediaType:      orig.Mimetype,      // 始终使用原始 MIME
		Encoding:       selectedEncoding,   // 实际使用的编码
		AcceptEncoding: requestedEncodings, // 如果未命中可用于后台压缩
		ETag:           etag,
	}, nil
}

// parseAcceptEncoding parses Accept-Encoding and returns preferred supported encodings.
func (p *Finder) parseAcceptEncoding(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}

	explicit := make(map[string]float64, 4)
	wildcardQ := 0.0
	hasWildcard := false
	parts := strings.Split(header, ",")
	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		fragments := strings.Split(part, ";")
		enc := strings.ToLower(strings.TrimSpace(fragments[0]))
		if enc == "" {
			continue
		}
		q := 1.0
		for _, rawParam := range fragments[1:] {
			param := strings.TrimSpace(rawParam)
			if !strings.HasPrefix(strings.ToLower(param), "q=") {
				continue
			}
			v := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(param), "q="))
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			if parsed < 0 {
				parsed = 0
			}
			if parsed > 1 {
				parsed = 1
			}
			q = parsed
		}
		if enc == "*" {
			wildcardQ = q
			hasWildcard = true
			continue
		}
		if oldQ, exists := explicit[enc]; !exists || q > oldQ {
			explicit[enc] = q
		}
	}

	type candidate struct {
		encoding string
		q        float64
		priority int
	}
	supported := []string{"br", "gzip"}
	candidates := make([]candidate, 0, len(supported))
	for i, enc := range supported {
		q, exists := explicit[enc]
		if !exists {
			if hasWildcard {
				q = wildcardQ
			} else {
				q = 0
			}
		}
		if q <= 0 {
			continue
		}
		candidates = append(candidates, candidate{
			encoding: enc,
			q:        q,
			priority: i,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].q == candidates[j].q {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].q > candidates[j].q
	})

	return lo.Map(candidates, func(c candidate, _ int) string {
		return c.encoding
	})
}

func (p *Finder) isUsableVariant(parent, child *model.ObjectInfo, enc string) bool {
	if !p.matchesEncoding(enc, child.Key, parent.Key) {
		return false
	}
	if child.Metadata != nil {
		if variantEncoding := strings.ToLower(strings.TrimSpace(child.Metadata["encoding"])); variantEncoding != "" && variantEncoding != enc {
			return false
		}
	}

	if parent.Metadata != nil && child.Metadata != nil {
		sourceHash := strings.TrimSpace(parent.Metadata["source_hash"])
		variantSourceHash := strings.TrimSpace(child.Metadata["source_hash"])
		if sourceHash != "" && variantSourceHash != "" && sourceHash != variantSourceHash {
			p.Debug("Skip stale variant due to source hash mismatch",
				slog.String("parent", parent.Key),
				slog.String("child", child.Key),
				slog.String("parentHash", sourceHash),
				slog.String("variantHash", variantSourceHash),
			)
			return false
		}
	}

	if child.FullPath == "" {
		return false
	}
	if _, err := os.Stat(child.FullPath); err != nil {
		p.Debug("Skip missing variant file",
			slog.String("child", child.Key),
			slog.String("path", child.FullPath),
			slog.String("err", err.Error()),
		)
		return false
	}
	return true
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

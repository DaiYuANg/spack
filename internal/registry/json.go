package registry

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

func (r *Metadata) Json() (string, error) {
	originals := r.registry.ListOriginals()

	// 构造 JSON 对象
	jsonOrig := lo.Map(originals, func(e *OriginalFileInfo, _ int) *jsonOriginal {
		return &jsonOriginal{
			Path: e.Path,
			MIME: e.Mimetype,
			Size: e.Size,
			Hash: e.Hash,
		}
	})

	// 先按 Path 排序
	sort.SliceStable(jsonOrig, func(i, j int) bool {
		return jsonOrig[i].Path < jsonOrig[j].Path
	})

	// 按 MIME 分类
	byMime := lo.GroupBy(jsonOrig, func(o *jsonOriginal) constant.MimeType {
		return o.MIME
	})

	report := jsonReport{
		Originals: jsonOrig,
		ByMIME:    byMime,
		Total:     len(jsonOrig),
	}

	// 序列化为字符串
	bs, err := json.Marshal(report)
	if err != nil {
		mashhalerr := errors.New("json marshal error: " + err.Error())
		return "", oops.Wrap(mashhalerr)
	}
	return string(bs), nil
}

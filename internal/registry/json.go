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
		vars, _ := r.registry.GetVariants(e.Path)
		return &jsonOriginal{
			Path:     e.Path,
			MIME:     e.Mimetype,
			Size:     e.Size,
			Hash:     e.Hash,
			Variants: mapVariants(vars),
			VarCount: len(vars),
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

func mapVariants(vars []*VariantFileInfo) []*jsonVariant {
	return lo.Map(vars, func(v *VariantFileInfo, _ int) *jsonVariant {
		return &jsonVariant{
			Ext:         v.Ext,
			VariantType: string(v.VariantType),
			Size:        v.Size,
			StorageKey:  v.StorageKey,
		}
	})
}

package printer

import (
	"encoding/json"
	"os"
	"sort"

	"log/slog"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/samber/lo"
)

func PrintJSON(r registry.Registry, logger *slog.Logger) {
	originals := r.ListOriginals()

	jsonOrig := lo.Map(originals, func(e *registry.OriginalFileInfo, _ int) *jsonOriginal {
		vars, _ := r.GetVariants(e.Path)
		return &jsonOriginal{
			Path:     e.Path,
			MIME:     e.Mimetype,
			Size:     e.Size,
			Hash:     e.Hash,
			Variants: mapVariants(vars),
			VarCount: len(vars),
		}
	})

	byMime := lo.GroupBy(jsonOrig, func(o *jsonOriginal) string {
		return o.MIME
	})

	for _, arr := range byMime {
		sort.SliceStable(arr, func(i, j int) bool {
			return arr[i].Path < arr[j].Path
		})
	}

	report := jsonReport{
		Originals: jsonOrig,
		ByMIME:    byMime,
		Total:     len(jsonOrig),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		logger.Error("json encode failed", "err", err)
	}
}

func mapVariants(vars []*registry.VariantFileInfo) []*jsonVariant {
	return lo.Map(vars, func(v *registry.VariantFileInfo, _ int) *jsonVariant {
		return &jsonVariant{
			Path:        v.Path,
			Ext:         v.Ext,
			VariantType: string(v.VariantType),
			Size:        v.Size,
		}
	})
}

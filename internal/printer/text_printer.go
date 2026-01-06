package printer

import (
	"os"
	"sort"

	"log/slog"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
)

func PrintTable(r registry.Registry, logger *slog.Logger) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{
		"Request Path", "MIME Type", "Size(Bytes)", "Hash", "Variants",
	})

	originals := r.ListOriginals()

	sort.SliceStable(originals, func(i, j int) bool {
		return originals[i].Path < originals[j].Path
	})

	lo.ForEach(originals, func(e *registry.OriginalFileInfo, _ int) {
		t.AppendRow(table.Row{
			e.Path,
			e.Mimetype,
			e.Size,
			e.Hash,
			r.CountVariants(e.Path),
		})
	})

	t.AppendFooter(table.Row{
		"TOTAL", "", "", "", len(originals),
	})

	t.Render()

	logger.Info("printer table rendered",
		"originals", len(originals),
	)
}

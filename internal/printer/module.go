package printer

import (
	"os"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

var Module = fx.Module("printer", fx.Invoke(printer))

func printer(r registry.Registry) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{"Request Path", "MIME Type", "Size", "Hash"})

	lo.ForEach(r.ListOriginal(), func(e *registry.OriginalFileInfo, index int) {
		t.AppendRow(table.Row{
			e.Path,
			e.Mimetype,
			e.Size,
			e.Hash,
		})
	})

	t.Render()
}

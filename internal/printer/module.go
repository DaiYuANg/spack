package printer

import (
	"os"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/jedib0t/go-pretty/v6/table"
	"go.uber.org/fx"
)

var Module = fx.Module("printer", fx.Invoke(printer))

func printer(registry registry.Registry) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{"Request Path", "MIME Type", "Size", "Hash"})

	for _, e := range registry.ListOriginal() {
		t.AppendRow(table.Row{
			e.Path,
			e.Mimetype,
			e.Size,
			e.Hash,
		})
	}

	t.Render()
}

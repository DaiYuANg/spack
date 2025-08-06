package printer

import (
	"os"
	"strings"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/jedib0t/go-pretty/v6/table"
	"go.uber.org/fx"
)

var Module = fx.Module("printer", fx.Invoke(printer))

func printer(registry registry.Registry) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{"Request Path", "Actual Path", "MIME Type", "Preprocessors", "Version"})

	for _, e := range registry.List() {
		t.AppendRow(table.Row{
			e.RequestPath,
			e.ActualPath,
			e.MimeType,
			strings.Join(e.Preprocessors, ", "),
			e.Version,
		})
	}

	t.Render()
}

package catalog

import "github.com/DaiYuANg/arcgo/dix"

var Module = dix.NewModule("catalog",
	dix.WithModuleProviders(
		dix.Provider0(NewRuntimeMetrics),
		dix.Provider0(NewCatalog),
	),
)

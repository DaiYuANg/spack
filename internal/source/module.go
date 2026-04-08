package source

import "github.com/DaiYuANg/arcgo/dix"

var Module = dix.NewModule("source",
	dix.WithModuleProviders(
		dix.ProviderErr2(newSourceFromConfig),
	),
)

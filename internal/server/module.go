package server

import "github.com/DaiYuANg/arcgo/dix"

var Module = dix.NewModule("server",
	dix.WithModuleProviders(
		dix.Provider6(newServer),
	),
)

// Package resolver maps requests to source assets or generated variants.
package resolver

import "github.com/DaiYuANg/arcgo/dix"

var Module = dix.NewModule("resolver",
	dix.WithModuleProviders(
		dix.Provider5(newResolver),
	),
)

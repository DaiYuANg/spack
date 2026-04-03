package resolver

import "github.com/DaiYuANg/arcgo/dix"

var Module = dix.NewModule("resolver",
	dix.WithModuleProviders(
		dix.Provider3(newResolverFromDeps),
	),
)

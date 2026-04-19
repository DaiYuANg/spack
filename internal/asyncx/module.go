package asyncx

import (
	"github.com/DaiYuANg/arcgo/dix"
)

var Module = dix.NewModule("asyncx",
	dix.WithModuleProviders(
		dix.Provider1(newSettings),
		dix.Provider1(NewRuntimeMetrics),
	),
)

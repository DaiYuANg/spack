package assetcache

import (
	"github.com/DaiYuANg/arcgo/dix"
)

var Module = dix.NewModule("assetcache",
	dix.WithModuleProviders(
		dix.Provider3(newCache),
	),
)

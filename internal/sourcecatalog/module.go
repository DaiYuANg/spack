// Package sourcecatalog scans source files into catalog assets and source-managed variants.
package sourcecatalog

import "github.com/DaiYuANg/arcgo/dix"

var Module = dix.NewModule("sourcecatalog",
	dix.WithModuleProviders(
		dix.Provider2(NewScanner),
	),
)

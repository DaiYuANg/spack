// Package validation provides shared validator wiring and helper utilities.
package validation

import (
	"github.com/arcgolabs/dix"
)

var Module = dix.NewModule("validation",
	dix.WithModuleProviders(
		dix.ProviderErr0(New),
	),
)

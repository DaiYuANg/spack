// Package appmeta wires application metadata into the DI container.
package appmeta

import (
	"errors"
	"runtime/debug"

	"github.com/arcgolabs/dix"
	"github.com/samber/oops"
)

var Module = dix.NewModule("appmeta",
	dix.WithModuleProviders(
		dix.ProviderErr0(func() (dix.AppMeta, error) {
			info, ok := debug.ReadBuildInfo()
			if !ok {
				return dix.AppMeta{}, oops.In("appmeta").Wrap(errors.New("could not read build info"))
			}
			return dix.AppMeta{
				Version: info.Main.Version,
			}, nil
		}),
	),
)

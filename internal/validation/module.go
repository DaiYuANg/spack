// Package validation provides shared validator wiring and helper utilities.
package validation

import (
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/go-playground/validator/v10"
	"github.com/samber/do/v2"
)

var Module = dix.NewModule("validation",
	dix.WithModuleProviders(
		dix.RawProviderWithMetadata(registerValidatorProvider, dix.ProviderMetadata{
			Label:  "ValidatorProvider",
			Output: dix.TypedService[*validator.Validate](),
			Raw:    true,
		}),
	),
)

func registerValidatorProvider(c *dix.Container) {
	do.ProvideNamed(c.Raw(), dix.TypedService[*validator.Validate]().Name, func(do.Injector) (*validator.Validate, error) {
		return New()
	})
}

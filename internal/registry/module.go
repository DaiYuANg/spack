package registry

import (
	"context"
	"go.uber.org/fx"
)

var Module = fx.Module("registry", fx.Provide(newRegistry, newContext))

func newRegistry() (*InMemoryRegistry, error) {
	return NewInMemoryRegistry(), nil

}

func newContext() context.Context {
	return context.Background()
}

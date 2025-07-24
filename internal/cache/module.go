package cache

import (
	"context"
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	ristrettostore "github.com/eko/gocache/store/ristretto/v4"
	"go.uber.org/fx"
)

var Module = fx.Module("cache", fx.Provide(newCache, newContext, newStaticBuffer))

func newCache() *cache.Cache[string] {
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	ristrettoStore := ristrettostore.NewRistretto(ristrettoCache)

	return cache.New[string](ristrettoStore)
}

func newContext() context.Context {
	return context.Background()
}

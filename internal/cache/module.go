package cache

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/daiyuang/spack/internal/config"
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	bigcachestore "github.com/eko/gocache/store/bigcache/v4"
	redisstore "github.com/eko/gocache/store/redis/v4"
	ristrettostore "github.com/eko/gocache/store/ristretto/v4"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

var Module = fx.Module("cache",
	fx.Provide(
		newContext,
		newCache,
	),
)

func newCache(config *config.Config, ctx context.Context) *cache.Cache[string] {
	cacheConfig := config.Cache
	var s store.StoreInterface
	switch cacheConfig.Type {
	case "bigCache":
		bigcacheClient, _ := bigcache.New(ctx, bigcache.DefaultConfig(5*time.Minute))
		s = bigcachestore.NewBigcache(bigcacheClient)
		break
	case "redis":
		s = redisstore.NewRedis(redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		}))
	default:
	case "ristretto":
		ristrettoCache := lo.Must(ristretto.NewCache(&ristretto.Config{
			NumCounters: 1000,
			MaxCost:     100,
			BufferItems: 64,
		}))
		s = ristrettostore.NewRistretto(ristrettoCache)
		break
	}

	return cache.New[string](s)
}

func newContext() context.Context {
	return context.Background()
}

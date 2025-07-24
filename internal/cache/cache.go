package cache

import (
	"context"
	"github.com/eko/gocache/lib/v4/cache"
)

type StaticBuffer struct {
	store   *cache.Cache[string]
	context context.Context
}

func newStaticBuffer(store *cache.Cache[string], context context.Context) *StaticBuffer {
	return &StaticBuffer{store: store, context: context}
}

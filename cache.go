package login_sdk_go

import (
	"github.com/patrickmn/go-cache"
	"time"
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

type DefaultCache struct {
	cache *cache.Cache
}

func NewDefaultCache(expirationTime time.Duration) DefaultCache {
	return DefaultCache{
		cache.New(expirationTime, expirationTime),
	}
}

func (dc DefaultCache) Get(key string) (interface{}, bool) {
	return dc.cache.Get(key)
}

func (dc DefaultCache) Set(key string, values interface{}) {
	dc.cache.Set(key, values, cache.DefaultExpiration)
}

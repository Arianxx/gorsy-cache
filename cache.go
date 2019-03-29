package gorsy_cache

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// cacheCollected collects the specific cache object constructor.
var cacheCollected = make(map[string]func(int) interface{})

// Cache represents a interface used by the user to r/w the cache store.
type Cache interface {
	init()

	Get(key interface{}) (interface{}, error)
	Set(key, value interface{}) error
	SetWithExpire(key, value interface{}, duration time.Duration) error
	Has(key interface{}) bool
	Remove(key interface{}) bool
	Keys() []interface{}
	CleanExpired() int
	Flush()
	Len() int
}

// baseCache provides a set of common attributes. A specific cache implementation is required to inherit it.
type baseCache struct {
	sync.RWMutex
	size int
	// expiration provides a global expiration information for a cache store.
	Expiration time.Duration
	LoaderFunc
	SerializerFunc
	DeserializerFunc
	BeforeEvictedFunc
}

const (
	NoExpiration      = -1
	DefaultExpiration = 0
)

type (
	LoaderFunc func(key interface{}) (interface{}, error)
	SerializerFunc func(key, value interface{}) (interface{}, error)
	DeserializerFunc func(key, value interface{}) (interface{}, error)
	BeforeEvictedFunc func(key, value interface{})
)

// cacheBuilder used to build a specific cache.
type cacheBuilder struct {
	cache interface{}
}

func New(name string, size int) (*cacheBuilder, error) {
	f, ok := cacheCollected[name]
	if !ok {
		return nil, fmt.Errorf("no specific cache was found")
	}
	builder := &cacheBuilder{f(size)}
	if err := builder.checkCacheValid(); err != nil {
		return nil, fmt.Errorf("specific cache invalid: %s", err.Error())
	}

	return builder, nil
}

func (c *cacheBuilder) Build() Cache {
	cache := c.cache.(Cache)
	cache.init()
	return cache
}

func (c *cacheBuilder) SetDefaultExpiration(t time.Duration) *cacheBuilder {
	p := c.getFieldAddrInterface("Expiration").(*time.Duration)
	*p = t
	return c
}

func (c *cacheBuilder) SetLoaderFunc(f LoaderFunc) *cacheBuilder {
	p := c.getFieldAddrInterface("LoaderFunc").(*LoaderFunc)
	*p = f
	return c
}

func (c *cacheBuilder) getFieldAddrInterface(name string) interface{} {
	v := reflect.ValueOf(c.cache).Elem()
	f := v.FieldByName(name)
	return f.Addr().Interface()
}

func (c *cacheBuilder) checkCacheValid() error {
	if !c.implementedCache() {
		return fmt.Errorf("cache has not implement the Cache interface")
	}
	if !c.inheritedBaseCache() {
		return fmt.Errorf("cache has not inherited baseCache")
	}
	return nil
}

func (c *cacheBuilder) implementedCache() bool {
	_, ok := c.cache.(Cache)
	return ok
}

func (c *cacheBuilder) inheritedBaseCache() bool {
	field, ok := reflect.ValueOf(c.cache).Elem().Type().FieldByName("baseCache")
	if !ok || fmt.Sprint(field.Type.Name()) != "baseCache" {
		return false
	}
	return true
}

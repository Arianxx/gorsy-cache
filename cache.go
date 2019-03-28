package gorsy_cache

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// cacheCollected collects the specific cache implementation.
var cacheCollected = make(map[string]func() interface{})

// Cache represents a interface used by the user to r/w the cache store.
type Cache interface {
	init()

	Get(key interface{}) (interface{}, error)
	Set(key, value interface{}) error
	SetWithExpire(key, value interface{}, duration time.Duration) error
	Has(key interface{}) bool
	Remove(key interface{}) bool
	Keys() []interface{}
	Flush()
	Len() int
}

// baseCache provides a set of common attributes. A specific cache implementation is required to inherit it.
type baseCache struct {
	sync.RWMutex
	size int
	// expiration provides a global expiration information for a cache store.
	expiration time.Duration

	loaderFunc        LoaderFunc
	serializerFunc    SerializerFunc
	deserializerFunc  DeserializerFunc
	beforeEvictedFunc BeforeEvictedFunc
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

func New(name string) (*cacheBuilder, error) {
	f, ok := cacheCollected[name]
	if !ok {
		return nil, fmt.Errorf("no specific cache was found")
	}
	cache := f()

	builder := &cacheBuilder{cache}
	if err := builder.checkCacheValid(); err != nil {
		return nil, fmt.Errorf("specific cache invalid: %s", err.Error())
	}

	return builder, nil
}

func (c *cacheBuilder) Build() Cache {
	reflect.ValueOf(c.cache).MethodByName("init").Call(nil)
	return c.cache.(Cache)
}

func (c *cacheBuilder) SetDefaultExpiration(t time.Duration) *cacheBuilder {
	p := c.getFieldAddrInterface("expiration").(*time.Duration)
	*p = t
	return c
}

func (c *cacheBuilder) SetLoaderFunc(f LoaderFunc) *cacheBuilder {
	p := c.getFieldAddrInterface("loaderFunc").(*LoaderFunc)
	*p = f
	return c
}

func (c *cacheBuilder) getFieldAddrInterface(name string) interface{} {
	v := reflect.ValueOf(c.cache)
	f := v.FieldByName(name)
	return f.Addr().Interface()
}

func (c *cacheBuilder) checkCacheValid() error {
	if !c.implementedCache() {
		return fmt.Errorf("cache has not implement the Cache interfac")
	} else if !c.inheritedBaseCache() {
		return fmt.Errorf("cache has not inherited baseCache")
	}
	return nil
}

func (c *cacheBuilder) implementedCache() bool {
	_, ok := c.cache.(Cache)
	return ok
}

func (c *cacheBuilder) inheritedBaseCache() bool {
	field, ok := reflect.TypeOf(c.cache).FieldByName("baseCache")
	if !ok || fmt.Sprint(field.Type.Kind()) != "baseCache" {
		return false
	}

	return true
}

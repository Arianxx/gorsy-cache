package gorsy_cache

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

const (
	NoExpiration      = -1
	DefaultExpiration = 0
)

// cacheCollected collects the specific cache object constructor.
var cacheCollected = make(map[string]func() interface{})

// cacheCounter distinguish anonymous cache store
var cacheCounter int

// Cache represents a interface used by the user to r/w the cache store.
type Cache interface {
	getBaseCache() *baseCache

	Init()
	Get(key interface{}) (interface{}, error)
	GetOnlyPresent(key interface{}) (interface{}, bool)
	Set(key, value interface{})
	SetWithExpire(key, value interface{}, duration time.Duration)
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
	Name string
	// expiration provides a global expiration information for a cache store.
	Expiration time.Duration
	LoaderFunc
	BeforeEvictedFunc
}

type (
	LoaderFunc func(key interface{}) (interface{}, error)
	BeforeEvictedFunc func(key, value interface{})
)

// cacheBuilder used to build a specific cache.
type cacheBuilder struct {
	cache Cache
	bc    *baseCache
}

// NewBuilder receive a constant cache name and a cache size, return a specific cache builder.
// A error will be returned if the specific cache is not registered or the cache implementation is invalid.
func NewBuilder(name string, size int) (*cacheBuilder, error) {
	f, ok := cacheCollected[name]
	if !ok {
		return nil, fmt.Errorf("no specific cache was found")
	}

	c := f()
	if err := checkCacheValid(c); err != nil {
		return nil, fmt.Errorf("specific cache invalid: %s", err.Error())
	}

	builder := &cacheBuilder{cache: c.(Cache)}
	builder.bc = builder.cache.getBaseCache()
	builder.bc.size = size
	return builder, nil
}

// Build a cache store by the previous setups.
// Typically it will perform some tasks such as allocating cache space.
func (c *cacheBuilder) Build() Cache {
	if c.bc.Name == "" {
		c.bc.Name = fmt.Sprintf("cache: %d", cacheCounter)
		cacheCounter++
	}
	if c.bc.Expiration == DefaultExpiration {
		c.bc.Expiration = 60
	}
	c.cache.Init()
	return c.cache
}

func (c *cacheBuilder) SetName(n string) *cacheBuilder {
	c.bc.Name = n
	return c
}

func (c *cacheBuilder) SetDefaultExpiration(t time.Duration) *cacheBuilder {
	c.bc.Expiration = t
	return c
}

func (c *cacheBuilder) SetLoaderFunc(f LoaderFunc) *cacheBuilder {
	c.bc.LoaderFunc = f
	return c
}

func (c *cacheBuilder) SetBeforeEvictedFunc(f BeforeEvictedFunc) *cacheBuilder {
	c.bc.BeforeEvictedFunc = f
	return c
}

func checkCacheValid(c interface{}) error {
	if !implementedCache(c) {
		return fmt.Errorf("cache has not implement the Cache interface")
	}
	if !inheritedBaseCache(c) {
		return fmt.Errorf("cache has not inherited baseCache")
	}
	return nil
}

func implementedCache(c interface{}) bool {
	_, ok := c.(Cache)
	return ok
}

func inheritedBaseCache(c interface{}) bool {
	field, ok := reflect.ValueOf(c).Elem().Type().FieldByName("baseCache")
	if !ok || fmt.Sprint(field.Type.Name()) != "baseCache" {
		return false
	}
	return true
}

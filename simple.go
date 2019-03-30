package gorsy_cache

import (
	"time"
)

func init() {
	cacheCollected[SIMPLE] = func() interface{} {
		return &simpleCache{}
	}
}

const (
	SIMPLE = "simpleCache"
)

type simpleCache struct {
	baseCache
	items map[interface{}]*simpleItem
}

func (c *simpleCache) getBaseCache() *baseCache {
	return &c.baseCache
}

func (c *simpleCache) Init() {
	c.items = make(map[interface{}]*simpleItem, c.size)
}

func (c *simpleCache) Get(key interface{}) (interface{}, error) {
	c.RLock()
	defer c.RUnlock()

	return c.get(key, true)
}

func (c *simpleCache) GetOnlyPresent(key interface{}) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()

	v, err := c.get(key, false)
	if err != nil {
		return nil, false
	} else {
		return v, true
	}
}

func (c *simpleCache) get(key interface{}, fromLoader bool) (interface{}, error) {
	item, ok := c.items[key]
	if ok && !item.isExpired() {
		return item.value, nil
	}

	if fromLoader {
		return c.getFromLoader(key)
	} else {
		return nil, &KeyNotFoundError{}
	}
}

func (c *simpleCache) getFromLoader(key interface{}) (interface{}, error) {
	var v interface{}
	var err error
	if c.LoaderFunc != nil {
		v, err = c.LoaderFunc(key)
		if err != nil {
			return nil, &KeyNotFoundError{c.Name, key, err}
		}
	}

	return v, &KeyNotFoundError{c.Name, key, nil}
}

func (c *simpleCache) Set(key, value interface{}) {
	c.Lock()
	defer c.Unlock()

	c.set(key, value, DefaultExpiration)
}

func (c *simpleCache) set(key, value interface{}, expiration time.Duration) {
	if len(c.items) == c.size {
		c.evict(1)
	}

	item := &simpleItem{baseItem{key, value, nil}}
	item.setExpiration(expiration, &c.baseCache)
	c.items[key] = item
}

func (c *simpleCache) SetWithExpire(key, value interface{}, expiration time.Duration) {
	c.Lock()
	defer c.Unlock()

	c.set(key, value, expiration)
}

func (c *simpleCache) evict(num int) {
	keys := make([]interface{}, 0)
	for k, v := range c.items {
		if v.isExpired() {
			keys = append(keys, k)
		}

		if len(keys) >= num {
			break
		}
	}

	for _, k := range keys {
		c.remove(k)
	}
}

func (c *simpleCache) Has(key interface{}) bool {
	c.RLock()
	defer c.RUnlock()

	item, ok := c.items[key]
	return ok && !item.isExpired()
}

func (c *simpleCache) Remove(key interface{}) bool {
	c.Lock()
	defer c.Unlock()

	return c.remove(key)
}

func (c *simpleCache) remove(key interface{}) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	delete(c.items, key)

	if c.BeforeEvictedFunc != nil {
		c.BeforeEvictedFunc(key, item.value)
	}

	return !item.isExpired()
}

func (c *simpleCache) Keys() []interface{} {
	c.RLock()
	defer c.RUnlock()

	keys := make([]interface{}, 0)
	for k, v := range c.items {
		if !v.isExpired() {
			keys = append(keys, k)
		}
	}

	return keys
}

func (c *simpleCache) CleanExpired() int {
	c.Lock()
	defer c.Unlock()

	expiredKeys := make([]interface{}, 0)
	for k, v := range c.items {
		if v.isExpired() {
			expiredKeys = append(expiredKeys, k)
		}
	}

	for _, k := range expiredKeys {
		c.remove(k)
	}

	return len(expiredKeys)
}

func (c *simpleCache) Flush() {
	c.Lock()
	defer c.Unlock()

	c.Init()
}

func (c *simpleCache) Len() int {
	return len(c.items)
}

type simpleItem struct {
	baseItem
}

package gorsy_cache

import (
	"container/list"
	"time"
)

func init() {
	cacheCollected[LRU] = func() interface{} {
		return &lruCache{}
	}
}

const (
	LRU = "lru"
)

type lruCache struct {
	baseCache
	items map[interface{}]*list.Element
	list  *list.List
}

func (c *lruCache) getBaseCache() *baseCache {
	return &c.baseCache
}

func (c *lruCache) Init() {
	c.items = make(map[interface{}]*list.Element, c.size)
	c.list = list.New()
}

func (c *lruCache) Get(key interface{}) (interface{}, error) {
	c.RLock()
	defer c.RUnlock()

	return c.get(key, true)
}

func (c *lruCache) GetOnlyPresent(key interface{}) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()

	v, err := c.get(key, false)
	if err != nil {
		return nil, false
	} else {
		return v, true
	}
}

func (c *lruCache) get(key interface{}, fromLoader bool) (interface{}, error) {
	item, ok := c.items[key]
	if ok && !item.Value.(*lruItem).isExpired() {
		c.list.PushBack(c.list.Remove(item))
		return item.Value.(*lruItem).value, nil
	}

	if fromLoader {
		return c.getFromLoader(key)
	} else {
		return nil, &KeyNotFoundError{}
	}
}

func (c *lruCache) getFromLoader(key interface{}) (interface{}, error) {
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

func (c *lruCache) Set(key, value interface{}) {
	c.Lock()
	defer c.Unlock()

	c.set(key, value, DefaultExpiration)
}

func (c *lruCache) set(k, v interface{}, e time.Duration) {
	if len(c.items) == c.size {
		c.evict(1)
	}

	item := &lruItem{baseItem{k, v, nil}}
	item.setExpiration(e, &c.baseCache)
	c.items[k] = c.list.PushBack(item)
}

func (c *lruCache) SetWithExpire(k, v interface{}, e time.Duration) {
	c.Lock()
	defer c.Unlock()

	c.set(k, v, e)
}

func (c *lruCache) evict(size int) {
	for i := 0; i < size; i++ {
		e := c.list.Front()
		delete(c.items, e.Value.(*lruItem).key)
		c.list.Remove(e)
	}
}

func (c *lruCache) Has(key interface{}) bool {
	c.RLock()
	defer c.RUnlock()

	item, ok := c.items[key]
	return ok && !item.Value.(*lruItem).isExpired()
}

func (c *lruCache) Remove(key interface{}) bool {
	c.Lock()
	defer c.Unlock()

	return c.remove(key)
}

func (c *lruCache) remove(key interface{}) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	delete(c.items, key)
	c.list.Remove(item)

	if c.BeforeEvictedFunc != nil {
		c.BeforeEvictedFunc(key, item.Value.(*lruItem).value)
	}

	return !item.Value.(*lruItem).isExpired()
}

func (c *lruCache) Keys() []interface{} {
	c.RLock()
	defer c.RUnlock()

	keys := make([]interface{}, 0)
	for k, v := range c.items {
		if !v.Value.(*lruItem).isExpired() {
			keys = append(keys, k)
		}
	}

	return keys
}

func (c *lruCache) CleanExpired() int {
	c.Lock()
	defer c.Unlock()

	expiredKey := make([]interface{}, 0)
	for k, v := range c.items {
		if v.Value.(*lruItem).isExpired() {
			expiredKey = append(expiredKey, k)
		}
	}

	for _, k := range expiredKey {
		c.list.Remove(c.items[k])
		delete(c.items, k)
	}

	return len(expiredKey)
}

func (c *lruCache) Flush() {
	c.Lock()
	defer c.Unlock()

	c.Init()
}

func (c *lruCache) Len() int {
	return len(c.items)
}

type lruItem struct {
	baseItem
}

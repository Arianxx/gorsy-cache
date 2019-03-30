package gorsy_cache

import (
	"container/heap"
	"time"
)

func init() {
	cacheCollected[LFU] = func() interface{} {
		return &lfuCache{}
	}
}

const (
	LFU = "lfu"
)

type lfuCache struct {
	baseCache
	items map[interface{}]*lfuItem
	heap  lfuHeap
}

func (c *lfuCache) getBaseCache() *baseCache {
	return &c.baseCache
}

func (c *lfuCache) Init() {
	c.items = make(map[interface{}]*lfuItem, c.size)
	c.heap = make(lfuHeap, 0)
}

func (c *lfuCache) Get(key interface{}) (interface{}, error) {
	c.RLock()
	defer c.RUnlock()

	return c.get(key, true)
}

func (c *lfuCache) GetOnlyPresent(key interface{}) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()

	v, err := c.get(key, false)
	if err != nil {
		return nil, false
	} else {
		return v, true
	}
}

func (c *lfuCache) get(key interface{}, fromLoader bool) (interface{}, error) {
	item, ok := c.items[key]
	if ok && !item.isExpired() {
		item.freq++
		heap.Fix(&c.heap, item.index)
		return item.value, nil
	}

	if fromLoader {
		return c.getFromLoader(key)
	} else {
		return nil, &KeyNotFoundError{}
	}
}

func (c *lfuCache) getFromLoader(key interface{}) (interface{}, error) {
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

func (c *lfuCache) Set(key, value interface{}) {
	c.Lock()
	defer c.Unlock()

	c.set(key, value, DefaultExpiration)
}

func (c *lfuCache) set(k, v interface{}, e time.Duration) {
	ele, ok := c.items[k]
	if ok {
		ele.value = v
		ele.setExpiration(e, &c.baseCache)
		return
	}

	if len(c.items) == c.size {
		c.evict(1)
	}

	item := &lfuItem{baseItem{k, v, nil}, 0, 0}
	item.setExpiration(e, &c.baseCache)
	heap.Push(&c.heap, item)
	c.items[k] = item
}

func (c *lfuCache) SetWithExpire(k, v interface{}, e time.Duration) {
	c.Lock()
	defer c.Unlock()

	c.set(k, v, e)
}

func (c *lfuCache) evict(size int) {
	for i := 0; i < size; i++ {
		e := heap.Pop(&c.heap)
		delete(c.items, e.(*lruItem).key)
	}
}

func (c *lfuCache) Remove(key interface{}) bool {
	c.Lock()
	defer c.Unlock()

	return c.remove(key)
}

func (c *lfuCache) remove(key interface{}) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	heap.Remove(&c.heap, item.index)
	delete(c.items, key)

	if c.BeforeEvictedFunc != nil {
		c.BeforeEvictedFunc(key, item.value)
	}

	return !item.isExpired()
}

func (c *lfuCache) Keys() []interface{} {
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

func (c *lfuCache) CleanExpired() int {
	c.Lock()
	defer c.Unlock()

	expiredKey := make([]interface{}, 0)
	for k, v := range c.items {
		if v.isExpired() {
			expiredKey = append(expiredKey, k)
		}
	}

	for _, k := range expiredKey {
		c.remove(k)
	}

	return len(expiredKey)
}

func (c *lfuCache) Flush() {
	c.Lock()
	defer c.Unlock()

	c.Init()
}

func (c *lfuCache) Len() int {
	return len(c.items)
}

func (c *lfuCache) Has(key interface{}) bool {
	c.RLock()
	defer c.RUnlock()

	item, ok := c.items[key]
	return ok && !item.isExpired()
}

type lfuItem struct {
	baseItem
	index, freq int
}

type lfuHeap []*lfuItem

func (l lfuHeap) Len() int           { return len(l) }
func (l lfuHeap) Less(i, j int) bool { return l[i].freq < l[j].freq }
func (l lfuHeap) Swap(i, j int) {
	l[i].index, l[j].index = j, i
	l[i], l[j] = l[j], l[i]
}

func (l *lfuHeap) Push(x interface{}) {
	*l = append(*l, x.(*lfuItem))
	x.(*lfuItem).index = len(*l) - 1
}

func (l *lfuHeap) Pop() interface{} {
	x, n := (*l)[len(*l)-1], (*l)[:len(*l)-1]
	*l = n
	return x
}

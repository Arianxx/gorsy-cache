package gorsy_cache

import (
	"container/list"
	"time"
)

func init() {
	cacheCollected[ARC] = func() interface{} {
		return &arcCache{}
	}
}

const (
	ARC = "arc"
)

type arcCache struct {
	baseCache
	// size of t1
	part  int
	items map[interface{}]*arcItem

	t1 *arcList
	t2 *arcList
	b1 *arcList
	b2 *arcList
}

func (c *arcCache) getBaseCache() *baseCache {
	return &c.baseCache
}

func (c *arcCache) Init() {
	c.part = c.size / 2
	c.items = make(map[interface{}]*arcItem, c.size)
	c.t1 = newArcList()
	c.t2 = newArcList()
	c.b1 = newArcList()
	c.b2 = newArcList()
}

func (c *arcCache) Replace(key interface{}) {
	var old *arcItem
	if c.t1.Len() != 0 && ((c.b2.Has(key) && c.t1.Len() == c.part) || (c.t1.Len() > c.part)) {
		old = c.t1.Pop()
		c.b1.Push(old)
	} else {
		old = c.t2.Pop()
		c.b2.Push(old)
	}

	if old != nil {
		delete(c.items, old.key)
	}
}

func (c *arcCache) Get(key interface{}) (interface{}, error) {
	c.Lock()
	defer c.Unlock()

	return c.get(key, true)
}

func (c *arcCache) GetOnlyPresent(key interface{}) (interface{}, bool) {
	c.Lock()
	defer c.Unlock()

	v, err := c.get(key, false)
	if err != nil {
		return nil, false
	} else {
		return v, true
	}
}

func (c *arcCache) get(key interface{}, fromLoader bool) (interface{}, error) {
	item, ok := c.items[key]
	if ok && !item.isExpired() {
		if c.t1.Has(key) {
			c.t1.MoveFront(key)
		} else {
			c.t2.MoveFront(key)
		}
		return item.value, nil
	}

	if fromLoader {
		return c.getFromLoader(key)
	} else {
		return nil, &KeyNotFoundError{}
	}
}

func (c *arcCache) Set(key, value interface{}) {
	c.Lock()
	defer c.Unlock()

	c.set(key, value, DefaultExpiration)
}

func (c *arcCache) set(k, v interface{}, e time.Duration) {
	item, ok := c.items[k]
	if ok {
		item.value = v
		item.setExpiration(e, &c.baseCache)
		return
	}

	item = &arcItem{baseItem{k, v, nil}}
	item.setExpiration(e, &c.baseCache)
	c.items[k] = item

	if c.b1.Has(k) {
		c.part = min(c.size, c.part+max(
			c.b2.Len()/c.b1.Len(), 1,
		))
		c.Replace(k)
		c.b1.Remove(k)
		c.t2.Push(item)
		c.t2.MoveFront(item.key)
		return
	} else if c.b2.Has(k) {
		c.part = min(c.size, c.part-max(
			c.b2.Len()/c.b1.Len(), 1,
		))
		c.Replace(k)
		c.b2.Remove(k)
		c.t2.Push(item)
		c.t2.MoveFront(item)
		return
	} else if c.t1.Len()+c.b1.Len() == c.size {
		if c.t1.Len() < c.size {
			c.b1.Pop()
			c.Replace(k)
		} else {
			e := c.t1.Pop()
			delete(c.items, e.key)
		}
	} else {
		total := c.t1.Len() + c.b1.Len() + c.t2.Len() + c.b2.Len()
		if total >= (2 * c.size) {
			c.b2.Pop()
		}

		c.Replace(k)
	}

	c.t1.Push(item)
	c.t1.MoveFront(item.key)
}

func (c *arcCache) SetWithExpire(k, v interface{}, e time.Duration) {
	c.Lock()
	defer c.Unlock()

	c.set(k, v, e)
}

func (c *arcCache) Has(key interface{}) bool {
	c.RLock()
	defer c.RUnlock()

	item, ok := c.items[key]
	return ok && !item.isExpired()
}

func (c *arcCache) Remove(key interface{}) bool {
	c.Lock()
	defer c.Unlock()

	return c.remove(key)
}

func (c *arcCache) remove(key interface{}) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	delete(c.items, key)
	c.t1.Remove(item)
	c.t2.Remove(item)

	if c.BeforeEvictedFunc != nil {
		c.BeforeEvictedFunc(key, item.value)
	}

	return !item.isExpired()
}

func (c *arcCache) Keys() []interface{} {
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

func (c *arcCache) CleanExpired() int {
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

func (c *arcCache) Flush() {
	c.Lock()
	defer c.Unlock()

	c.Init()
}

func (c *arcCache) Len() int {
	return len(c.items)
}

func (c *arcCache) getFromLoader(key interface{}) (interface{}, error) {
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

type arcItem struct {
	baseItem
}

type arcList struct {
	l *list.List
	m map[interface{}]*list.Element
}

func newArcList() *arcList {
	return &arcList{
		list.New(),
		make(map[interface{}]*list.Element),
	}
}

func (a *arcList) Has(key interface{}) bool {
	_, ok := a.m[key]
	return ok
}

func (a *arcList) MoveFront(key interface{}) bool {
	item, ok := a.m[key]
	if !ok {
		return false
	}
	a.l.MoveToFront(item)
	return true
}

func (a *arcList) Len() int {
	return a.l.Len()
}

func (a *arcList) Pop() *arcItem {
	if len(a.m) == 0 {
		return nil
	}

	old := a.l.Remove(a.l.Back()).(*arcItem)
	delete(a.m, old.key)
	return old
}

func (a *arcList) Push(i *arcItem) {
	if i == nil {
		return
	}

	e := a.l.PushFront(i)
	a.m[i.key] = e
}

func (a *arcList) Remove(k interface{}) bool {
	e, ok := a.m[k]
	if !ok {
		return false
	}
	a.l.Remove(e)
	delete(a.m, e.Value.(*arcItem).key)
	return true
}

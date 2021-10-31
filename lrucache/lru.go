package lrucache

import (
	"container/list"
	"errors"
)

// LRU struct to represent the LRU cache
type LRUCache struct {
	size      int
	evictList *list.List
	items     map[interface{}]*list.Element
}

// Entry struct containing key value pair to represent a cache entry
type cacheEntry struct {
	key   interface{}
	value interface{}
}

// New constructs a new cache instance
func NewLRUCache(size int) (*LRUCache, error) {
	// no use for a cache with one entry
	if size <= 1 {
		return nil, errors.New("cache size must be bigger than 1")
	}

	c := &LRUCache{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element),
	}

	return c, nil
}

func (c *LRUCache) Add(key, value interface{}) (evicted bool) {
	// check for existing entry
	if e, ok := c.items[key]; ok {
		c.evictList.MoveToFront(e)
		e.Value.(*cacheEntry).value = value

		return false
	}

	// add the new entry
	ent := &cacheEntry{key, value}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry

	// remove last element if number of entries exceed limit
	evict := c.evictList.Len() > c.size
	if evict {
		c.removeOldest()
	}

	return evict
}

func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool) {
	e, ok := c.items[key]

	if ok {
		// update recent-ness
		c.evictList.MoveToFront(e)

		if e.Value.(*cacheEntry) == nil {
			return nil, false
		}

		return e.Value.(*cacheEntry).value, true
	}

	return
}

func (c *LRUCache) Contains(key interface{}) (ok bool) {
	_, ok = c.items[key]

	return ok
}

func (c *LRUCache) Remove(key interface{}) (present bool) {
	e, ok := c.items[key]

	if ok {
		c.removeElement(e)

		return true
	}

	return false
}

func (c *LRUCache) Keys() []interface{} {
	keys := make([]interface{}, len(c.items))

	i := 0
	for e := c.evictList.Front(); e != nil; e = e.Next() {
		keys[i] = e.Value.(*cacheEntry).key
		i++
	}

	return keys
}

func (c *LRUCache) Len() int {
	return c.evictList.Len()
}

func (c *LRUCache) Purge() {
	for k := range c.items {
		delete(c.items, k)
	}

	c.evictList.Init()
}

func (c *LRUCache) removeOldest() {
	e := c.evictList.Back()

	if e != nil {
		c.removeElement(e)
	}
}

func (c *LRUCache) removeElement(entry *list.Element) {
	c.evictList.Remove(entry)

	e := entry.Value.(*cacheEntry)
	delete(c.items, e.key)
}

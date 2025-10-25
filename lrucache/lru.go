// Package lrucache provides a very basic LRU cache implementation.
package lrucache

import (
	"container/list"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// LRU struct to represent the LRU cache
type LRUCache struct {
	lock      sync.RWMutex
	size      int
	evictList *list.List
	items     map[interface{}]*list.Element
}

// Entry struct containing key value pair to represent a cache entry
type cacheEntry struct {
	key   interface{}
	value interface{}
}

// on-disk shape (MRU -> LRU)
type kv struct {
	K interface{}
	V interface{}
}
type onDisk struct {
	Size    int
	Entries []kv
}

// New constructs a new cache instance
func NewLRUCache(size int) (*LRUCache, error) {
	// no use for a cache with one entry
	if size <= 1 {
		return nil, errors.New("cache size must be bigger than 1")
	}
	return &LRUCache{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element),
	}, nil
}

func (c *LRUCache) Add(key, value interface{}) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// update existing
	if e, ok := c.items[key]; ok {
		c.evictList.MoveToFront(e)
		e.Value.(*cacheEntry).value = value
		return false
	}

	// add new at front (MRU)
	ent := &cacheEntry{key, value}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry

	// evict if needed
	if c.evictList.Len() > c.size {
		c.removeOldest()
		return true
	}
	return false
}

func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	e, ok := c.items[key]
	if !ok {
		return nil, false
	}

	// move to MRU
	c.evictList.MoveToFront(e)

	if e.Value.(*cacheEntry) == nil {
		return nil, false
	}
	return e.Value.(*cacheEntry).value, true
}

func (c *LRUCache) Contains(key interface{}) (ok bool) {
	c.lock.RLock()
	_, ok = c.items[key]
	c.lock.RUnlock()
	return ok
}

func (c *LRUCache) Remove(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	e, ok := c.items[key]
	if !ok {
		return false
	}
	c.removeElement(e)
	return true
}

// Keys returns keys in MRU -> LRU order (does not change recency).
func (c *LRUCache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]interface{}, 0, len(c.items))
	for e := c.evictList.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*cacheEntry).key)
	}
	return keys
}

func (c *LRUCache) Length() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.evictList.Len()
}

func (c *LRUCache) Purge() {
	c.lock.Lock()
	for k := range c.items {
		delete(c.items, k)
	}
	c.evictList.Init()
	c.lock.Unlock()
}

func (c *LRUCache) removeOldest() {
	if e := c.evictList.Back(); e != nil {
		c.removeElement(e)
	}
}

func (c *LRUCache) removeElement(entry *list.Element) {
	c.evictList.Remove(entry)
	e := entry.Value.(*cacheEntry)
	delete(c.items, e.key)
}

// Pair is a serializable key/value used for snapshots and export.
type Pair struct {
	Key   interface{}
	Value interface{}
}

// Snapshot returns a copy of the cache contents in MRU -> LRU order,
// plus the configured size. It does NOT change recency.
func (c *LRUCache) Snapshot() (size int, entries []Pair) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	size = c.size
	entries = make([]Pair, 0, c.evictList.Len())
	for e := c.evictList.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*cacheEntry)
		entries = append(entries, Pair{Key: ent.key, Value: ent.value})
	}
	return
}

// Export writes size + entries (MRU -> LRU) in gob format WITHOUT
// holding locks during encoding.
func (c *LRUCache) Export(w io.Writer) error {
	size, pairs := c.Snapshot() // short RLock inside

	data := onDisk{
		Size:    size,
		Entries: make([]kv, len(pairs)),
	}
	for i, p := range pairs {
		data.Entries[i] = kv{K: p.Key, V: p.Value}
	}
	return gob.NewEncoder(w).Encode(&data)
}

// Import replaces the cache contents, preserving LRU order.
// Assumes Entries are MRU -> LRU (same as Export).
func (c *LRUCache) Import(r io.Reader) error {
	var data onDisk
	if err := gob.NewDecoder(r).Decode(&data); err != nil {
		return err
	}
	if data.Size <= 1 {
		return errors.New("invalid cache size in import")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.size = data.Size
	c.items = make(map[interface{}]*list.Element, len(data.Entries))
	c.evictList.Init()

	// Rebuild: PushBack in MRU -> LRU order keeps MRU at Front, LRU at Back.
	for _, p := range data.Entries {
		ent := &cacheEntry{key: p.K, value: p.V}
		el := c.evictList.PushBack(ent)
		c.items[p.K] = el
	}

	for c.evictList.Len() > c.size {
		c.removeOldest()
	}
	return nil
}

// ExportToFile writes (non-atomic) to a file path.
func (c *LRUCache) ExportToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Export(f)
}

// ExportToFileAtomic writes to a temp file and atomically renames it.
func (c *LRUCache) ExportToFileAtomic(path string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "lru-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		// cleanup if still present
		_ = os.Remove(tmpPath)
	}()

	if err := c.Export(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// atomic on same filesystem
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}

func (c *LRUCache) ImportFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Import(f)
}

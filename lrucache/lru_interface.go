// The lru package provides a very basic LRU cache implementation
package lrucache

// Cache defines the interface for the LRU cache
type Cache interface {
	// Add a new value to the cache and updates the recent-ness. Returns true if an eviction occurred.
	Add(key, value interface{}) bool

	// Return a key's value if found in the cache and updates the recent-ness.
	Get(key interface{}) (value interface{}, ok bool)

	// Check if a key exists without updating the recent-ness.
	Contains(key interface{}) (ok bool)

	// Remove a key from the cache.
	Remove(key interface{}) bool

	// Return a slice which all keys ordered by newest to oldest.
	Keys() []interface{}

	// Return number of entry in the cache.
	Length() int

	// Remove all entries form the cache.
	Purge()
}

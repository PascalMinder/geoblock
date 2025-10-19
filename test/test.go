package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"time"

	"github.com/PascalMinder/geoblock/lrucache"
)

// ipEntry represents the value type we want to cache.
type ipEntry struct {
	Country   string
	Timestamp time.Time
}

func init() {
	// Register types used in the cache for gob serialization
	gob.Register(ipEntry{})
	gob.Register("") // string keys are basic, but harmless to register
}

func main() {
	cache, err := lrucache.NewLRUCache(5)
	if err != nil {
		log.Fatal(err)
	}

	// Add some example entries
	cache.Add("82.220.110.18", ipEntry{Country: "CH", Timestamp: time.Now()})
	cache.Add("99.220.109.148", ipEntry{Country: "CA", Timestamp: time.Now().Add(-time.Hour)})
	cache.Add("203.0.113.7", ipEntry{Country: "US", Timestamp: time.Now().Add(-2 * time.Hour)})

	fmt.Println("== Original Cache ==")
	printCache(cache)

	// Export to disk
	file := "cache.gob"
	if err := cache.ExportToFile(file); err != nil {
		log.Fatalf("Export failed: %v", err)
	}
	fmt.Printf("Cache exported to %s\n", file)

	// Purge the cache (simulate a fresh start)
	cache.Purge()
	fmt.Printf("Cache purged. Length now: %d\n", cache.Length())

	// Import from disk
	if err := cache.ImportFromFile(file); err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	fmt.Println("== Imported Cache ==")
	printCache(cache)
}

// printCache iterates keys in LRU order (most recent → least recent)
func printCache(c *lrucache.LRUCache) {
	for _, key := range c.Keys() {
		if val, ok := c.Get(key); ok {
			entry := val.(ipEntry)
			fmt.Printf("%-15s | Country=%s | Timestamp=%s\n", key, entry.Country, entry.Timestamp.Format(time.RFC3339))
		}
	}
	fmt.Println()
}

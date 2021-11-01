package lrucache

import (
	"fmt"
	"testing"
)

func TestNewLRUCache(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	if cache.Length() != 0 {
		t.Errorf("NewLRUCache() length = %v, want %v", cache.Length(), 0)
	}
}

func TestNewLRUCacheInvalidSize(t *testing.T) {
	expectedError := "cache size must be bigger than 1"
	_, err := NewLRUCache(1)

	if err.Error() != expectedError {
		t.Errorf("NewLRUCache() length = %v, want %v", err.Error(), expectedError)
	}
}

func TestLRUCacheAddElement(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	cache.Add("Apple", 2.20)

	if cache.Length() != 1 {
		t.Errorf("NewLRUCache() error = %v, want %v", cache.Length(), 1)
	}
}

func TestLRUCacheAddElementEviction(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	for i := 0; i < 11; i++ {
		cache.Add("Apple_"+fmt.Sprint(i), 2.0+(0.1*float32(i)))
	}

	if cache.Length() != 10 {
		t.Errorf("NewLRUCache() error = %v, want %v", cache.Length(), 1)
	}
}

func TestLRUCacheAddElementExisting(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	cache.Add("Apple", 2.20)

	e, ok := cache.Get("Apple")

	if ok != true {
		t.Errorf("NewLRUCache() existing element = %v, want %v", ok, true)
	}

	if e != 2.20 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", e, 2.20)
	}

	evicted := cache.Add("Apple", 2.50)

	if evicted != false {
		t.Errorf("NewLRUCache() existing element = %v, want %v", evicted, true)
	}

	e, ok = cache.Get("Apple")

	if ok != true {
		t.Errorf("NewLRUCache() existing element = %v, want %v", ok, true)
	}

	if e != 2.50 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", e, 2.50)
	}
}

func TestLRUCacheGetElementNotExisting(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	cache.Add("Apple", 2.20)

	e, ok := cache.Get("Pear")

	if ok != false {
		t.Errorf("NewLRUCache() existing element = %v, want %v", ok, true)
	}

	if e != nil {
		t.Errorf("NewLRUCache() element = %v, want %v", e, nil)
	}
}

func TestLRUCacheContainsElement(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	cache.Add("Apple", 2.20)

	ok := cache.Contains("Pear")

	if ok != false {
		t.Errorf("NewLRUCache() existing element = %v, want %v", ok, true)
	}

	ok = cache.Contains("Apple")

	if ok != true {
		t.Errorf("NewLRUCache() element = %v, want %v", ok, true)
	}
}

func TestLRUCacheRemove(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	cache.Add("Apple", 2.20)
	cache.Add("Pear", 3.20)

	ok := cache.Contains("Apple")

	if ok != true {
		t.Errorf("NewLRUCache() existing element = %v, want %v", ok, true)
	}

	ok = cache.Contains("Pear")

	if ok != true {
		t.Errorf("NewLRUCache() element = %v, want %v", ok, true)
	}

	cache.Remove("Apple")

	ok = cache.Contains("Apple")

	if ok != false {
		t.Errorf("NewLRUCache() existing element = %v, want %v", ok, true)
	}

	ok = cache.Contains("Pear")

	if ok != true {
		t.Errorf("NewLRUCache() element = %v, want %v", ok, true)
	}
}

func TestLRUCacheKeys(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	for i := 0; i < 15; i++ {
		cache.Add("Apple_"+fmt.Sprint(i), 2.0+(0.1*float32(i)))
	}

	len := cache.Length()

	if len != 10 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", len, 10)
	}

	keys := cache.Keys()

	for i := 0; i < 10; i++ {
		want := "Apple_" + fmt.Sprint(14-i)
		if keys[i] != want {
			t.Errorf("NewLRUCache() existing element = %v, want %v", keys[i], want)
		}
	}
}

func TestLRUCachePurge(t *testing.T) {
	cache, err := NewLRUCache(10)

	if err != nil {
		t.Errorf("NewLRUCache() error = %v, want %v", err, nil)
	}

	for i := 0; i < 15; i++ {
		cache.Add("Apple_"+fmt.Sprint(i), 2.0+(0.1*float32(i)))
	}

	len := cache.Length()

	if len != 10 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", len, 10)
	}

	cache.Purge()

	len = cache.Length()

	if len != 0 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", len, 10)
	}
}

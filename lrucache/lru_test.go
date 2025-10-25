package lrucache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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

	if cacheLen := cache.Length(); cacheLen != 10 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", cacheLen, 10)
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

	if cacheLen := cache.Length(); cacheLen != 10 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", cacheLen, 10)
	}

	cache.Purge()

	if cacheLen := cache.Length(); cacheLen != 0 {
		t.Errorf("NewLRUCache() existing element = %v, want %v", cacheLen, 10)
	}
}

// userData is a lightweight stand-in value type to test gob export/import.
type userData struct {
	Name      string
	LastLogin time.Time
}

func TestLRUCacheExportImportMemory(t *testing.T) {
	gob.Register(userData{})

	cache, err := NewLRUCache(3)
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}

	cache.Add("alice", userData{"Alice", time.Now()})
	cache.Add("bob", userData{"Bob", time.Now().Add(-time.Hour)})
	cache.Add("carol", userData{"Carol", time.Now().Add(-2 * time.Hour)})

	wantOrder := cache.Keys()

	var buf bytes.Buffer
	if err := cache.Export(&buf); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	cache.Purge()
	if err := cache.Import(&buf); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	gotOrder := cache.Keys()
	if !reflect.DeepEqual(wantOrder, gotOrder) {
		t.Fatalf("key order not preserved: want=%v, got=%v", wantOrder, gotOrder)
	}

	// verify data round-tripped
	v, ok := cache.Get("alice")
	if !ok {
		t.Fatalf("missing key after import")
	}
	if v.(userData).Name != "Alice" {
		t.Fatalf("wrong data after import: %+v", v)
	}
}

func TestLRUCacheExportImportFile(t *testing.T) {
	cache, _ := NewLRUCache(2)
	cache.Add("u1", userData{"User1", time.Now()})
	cache.Add("u2", userData{"User2", time.Now()})
	wantKeys := cache.Keys()

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "cache.gob")

	if err := cache.ExportToFile(file); err != nil {
		t.Fatalf("ExportToFile failed: %v", err)
	}
	if st, err := os.Stat(file); err != nil || st.Size() == 0 {
		t.Fatalf("expected non-empty file, got err=%v size=%d", err, st.Size())
	}

	cache.Purge()
	if err := cache.ImportFromFile(file); err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}

	gotKeys := cache.Keys()
	if !reflect.DeepEqual(wantKeys, gotKeys) {
		t.Fatalf("key order mismatch after file import: want=%v got=%v", wantKeys, gotKeys)
	}
}

func TestLRUCacheExportImportPreservesRecency(t *testing.T) {
	cache, _ := NewLRUCache(3)
	cache.Add("A", userData{"A", time.Now()})
	cache.Add("B", userData{"B", time.Now()})
	cache.Add("C", userData{"C", time.Now()})

	// Access B to make it MRU
	cache.Get("B")
	wantOrder := cache.Keys()

	var buf bytes.Buffer
	if err := cache.Export(&buf); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	cache.Purge()
	if err := cache.Import(&buf); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	gotOrder := cache.Keys()
	if !reflect.DeepEqual(wantOrder, gotOrder) {
		t.Fatalf("recency not preserved: want=%v got=%v", wantOrder, gotOrder)
	}
}

func TestLRUCacheImportInvalidData(t *testing.T) {
	cache, _ := NewLRUCache(2)
	buf := bytes.NewBufferString("this-is-not-valid-gob-data")

	if err := cache.Import(buf); err == nil {
		t.Fatalf("expected error on invalid gob data, got nil")
	}
}

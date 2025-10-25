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

func TestLRUCacheExportDoesNotMutateRecency(t *testing.T) {
	gob.Register(userData{})

	cache, _ := NewLRUCache(3)
	cache.Add("A", userData{"A", time.Now()})
	cache.Add("B", userData{"B", time.Now()})
	cache.Add("C", userData{"C", time.Now()})

	wantOrder := cache.Keys() // MRU->LRU

	var buf bytes.Buffer
	if err := cache.Export(&buf); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	gotOrder := cache.Keys()
	if !reflect.DeepEqual(wantOrder, gotOrder) {
		t.Fatalf("Export changed recency: want=%v got=%v", wantOrder, gotOrder)
	}
}

func TestLRUCacheSnapshotBasic(t *testing.T) {
	gob.Register(userData{})

	cache, _ := NewLRUCache(4)
	cache.Add("X", userData{"X", time.Now()})
	cache.Add("Y", userData{"Y", time.Now()})
	cache.Add("Z", userData{"Z", time.Now()})

	size, pairs := cache.Snapshot()
	if size != 4 {
		t.Fatalf("Snapshot size mismatch: want=4 got=%d", size)
	}
	if len(pairs) != 3 {
		t.Fatalf("Snapshot entries mismatch: want=3 got=%d", len(pairs))
	}

	// Verify MRU->LRU order via Keys()
	keys := cache.Keys()
	for i, k := range keys {
		if pairs[i].Key != k {
			t.Fatalf("Snapshot order mismatch at %d: want=%v got=%v", i, k, pairs[i].Key)
		}
	}

	// Mutate snapshot slice; cache must stay intact
	if len(pairs) > 0 {
		pairs[0].Key = "MUTATED"
	}
	keys2 := cache.Keys()
	if keys2[0] == "MUTATED" {
		t.Fatalf("Cache affected by snapshot slice mutation")
	}
}

func TestLRUCacheImportEnforcesSize(t *testing.T) {
	gob.Register(userData{})

	// Build an on-disk payload with Size=2 but 3 entries (MRU->LRU: K1, K2, K3)
	now := time.Now()
	payload := onDisk{
		Size: 2,
		Entries: []kv{
			{K: "K1", V: userData{"V1", now}},
			{K: "K2", V: userData{"V2", now}},
			{K: "K3", V: userData{"V3", now}},
		},
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&payload); err != nil {
		t.Fatalf("encode payload failed: %v", err)
	}

	cache, _ := NewLRUCache(10) // will be overwritten by import
	if err := cache.Import(&buf); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if gotLen := cache.Length(); gotLen != 2 {
		t.Fatalf("Import did not enforce size: want=2 got=%d", gotLen)
	}

	// The two most recent should remain: K1, K2 (MRU->LRU)
	want := []interface{}{"K1", "K2"}
	got := cache.Keys()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("Unexpected keys after size enforcement: want=%v got=%v", want, got)
	}
	if cache.Contains("K3") {
		t.Fatalf("Oldest (K3) should have been evicted")
	}
}

func TestLRUCacheExportToFileAtomicOverwritesCorruptFile(t *testing.T) {
	gob.Register(userData{})

	cache, _ := NewLRUCache(5)
	cache.Add("u1", userData{"User1", time.Now()})
	cache.Add("u2", userData{"User2", time.Now()})

	dir := t.TempDir()
	path := filepath.Join(dir, "cache.gob")

	// Write corrupt content first
	if err := os.WriteFile(path, []byte("corrupt-data"), 0o600); err != nil {
		t.Fatalf("failed to seed corrupt file: %v", err)
	}

	// Atomic export should overwrite with a valid gob
	if err := cache.ExportToFileAtomic(path); err != nil {
		t.Fatalf("ExportToFileAtomic failed: %v", err)
	}

	// Now ImportFromFile must succeed and match keys
	cache2, _ := NewLRUCache(2) // size should be replaced by import
	if err := cache2.ImportFromFile(path); err != nil {
		t.Fatalf("ImportFromFile failed after atomic export: %v", err)
	}

	want := cache.Keys()
	got := cache2.Keys()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("keys mismatch after atomic file roundtrip: want=%v got=%v", want, got)
	}
}

func TestLRUCacheExportImportEmptyCache(t *testing.T) {
	cache, err := NewLRUCache(3) // valid size
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}

	// Export an empty cache
	var buf bytes.Buffer
	if err := cache.Export(&buf); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import into a (different) valid cache instance
	cache2, err := NewLRUCache(2) // must be >1, will be overwritten by import anyway
	if err != nil {
		t.Fatalf("NewLRUCache failed: %v", err)
	}
	if err := cache2.Import(&buf); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Still empty after roundtrip
	if cache2.Length() != 0 {
		t.Fatalf("expected empty cache after importing empty export, got %d", cache2.Length())
	}

	// Size should match the exporter’s size (3)
	if got := cache2.size; got != 3 { // ok: test is in same package
		t.Fatalf("expected imported size 3, got %d", got)
	}
}

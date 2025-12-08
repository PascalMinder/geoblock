package geoblock

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	lru "github.com/PascalMinder/geoblock/lrucache"
)

const defaultWriteCycle = 15

// Options configures cache initialization and persistence behavior.
type Options struct {
	// CacheSize specifies the number of entries to store in memory.
	CacheSize int

	// CachePath is the file path used for on-disk persistence.
	// Leave empty to disable persistence.
	CachePath string

	// PersistInterval defines how often the cache is automatically
	// flushed to disk. Defaults to 15 seconds if zero.
	PersistInterval time.Duration

	// Logger is used for diagnostic messages.
	Logger *log.Logger

	// Name is included in log messages to identify the cache instance.
	Name string
}

// InitializeCache creates a new LRU cache and, if a valid persistence
// path is provided, starts a background goroutine to periodically
// save snapshots to disk.
//
// The returned `CachePersist` can be used to mark the cache as dirty when
// data changes. If persistence is disabled, it is a no-op.
//
// Callers should cancel the provided context to stop persistence
// and ensure a final snapshot is written.
func InitializeCache(ctx context.Context, opts Options) (*lru.LRUCache, *CachePersist, error) {
	if opts.PersistInterval <= 0 {
		opts.PersistInterval = defaultWriteCycle * time.Second
	}

	cache, err := lru.NewLRUCache(opts.CacheSize)
	if err != nil {
		return nil, nil, fmt.Errorf("create LRU cache: %w", err)
	}

	var ipDB *CachePersist // stays nil if disabled
	if path, err := ValidatePersistencePath(opts.CachePath); len(path) > 0 {
		// load existing cache
		if err := cache.ImportFromFile(path); err != nil && !os.IsNotExist(err) {
			opts.Logger.Printf("%s: could not load IP DB snapshot (%s): %v", opts.Name, path, err)
		}

		ipDB = &CachePersist{
			path:           path,
			persistTicker:  time.NewTicker(opts.PersistInterval),
			persistChannel: make(chan struct{}, 1),
			cache:          cache,
			log:            opts.Logger,
			name:           opts.Name,
		}

		go func(ctx context.Context, p *CachePersist) {
			defer p.persistTicker.Stop()
			for {
				select {
				case <-ctx.Done():
					p.snapshotToDisk()
					return
				case <-p.persistTicker.C:
					p.snapshotToDisk()
				case <-p.persistChannel:
					p.snapshotToDisk()
				}
			}
		}(ctx, ipDB)

		opts.Logger.Printf("%s: IP database persistence enabled -> %s", opts.Name, path)
	} else if err != nil {
		opts.Logger.Printf("%s: IP database persistence disabled: %v", opts.Name, err)
	} else {
		opts.Logger.Printf("%s: IP database persistence disabled (no path)", opts.Name)
	}

	return cache, ipDB, nil
}

// CachePersist periodically snapshots a cache to disk.
type CachePersist struct {
	path           string
	persistTicker  *time.Ticker
	persistChannel chan struct{}
	ipDirty        uint32 // 0 clean, 1 dirty

	cache *lru.LRUCache
	log   *log.Logger
	name  string
}

// MarkDirty marks the cache as modified and schedules a snapshot.
func (p *CachePersist) MarkDirty() {
	if p == nil { // feature OFF
		return
	}
	atomic.StoreUint32(&p.ipDirty, 1)
	select {
	case p.persistChannel <- struct{}{}:
	default:
	}
}

// Snapshot writes the cache to disk if it has been marked dirty.
func (p *CachePersist) snapshotToDisk() {
	if p == nil || atomic.LoadUint32(&p.ipDirty) == 0 {
		return
	}

	var buf bytes.Buffer
	if err := p.cache.Export(&buf); err != nil {
		p.log.Printf("%s: cache snapshot encode error: %v", p.name, err)
		return
	}

	dir := filepath.Dir(p.path)
	tmp, err := os.CreateTemp(dir, "ipdb-*.tmp")
	if err != nil {
		p.log.Printf("%s: snapshot temp file error: %v", p.name, err)
		return
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		p.log.Printf("%s: snapshot write error: %v", p.name, err)
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		p.log.Printf("%s: snapshot fsync error: %v", p.name, err)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		p.log.Printf("%s: snapshot close error: %v", p.name, err)
		return
	}
	if err := os.Rename(tmpPath, p.path); err != nil {
		os.Remove(tmpPath)
		p.log.Printf("%s: snapshot rename error: %v", p.name, err)
		return
	}

	atomic.StoreUint32(&p.ipDirty, 0)
}

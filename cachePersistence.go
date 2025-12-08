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

const (
	DefaultPersistInterval      = 10 * time.Second
	PersistIntervalMultiplicate = 3
)

// Options describes how to initialize the in-memory cache and optional persistence.
type Options struct {
	CacheSize       int
	CachePath       string        // file path for persisted cache; if empty or invalid → feature OFF
	PersistInterval time.Duration // base interval; used for debounce + max interval
	Logger          *log.Logger
	Name            string
}

// CachePersist manages debounced, low-CPU persistence of the LRU cache.
type CachePersist struct {
	path  string
	cache *lru.LRUCache
	log   *log.Logger
	name  string

	ch   chan struct{} // edge-trigger signal (coalesced)
	quit chan struct{} // stop signal

	minInterval time.Duration // debounce interval
	maxInterval time.Duration // hard max between flushes

	ipDirty   uint32       // 0 clean, 1 dirty
	lastFlush atomic.Int64 // unix nano of last successful flush
}

// NewCachePersist constructs a new persistence controller.
// It does NOT start the worker; caller must call go p.Run(ctx).
func NewCachePersist(
	path string, cache *lru.LRUCache, logger *log.Logger, name string, persistInterval time.Duration) *CachePersist {
	if persistInterval <= 0 {
		persistInterval = DefaultPersistInterval
	}

	p := &CachePersist{
		path:        path,
		cache:       cache,
		log:         logger,
		name:        name,
		ch:          make(chan struct{}, 1),
		quit:        make(chan struct{}),
		minInterval: persistInterval,
		maxInterval: PersistIntervalMultiplicate * persistInterval,
	}
	p.lastFlush.Store(time.Now().UnixNano())
	return p
}

// MarkDirty marks the cache as needing a flush and nudges the worker.
func (p *CachePersist) MarkDirty() {
	if p == nil {
		return
	}
	atomic.StoreUint32(&p.ipDirty, 1)
	select {
	case p.ch <- struct{}{}:
	default:
		// already scheduled; no need to push again
	}
}

func (p *CachePersist) Run(ctx context.Context) {
	if p == nil {
		return
	}

	wait := func(d time.Duration) bool {
		if d <= 0 {
			return false
		}
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return true
		case <-p.quit:
			return true
		case <-t.C:
			return false
		}
	}

	for {
		select {
		case <-ctx.Done():
			p.flushIfDirty()
			return
		case <-p.quit:
			p.flushIfDirty()
			return

		case <-p.ch:
			// Drain additional nudges to coalesce bursts
			drained := true
			for drained {
				select {
				case <-p.ch:
				default:
					drained = false
				}
			}

			now := time.Now()
			last := time.Unix(0, p.lastFlush.Load())
			since := now.Sub(last)
			untilMin := p.minInterval - since
			untilMax := p.maxInterval - since

			// If we already exceeded max interval, flush immediately.
			if untilMax <= 0 {
				p.flushIfDirty()
				continue
			}

			// Otherwise wait the rest of the debounce window.
			if wait(untilMin) {
				p.flushIfDirty()
				return
			}

			p.flushIfDirty()
		}
	}
}

// Stop asks the worker to stop and does a final flush.
func (p *CachePersist) Stop() {
	if p == nil {
		return
	}
	close(p.quit)
}

func (p *CachePersist) flushIfDirty() {
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
		_ = os.Remove(tmpPath)
		p.log.Printf("%s: snapshot write error: %v", p.name, err)
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		p.log.Printf("%s: snapshot fsync error: %v", p.name, err)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		p.log.Printf("%s: snapshot close error: %v", p.name, err)
		return
	}
	if err := os.Rename(tmpPath, p.path); err != nil {
		_ = os.Remove(tmpPath)
		p.log.Printf("%s: snapshot rename error: %v", p.name, err)
		return
	}

	atomic.StoreUint32(&p.ipDirty, 0)
	p.lastFlush.Store(time.Now().UnixNano())
}

func InitializeCache(ctx context.Context, opt Options) (*lru.LRUCache, *CachePersist, error) {
	if opt.CacheSize <= 1 {
		return nil, nil, fmt.Errorf("cache size must be bigger than 1")
	}

	cache, err := lru.NewLRUCache(opt.CacheSize)
	if err != nil {
		return nil, nil, fmt.Errorf("create lru cache: %w", err)
	}

	logger := opt.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	var persist *CachePersist

	if opt.CachePath != "" {
		path, err := ValidatePersistencePath(opt.CachePath)
		if err != nil {
			logger.Printf("%s: IP cache persistence disabled (path invalid): %v", opt.Name, err)
		} else {
			// Try warm load; ignore file-not-exist
			if err := cache.ImportFromFile(path); err != nil && !os.IsNotExist(err) {
				logger.Printf("%s: failed to warm-load IP cache from %s: %v", opt.Name, path, err)
			}

			persist = NewCachePersist(path, cache, logger, opt.Name, opt.PersistInterval)
			go persist.Run(ctx)
			logger.Printf("%s: IP cache persistence enabled -> %s", opt.Name, path)
		}
	} else {
		logger.Printf("%s: IP cache persistence disabled (no path configured)", opt.Name)
	}

	return cache, persist, nil
}

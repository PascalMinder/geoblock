package geoblock

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	lru "github.com/PascalMinder/geoblock/lrucache"
)

func InitializeCache(ctx context.Context, logger *log.Logger, name string, cacheSize int, cachePath string) (*lru.LRUCache, *CachePersist) {
	cache, err := lru.NewLRUCache(cacheSize)
	if err != nil {
		logger.Fatal(err)
	}

	var ipDB *CachePersist // stays nil if disabled
	if path, err := ValidatePersistencePath(cachePath); len(path) > 0 {
		// load existing cache
		if err := cache.ImportFromFile(path); err != nil && !os.IsNotExist(err) {
			logger.Printf("%s: could not load IP DB snapshot (%s): %v", name, path, err)
		}

		ipDB = &CachePersist{
			path:           path,
			persistTicker:  time.NewTicker(15 * time.Second),
			persistChannel: make(chan struct{}, 1),
			cache:          cache,
			log:            logger,
			name:           name,
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

		logger.Printf("%s: IP database persistence enabled -> %s", name, path)
	} else if err != nil {
		logger.Printf("%s: IP database persistence disabled: %v", name, err)
	} else {
		logger.Printf("%s: IP database persistence disabled (no path)", name)
	}

	return cache, ipDB
}

type CachePersist struct {
	path           string
	persistTicker  *time.Ticker
	persistChannel chan struct{}
	ipDirty        uint32 // 0 clean, 1 dirty

	cache *lru.LRUCache
	log   *log.Logger
	name  string
}

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
	p.log.Printf("%s: cache snapshot written to %s", p.name, p.path)
}

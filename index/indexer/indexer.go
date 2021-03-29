//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package indexer allows to search for metadata and content.
package indexer

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/index/memstore"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/strfun"
)

type indexer struct {
	store   index.Store
	ar      *anterooms
	ready   chan struct{} // Signal a non-empty anteroom to background task
	done    chan struct{} // Stop background task
	observe bool
	started bool

	// Stats data
	mx           sync.RWMutex
	lastReload   time.Time
	sinceReload  uint64
	durLastIndex time.Duration
}

// New creates a new indexer.
func New() index.Indexer {
	return &indexer{
		store: memstore.New(),
		ar:    newAnterooms(10),
		ready: make(chan struct{}, 1),
	}
}

func (idx *indexer) observer(ci change.Info) {
	switch ci.Reason {
	case change.OnReload:
		idx.ar.Reset()
	case change.OnUpdate:
		idx.ar.Enqueue(ci.Zid, arUpdate)
	case change.OnDelete:
		idx.ar.Enqueue(ci.Zid, arDelete)
	default:
		return
	}
	select {
	case idx.ready <- struct{}{}:
	default:
	}
}

func (idx *indexer) Start(p index.Port) {
	if idx.started {
		panic("Index already started")
	}
	idx.done = make(chan struct{})
	if !idx.observe {
		p.RegisterObserver(idx.observer)
		idx.observe = true
	}
	idx.ar.Reset() // Ensure an initial index run
	go idx.indexer(p)
	idx.started = true
}

func (idx *indexer) Stop() {
	if !idx.started {
		panic("Index already stopped")
	}
	close(idx.done)
	idx.started = false
}

// Enrich reads all properties in the index and updates the metadata.
func (idx *indexer) Enrich(ctx context.Context, m *meta.Meta) {
	if index.DoNotEnrich(ctx) {
		// Enrich is called indirectly via indexer or enrichment is not requested
		// because of other reasons -> ignore this call, do not update meta data
		return
	}
	idx.store.Enrich(ctx, m)
}

// Select all zettel that contains the given exact word.
// The word must be normalized through Unicode NKFD.
func (idx *indexer) Select(word string) id.Set {
	return idx.store.Select(word)
}

// Select all zettel that have a word with the given prefix.
// The prefix must be normalized through Unicode NKFD.
func (idx *indexer) SelectPrefix(prefix string) id.Set {
	return idx.store.SelectPrefix(prefix)
}

// Select all zettel that contains the given string.
// The string must be normalized through Unicode NKFD.
func (idx *indexer) SelectContains(s string) id.Set {
	return idx.store.SelectContains(s)
}

// ReadStats populates st with indexer statistics.
func (idx *indexer) ReadStats(st *index.IndexerStats) {
	idx.mx.RLock()
	st.LastReload = idx.lastReload
	st.IndexesSinceReload = idx.sinceReload
	st.DurLastIndex = idx.durLastIndex
	idx.mx.RUnlock()
	idx.store.ReadStats(&st.Store)
}

type indexerPort interface {
	getMetaPort
	FetchZids(ctx context.Context) (id.Set, error)
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
}

// indexer runs in the background and updates the index data structures.
// This is the main service of the indexer.
func (idx *indexer) indexer(p indexerPort) {
	// Something may panic. Ensure a running indexer.
	defer func() {
		if r := recover(); r != nil {
			log.Println("recovered from:", r)
			debug.PrintStack()
			go idx.indexer(p)
		}
	}()

	timerDuration := 15 * time.Second
	timer := time.NewTimer(timerDuration)
	ctx := index.NoEnrichContext(context.Background())
	for {
		start := time.Now()
		if idx.workService(ctx, p) {
			idx.mx.Lock()
			idx.durLastIndex = time.Since(start)
			idx.mx.Unlock()
		}
		if !idx.sleepService(timer, timerDuration) {
			return
		}
	}
}

func (idx *indexer) workService(ctx context.Context, p indexerPort) bool {
	changed := false
	for {
		switch action, zid := idx.ar.Dequeue(); action {
		case arNothing:
			return changed
		case arReload:
			zids, err := p.FetchZids(ctx)
			if err == nil {
				idx.ar.Reload(nil, zids)
				idx.mx.Lock()
				idx.lastReload = time.Now()
				idx.sinceReload = 0
				idx.mx.Unlock()
			}
		case arUpdate:
			changed = true
			idx.mx.Lock()
			idx.sinceReload++
			idx.mx.Unlock()
			zettel, err := p.GetZettel(ctx, zid)
			if err != nil {
				// TODO: on some errors put the zid into a "try later" set
				continue
			}
			idx.updateZettel(ctx, zettel, p)
		case arDelete:
			changed = true
			idx.mx.Lock()
			idx.sinceReload++
			idx.mx.Unlock()
			idx.deleteZettel(zid)
		}
	}
}

func (idx *indexer) sleepService(timer *time.Timer, timerDuration time.Duration) bool {
	select {
	case _, ok := <-idx.ready:
		if !ok {
			return false
		}
	case _, ok := <-timer.C:
		if !ok {
			return false
		}
		timer.Reset(timerDuration)
	case _, ok := <-idx.done:
		if !ok {
			if !timer.Stop() {
				<-timer.C
			}
			return false
		}
	}
	return true
}

type getMetaPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

func (idx *indexer) updateZettel(ctx context.Context, zettel domain.Zettel, p getMetaPort) {
	refs := id.NewSet()
	words := make(index.WordSet)
	collectZettelIndexData(parser.ParseZettel(zettel, ""), refs, words)
	m := zettel.Meta
	zi := index.NewZettelIndex(m.Zid)
	for _, pair := range m.Pairs(false) {
		descr := meta.GetDescription(pair.Key)
		if descr.IsComputed() {
			continue
		}
		switch descr.Type {
		case meta.TypeID:
			updateValue(ctx, descr.Inverse, pair.Value, p, zi)
		case meta.TypeIDSet:
			for _, val := range meta.ListFromValue(pair.Value) {
				updateValue(ctx, descr.Inverse, val, p, zi)
			}
		case meta.TypeZettelmarkup:
			collectInlineIndexData(parser.ParseMetadata(pair.Value), refs, words)
		default:
			for _, word := range strfun.NormalizeWords(pair.Value) {
				words[word] = words[word] + 1
			}
		}
	}
	for ref := range refs {
		if _, err := p.GetMeta(ctx, ref); err == nil {
			zi.AddBackRef(ref)
		} else {
			zi.AddDeadRef(ref)
		}
	}
	zi.SetWords(words)
	toCheck := idx.store.UpdateReferences(ctx, zi)
	idx.checkZettel(toCheck)
}

func updateValue(ctx context.Context, inverse string, value string, p getMetaPort, zi *index.ZettelIndex) {
	zid, err := id.Parse(value)
	if err != nil {
		return
	}
	if _, err := p.GetMeta(ctx, zid); err != nil {
		zi.AddDeadRef(zid)
		return
	}
	if inverse == "" {
		zi.AddBackRef(zid)
		return
	}
	zi.AddMetaRef(inverse, zid)
}

func (idx *indexer) deleteZettel(zid id.Zid) {
	toCheck := idx.store.DeleteZettel(context.Background(), zid)
	idx.checkZettel(toCheck)
}

func (idx *indexer) checkZettel(s id.Set) {
	for zid := range s {
		idx.ar.Enqueue(zid, arUpdate)
	}
}

//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various places and indexes of a Zettelstore.
package manager

import (
	"context"
	"net/url"
	"sync"
	"time"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/place/manager/index"
	"zettelstore.de/z/place/manager/memstore"
	"zettelstore.de/z/service"
	"zettelstore.de/z/strfun"
)

// Indexer is the process that collect asyncronous zettel data for faster search.
type Indexer struct {
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

// newIndexer creates a new indexer.
func newIndexer() *Indexer {
	return &Indexer{
		store: memstore.New(),
		ar:    newAnterooms(10),
		ready: make(chan struct{}, 1),
	}
}

func (idx *Indexer) observer(ci change.Info) {
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

// indexerPort contains all the used functions to access zettel to be indexed.
type indexerPort interface {
	RegisterObserver(change.Func)
	FetchZids(context.Context) (id.Set, error)
	GetMeta(context.Context, id.Zid) (*meta.Meta, error)
	GetZettel(context.Context, id.Zid) (domain.Zettel, error)
}

// Start the indexer.
func (idx *Indexer) startIndexer(p indexerPort) {
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

// Stop the indexer.
func (idx *Indexer) stopIndexer() {
	if !idx.started {
		panic("Index already stopped")
	}
	close(idx.done)
	idx.started = false
}

// SelectEqual all zettel that contains the given exact word.
// The word must be normalized through Unicode NKFD, trimmed and not empty.
func (idx *Indexer) SelectEqual(word string) id.Set {
	return idx.store.SelectEqual(word)
}

// SelectPrefix all zettel that have a word with the given prefix.
// The prefix must be normalized through Unicode NKFD, trimmed and not empty.
func (idx *Indexer) SelectPrefix(prefix string) id.Set {
	return idx.store.SelectPrefix(prefix)
}

// SelectSuffix all zettel that have a word with the given suffix.
// The suffix must be normalized through Unicode NKFD, trimmed and not empty.
func (idx *Indexer) SelectSuffix(suffix string) id.Set {
	return idx.store.SelectSuffix(suffix)
}

// SelectContains all zettel that contains the given string.
// The string must be normalized through Unicode NKFD, trimmed and not empty.
func (idx *Indexer) SelectContains(s string) id.Set {
	return idx.store.SelectContains(s)
}

// readStats populates st with indexer statistics.
func (idx *Indexer) readStats(st *place.Stats) {
	var storeSt index.Stats
	idx.mx.RLock()
	defer idx.mx.RUnlock()
	idx.store.ReadStats(&storeSt)

	st.LastReload = idx.lastReload
	st.IndexesSinceReload = idx.sinceReload
	st.DurLastIndex = idx.durLastIndex
	st.ZettelIndexed = storeSt.Zettel
	st.IndexUpdates = storeSt.Updates
	st.IndexedWords = storeSt.Words
	st.IndexedUrls = storeSt.Urls
}

type indexWorkerPort interface {
	getMetaPort
	FetchZids(ctx context.Context) (id.Set, error)
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
}

// indexer runs in the background and updates the index data structures.
// This is the main service of the indexer.
func (idx *Indexer) indexer(p indexWorkerPort) {
	// Something may panic. Ensure a running indexer.
	defer func() {
		if r := recover(); r != nil {
			service.Main.LogRecover("Indexer", r)
			go idx.indexer(p)
		}
	}()

	timerDuration := 15 * time.Second
	timer := time.NewTimer(timerDuration)
	ctx := place.NoEnrichContext(context.Background())
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

func (idx *Indexer) workService(ctx context.Context, p indexWorkerPort) bool {
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

func (idx *Indexer) sleepService(timer *time.Timer, timerDuration time.Duration) bool {
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
	case <-idx.done:
		if !timer.Stop() {
			<-timer.C
		}
		return false
	}
	return true
}

type getMetaPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

func (idx *Indexer) updateZettel(ctx context.Context, zettel domain.Zettel, p getMetaPort) {
	m := zettel.Meta
	if m.GetBool(meta.KeyNoIndex) {
		// Zettel maybe in index
		toCheck := idx.store.DeleteZettel(ctx, m.Zid)
		idx.checkZettel(toCheck)
		return
	}

	var cData collectData
	cData.initialize()
	collectZettelIndexData(parser.ParseZettel(zettel, ""), &cData)
	zi := index.NewZettelIndex(m.Zid)
	collectFromMeta(ctx, m, zi, &cData, p)
	processData(ctx, zi, &cData, p)
	toCheck := idx.store.UpdateReferences(ctx, zi)
	idx.checkZettel(toCheck)
}

func collectFromMeta(ctx context.Context, m *meta.Meta, zi *index.ZettelIndex, cData *collectData, p getMetaPort) {
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
			collectInlineIndexData(parser.ParseMetadata(pair.Value), cData)
		case meta.TypeURL:
			if _, err := url.Parse(pair.Value); err == nil {
				cData.urls.Add(pair.Value)
			}
		default:
			for _, word := range strfun.NormalizeWords(pair.Value) {
				cData.words.Add(word)
			}
		}
	}
}

func processData(ctx context.Context, zi *index.ZettelIndex, cData *collectData, p getMetaPort) {
	for ref := range cData.refs {
		if _, err := p.GetMeta(ctx, ref); err == nil {
			zi.AddBackRef(ref)
		} else {
			zi.AddDeadRef(ref)
		}
	}
	zi.SetWords(cData.words)
	zi.SetUrls(cData.urls)
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

func (idx *Indexer) deleteZettel(zid id.Zid) {
	toCheck := idx.store.DeleteZettel(context.Background(), zid)
	idx.checkZettel(toCheck)
}

func (idx *Indexer) checkZettel(s id.Set) {
	for zid := range s {
		idx.ar.Enqueue(zid, arUpdate)
	}
}

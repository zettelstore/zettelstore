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
	"time"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager/store"
	"zettelstore.de/z/service"
	"zettelstore.de/z/strfun"
)

// SelectEqual all zettel that contains the given exact word.
// The word must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SelectEqual(word string) id.Set {
	return mgr.idxStore.SelectEqual(word)
}

// SelectPrefix all zettel that have a word with the given prefix.
// The prefix must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SelectPrefix(prefix string) id.Set {
	return mgr.idxStore.SelectPrefix(prefix)
}

// SelectSuffix all zettel that have a word with the given suffix.
// The suffix must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SelectSuffix(suffix string) id.Set {
	return mgr.idxStore.SelectSuffix(suffix)
}

// SelectContains all zettel that contains the given string.
// The string must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SelectContains(s string) id.Set {
	return mgr.idxStore.SelectContains(s)
}

// idxIndexer runs in the background and updates the index data structures.
// This is the main service of the idxIndexer.
func (mgr *Manager) idxIndexer() {
	// Something may panic. Ensure a running indexer.
	defer func() {
		if r := recover(); r != nil {
			service.Main.LogRecover("Indexer", r)
			go mgr.idxIndexer()
		}
	}()

	timerDuration := 15 * time.Second
	timer := time.NewTimer(timerDuration)
	ctx := place.NoEnrichContext(context.Background())
	for {
		start := time.Now()
		if mgr.idxWorkService(ctx) {
			mgr.idxMx.Lock()
			mgr.idxDurLastIndex = time.Since(start)
			mgr.idxMx.Unlock()
		}
		if !mgr.idxSleepService(timer, timerDuration) {
			return
		}
	}
}

func (mgr *Manager) idxWorkService(ctx context.Context) bool {
	changed := false
	for {
		switch action, zid := mgr.idxAr.Dequeue(); action {
		case arNothing:
			return changed
		case arReload:
			zids, err := mgr.FetchZids(ctx)
			if err == nil {
				mgr.idxAr.Reload(nil, zids)
				mgr.idxMx.Lock()
				mgr.idxLastReload = time.Now()
				mgr.idxSinceReload = 0
				mgr.idxMx.Unlock()
			}
		case arUpdate:
			changed = true
			mgr.idxMx.Lock()
			mgr.idxSinceReload++
			mgr.idxMx.Unlock()
			zettel, err := mgr.GetZettel(ctx, zid)
			if err != nil {
				// TODO: on some errors put the zid into a "try later" set
				continue
			}
			mgr.idxUpdateZettel(ctx, zettel)
		case arDelete:
			changed = true
			mgr.idxMx.Lock()
			mgr.idxSinceReload++
			mgr.idxMx.Unlock()
			mgr.idxDeleteZettel(zid)
		}
	}
}

func (mgr *Manager) idxSleepService(timer *time.Timer, timerDuration time.Duration) bool {
	select {
	case _, ok := <-mgr.idxReady:
		if !ok {
			return false
		}
	case _, ok := <-timer.C:
		if !ok {
			return false
		}
		timer.Reset(timerDuration)
	case <-mgr.done:
		if !timer.Stop() {
			<-timer.C
		}
		return false
	}
	return true
}

func (mgr *Manager) idxUpdateZettel(ctx context.Context, zettel domain.Zettel) {
	m := zettel.Meta
	if m.GetBool(meta.KeyNoIndex) {
		// Zettel maybe in index
		toCheck := mgr.idxStore.DeleteZettel(ctx, m.Zid)
		mgr.idxCheckZettel(toCheck)
		return
	}

	var cData collectData
	cData.initialize()
	collectZettelIndexData(parser.ParseZettel(zettel, ""), &cData)
	zi := store.NewZettelIndex(m.Zid)
	mgr.idxCollectFromMeta(ctx, m, zi, &cData)
	mgr.idxProcessData(ctx, zi, &cData)
	toCheck := mgr.idxStore.UpdateReferences(ctx, zi)
	mgr.idxCheckZettel(toCheck)
}

func (mgr *Manager) idxCollectFromMeta(ctx context.Context, m *meta.Meta, zi *store.ZettelIndex, cData *collectData) {
	for _, pair := range m.Pairs(false) {
		descr := meta.GetDescription(pair.Key)
		if descr.IsComputed() {
			continue
		}
		switch descr.Type {
		case meta.TypeID:
			mgr.idxUpdateValue(ctx, descr.Inverse, pair.Value, zi)
		case meta.TypeIDSet:
			for _, val := range meta.ListFromValue(pair.Value) {
				mgr.idxUpdateValue(ctx, descr.Inverse, val, zi)
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

func (mgr *Manager) idxProcessData(ctx context.Context, zi *store.ZettelIndex, cData *collectData) {
	for ref := range cData.refs {
		if _, err := mgr.GetMeta(ctx, ref); err == nil {
			zi.AddBackRef(ref)
		} else {
			zi.AddDeadRef(ref)
		}
	}
	zi.SetWords(cData.words)
	zi.SetUrls(cData.urls)
}

func (mgr *Manager) idxUpdateValue(ctx context.Context, inverse string, value string, zi *store.ZettelIndex) {
	zid, err := id.Parse(value)
	if err != nil {
		return
	}
	if _, err := mgr.GetMeta(ctx, zid); err != nil {
		zi.AddDeadRef(zid)
		return
	}
	if inverse == "" {
		zi.AddBackRef(zid)
		return
	}
	zi.AddMetaRef(inverse, zid)
}

func (mgr *Manager) idxDeleteZettel(zid id.Zid) {
	toCheck := mgr.idxStore.DeleteZettel(context.Background(), zid)
	mgr.idxCheckZettel(toCheck)
}

func (mgr *Manager) idxCheckZettel(s id.Set) {
	for zid := range s {
		mgr.idxAr.Enqueue(zid, arUpdate)
	}
}

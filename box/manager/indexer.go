//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various boxes and indexes of a Zettelstore.
package manager

import (
	"context"
	"net/url"
	"time"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager/store"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"
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
			kernel.Main.LogRecover("Indexer", r)
			go mgr.idxIndexer()
		}
	}()

	timerDuration := 15 * time.Second
	timer := time.NewTimer(timerDuration)
	ctx := box.NoEnrichContext(context.Background())
	for {
		mgr.idxWorkService(ctx)
		if !mgr.idxSleepService(timer, timerDuration) {
			return
		}
	}
}

func (mgr *Manager) idxWorkService(ctx context.Context) {
	var roomNum uint64
	var start time.Time
	for {
		switch action, zid, arRoomNum := mgr.idxAr.Dequeue(); action {
		case arNothing:
			return
		case arReload:
			roomNum = 0
			zids, err := mgr.FetchZids(ctx)
			if err == nil {
				start = time.Now()
				if rno := mgr.idxAr.Reload(zids); rno > 0 {
					roomNum = rno
				}
				mgr.idxMx.Lock()
				mgr.idxLastReload = time.Now()
				mgr.idxSinceReload = 0
				mgr.idxMx.Unlock()
			}
		case arUpdate:
			zettel, err := mgr.GetZettel(ctx, zid)
			if err != nil {
				// TODO: on some errors put the zid into a "try later" set
				continue
			}
			mgr.idxMx.Lock()
			if arRoomNum == roomNum {
				mgr.idxDurReload = time.Since(start)
			}
			mgr.idxSinceReload++
			mgr.idxMx.Unlock()
			mgr.idxUpdateZettel(ctx, zettel)
		case arDelete:
			if _, err := mgr.GetMeta(ctx, zid); err == nil {
				// Zettel was not deleted. This might occur, if zettel was
				// deleted in secondary dirbox, but is still present in
				// first dirbox (or vice versa). Re-index zettel in case
				// a hidden zettel was recovered
				mgr.idxAr.Enqueue(zid, arUpdate)
			}
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
	collectZettelIndexData(parser.ParseZettel(zettel, "", mgr.rtConfig), &cData)
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

func (mgr *Manager) idxUpdateValue(ctx context.Context, inverseKey, value string, zi *store.ZettelIndex) {
	zid, err := id.Parse(value)
	if err != nil {
		return
	}
	if _, err := mgr.GetMeta(ctx, zid); err != nil {
		zi.AddDeadRef(zid)
		return
	}
	if inverseKey == "" {
		zi.AddBackRef(zid)
		return
	}
	zi.AddMetaRef(inverseKey, zid)
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

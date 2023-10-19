//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package manager

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager/store"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// SearchEqual returns all zettel that contains the given exact word.
// The word must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SearchEqual(word string) id.Set {
	found := mgr.idxStore.SearchEqual(word)
	mgr.idxLog.Debug().Str("word", word).Int("found", int64(len(found))).Msg("SearchEqual")
	if msg := mgr.idxLog.Trace(); msg.Enabled() {
		msg.Str("ids", fmt.Sprint(found)).Msg("IDs")
	}
	return found
}

// SearchPrefix returns all zettel that have a word with the given prefix.
// The prefix must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SearchPrefix(prefix string) id.Set {
	found := mgr.idxStore.SearchPrefix(prefix)
	mgr.idxLog.Debug().Str("prefix", prefix).Int("found", int64(len(found))).Msg("SearchPrefix")
	if msg := mgr.idxLog.Trace(); msg.Enabled() {
		msg.Str("ids", fmt.Sprint(found)).Msg("IDs")
	}
	return found
}

// SearchSuffix returns all zettel that have a word with the given suffix.
// The suffix must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SearchSuffix(suffix string) id.Set {
	found := mgr.idxStore.SearchSuffix(suffix)
	mgr.idxLog.Debug().Str("suffix", suffix).Int("found", int64(len(found))).Msg("SearchSuffix")
	if msg := mgr.idxLog.Trace(); msg.Enabled() {
		msg.Str("ids", fmt.Sprint(found)).Msg("IDs")
	}
	return found
}

// SearchContains returns all zettel that contains the given string.
// The string must be normalized through Unicode NKFD, trimmed and not empty.
func (mgr *Manager) SearchContains(s string) id.Set {
	found := mgr.idxStore.SearchContains(s)
	mgr.idxLog.Debug().Str("s", s).Int("found", int64(len(found))).Msg("SearchContains")
	if msg := mgr.idxLog.Trace(); msg.Enabled() {
		msg.Str("ids", fmt.Sprint(found)).Msg("IDs")
	}
	return found
}

// idxIndexer runs in the background and updates the index data structures.
// This is the main service of the idxIndexer.
func (mgr *Manager) idxIndexer() {
	// Something may panic. Ensure a running indexer.
	defer func() {
		if ri := recover(); ri != nil {
			kernel.Main.LogRecover("Indexer", ri)
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
			mgr.idxLog.Debug().Msg("reload")
			roomNum = 0
			zids, err := mgr.FetchZids(ctx)
			if err == nil {
				start = time.Now()
				if rno := mgr.idxAr.Reload(zids); rno > 0 {
					roomNum = rno
				}
				mgr.idxMx.Lock()
				mgr.idxLastReload = time.Now().Local()
				mgr.idxSinceReload = 0
				mgr.idxMx.Unlock()
			}
		case arZettel:
			mgr.idxLog.Debug().Zid(zid).Msg("zettel")
			zettel, err := mgr.GetZettel(ctx, zid)
			if err != nil {
				// Zettel was deleted or is not accessible b/c of other reasons
				mgr.idxLog.Trace().Zid(zid).Msg("delete")
				mgr.idxMx.Lock()
				mgr.idxSinceReload++
				mgr.idxMx.Unlock()
				mgr.idxDeleteZettel(ctx, zid)
				continue
			}
			mgr.idxLog.Trace().Zid(zid).Msg("update")
			mgr.idxMx.Lock()
			if arRoomNum == roomNum {
				mgr.idxDurReload = time.Since(start)
			}
			mgr.idxSinceReload++
			mgr.idxMx.Unlock()
			mgr.idxUpdateZettel(ctx, zettel)
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

func (mgr *Manager) idxUpdateZettel(ctx context.Context, zettel zettel.Zettel) {
	var cData collectData
	cData.initialize()
	collectZettelIndexData(parser.ParseZettel(ctx, zettel, "", mgr.rtConfig), &cData)

	m := zettel.Meta
	zi := store.NewZettelIndex(m)
	mgr.idxCollectFromMeta(ctx, m, zi, &cData)
	mgr.idxProcessData(ctx, zi, &cData)
	toCheck := mgr.idxStore.UpdateReferences(ctx, zi)
	mgr.idxCheckZettel(toCheck)
}

func (mgr *Manager) idxCollectFromMeta(ctx context.Context, m *meta.Meta, zi *store.ZettelIndex, cData *collectData) {
	for _, pair := range m.ComputedPairs() {
		descr := meta.GetDescription(pair.Key)
		if descr.IsProperty() {
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
			is := parser.ParseMetadata(pair.Value)
			collectInlineIndexData(&is, cData)
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
		if mgr.HasZettel(ctx, ref) {
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
	if !mgr.HasZettel(ctx, zid) {
		zi.AddDeadRef(zid)
		return
	}
	if inverseKey == "" {
		zi.AddBackRef(zid)
		return
	}
	zi.AddInverseRef(inverseKey, zid)
}

func (mgr *Manager) idxRenameZettel(ctx context.Context, curZid, newZid id.Zid) {
	toCheck := mgr.idxStore.RenameZettel(ctx, curZid, newZid)
	mgr.idxCheckZettel(toCheck)
}

func (mgr *Manager) idxDeleteZettel(ctx context.Context, zid id.Zid) {
	toCheck := mgr.idxStore.DeleteZettel(ctx, zid)
	mgr.idxCheckZettel(toCheck)
}

func (mgr *Manager) idxCheckZettel(s id.Set) {
	for zid := range s {
		mgr.idxAr.EnqueueZettel(zid)
	}
}

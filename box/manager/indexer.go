//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/strfun"
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
		case arUpdate:
			mgr.idxLog.Debug().Zid(zid).Msg("update")
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
			mgr.idxLog.Debug().Zid(zid).Msg("delete")
			if _, err := mgr.GetMeta(ctx, zid); err == nil {
				// Zettel was not deleted. This might occur, if zettel was
				// deleted in secondary dirbox, but is still present in
				// first dirbox (or vice versa). Re-index zettel in case
				// a hidden zettel was recovered
				mgr.idxLog.Debug().Zid(zid).Msg("not deleted")
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
	var cData collectData
	cData.initialize()
	collectZettelIndexData(parser.ParseZettel(zettel, "", mgr.rtConfig), &cData)

	m := zettel.Meta
	zi := store.NewZettelIndex(m.Zid)
	mgr.idxCollectFromMeta(ctx, m, zi, &cData)
	mgr.idxProcessData(ctx, zi, &cData)
	toCheck := mgr.idxStore.UpdateReferences(ctx, zi)
	mgr.idxCheckZettel(toCheck)
}

func (mgr *Manager) idxCollectFromMeta(ctx context.Context, m *meta.Meta, zi *store.ZettelIndex, cData *collectData) {
	for _, pair := range m.Pairs() {
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
		if _, err := mgr.GetMeta(ctx, ref); err == nil {
			zi.AddBackRef(ref)
		} else {
			zi.AddDeadRef(ref)
		}
	}
	zi.SetWords(cData.words)
	zi.SetUrls(cData.urls)
	zi.SetITags(cData.itags)
}

func (mgr *Manager) idxUpdateValue(ctx context.Context, inverseKey, value string, zi *store.ZettelIndex) {
	zid, err := id.Parse(value)
	if err != nil {
		return
	}
	if _, err = mgr.GetMeta(ctx, zid); err != nil {
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

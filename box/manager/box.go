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
	"errors"
	"strings"

	"zettelstore.de/z/box"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// Conatains all box.Box related functions

// Location returns some information where the box is located.
func (mgr *Manager) Location() string {
	if len(mgr.boxes) <= 2 {
		return "NONE"
	}
	var sb strings.Builder
	for i := 0; i < len(mgr.boxes)-2; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(mgr.boxes[i].Location())
	}
	return sb.String()
}

// CanCreateZettel returns true, if box could possibly create a new zettel.
func (mgr *Manager) CanCreateZettel(ctx context.Context) bool {
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	return mgr.State() == box.StartStateStarted && mgr.boxes[0].CanCreateZettel(ctx)
}

// CreateZettel creates a new zettel.
func (mgr *Manager) CreateZettel(ctx context.Context, zettel zettel.Zettel) (id.Zid, error) {
	mgr.mgrLog.Debug().Msg("CreateZettel")
	if mgr.State() != box.StartStateStarted {
		return id.Invalid, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	zid, err := mgr.boxes[0].CreateZettel(ctx, zettel)
	if err == nil {
		mgr.idxUpdateZettel(ctx, zettel)
	}
	return zid, err
}

// GetZettel retrieves a specific zettel.
func (mgr *Manager) GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error) {
	mgr.mgrLog.Debug().Zid(zid).Msg("GetZettel")
	if mgr.State() != box.StartStateStarted {
		return zettel.Zettel{}, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for i, p := range mgr.boxes {
		var errZNF box.ErrZettelNotFound
		if z, err := p.GetZettel(ctx, zid); !errors.As(err, &errZNF) {
			if err == nil {
				mgr.Enrich(ctx, z.Meta, i+1)
			}
			return z, err
		}
	}
	return zettel.Zettel{}, box.ErrZettelNotFound{Zid: zid}
}

// GetAllZettel retrieves a specific zettel from all managed boxes.
func (mgr *Manager) GetAllZettel(ctx context.Context, zid id.Zid) ([]zettel.Zettel, error) {
	mgr.mgrLog.Debug().Zid(zid).Msg("GetAllZettel")
	if mgr.State() != box.StartStateStarted {
		return nil, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	var result []zettel.Zettel
	for i, p := range mgr.boxes {
		if z, err := p.GetZettel(ctx, zid); err == nil {
			mgr.Enrich(ctx, z.Meta, i+1)
			result = append(result, z)
		}
	}
	return result, nil
}

// FetchZids returns the set of all zettel identifer managed by the box.
func (mgr *Manager) FetchZids(ctx context.Context) (id.Set, error) {
	mgr.mgrLog.Debug().Msg("FetchZids")
	if mgr.State() != box.StartStateStarted {
		return nil, box.ErrStopped
	}
	result := id.Set{}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for _, p := range mgr.boxes {
		err := p.ApplyZid(ctx, func(zid id.Zid) { result.Add(zid) }, func(id.Zid) bool { return true })
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (mgr *Manager) HasZettel(ctx context.Context, zid id.Zid) bool {
	mgr.mgrLog.Debug().Zid(zid).Msg("HasZettel")
	if mgr.State() != box.StartStateStarted {
		return false
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for _, bx := range mgr.boxes {
		if bx.HasZettel(ctx, zid) {
			return true
		}
	}
	return false
}

func (mgr *Manager) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	mgr.mgrLog.Debug().Zid(zid).Msg("GetMeta")
	if mgr.State() != box.StartStateStarted {
		return nil, box.ErrStopped
	}

	m, err := mgr.idxStore.GetMeta(ctx, zid)
	if err != nil {
		return nil, err
	}
	mgr.Enrich(ctx, m, 0)
	return m, nil
}

// SelectMeta returns all zettel meta data that match the selection
// criteria. The result is ordered by descending zettel id.
func (mgr *Manager) SelectMeta(ctx context.Context, metaSeq []*meta.Meta, q *query.Query) ([]*meta.Meta, error) {
	if msg := mgr.mgrLog.Debug(); msg.Enabled() {
		msg.Str("query", q.String()).Msg("SelectMeta")
	}
	if mgr.State() != box.StartStateStarted {
		return nil, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()

	compSearch := q.RetrieveAndCompile(ctx, mgr, metaSeq)
	if result := compSearch.Result(); result != nil {
		mgr.mgrLog.Trace().Int("count", int64(len(result))).Msg("found without ApplyMeta")
		return result, nil
	}
	selected := map[id.Zid]*meta.Meta{}
	for _, term := range compSearch.Terms {
		rejected := id.Set{}
		handleMeta := func(m *meta.Meta) {
			zid := m.Zid
			if rejected.ContainsOrNil(zid) {
				mgr.mgrLog.Trace().Zid(zid).Msg("SelectMeta/alreadyRejected")
				return
			}
			if _, ok := selected[zid]; ok {
				mgr.mgrLog.Trace().Zid(zid).Msg("SelectMeta/alreadySelected")
				return
			}
			if compSearch.PreMatch(m) && term.Match(m) {
				selected[zid] = m
				mgr.mgrLog.Trace().Zid(zid).Msg("SelectMeta/match")
			} else {
				rejected.Add(zid)
				mgr.mgrLog.Trace().Zid(zid).Msg("SelectMeta/reject")
			}
		}
		for _, p := range mgr.boxes {
			if err2 := p.ApplyMeta(ctx, handleMeta, term.Retrieve); err2 != nil {
				return nil, err2
			}
		}
	}
	result := make([]*meta.Meta, 0, len(selected))
	for _, m := range selected {
		result = append(result, m)
	}
	result = compSearch.AfterSearch(result)
	mgr.mgrLog.Trace().Int("count", int64(len(result))).Msg("found with ApplyMeta")
	return result, nil
}

// CanUpdateZettel returns true, if box could possibly update the given zettel.
func (mgr *Manager) CanUpdateZettel(ctx context.Context, zettel zettel.Zettel) bool {
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	return mgr.State() == box.StartStateStarted && mgr.boxes[0].CanUpdateZettel(ctx, zettel)
}

// UpdateZettel updates an existing zettel.
func (mgr *Manager) UpdateZettel(ctx context.Context, zettel zettel.Zettel) error {
	mgr.mgrLog.Debug().Zid(zettel.Meta.Zid).Msg("UpdateZettel")
	if mgr.State() != box.StartStateStarted {
		return box.ErrStopped
	}
	// Remove all (computed) properties from metadata before storing the zettel.
	zettel.Meta = zettel.Meta.Clone()
	for _, p := range zettel.Meta.ComputedPairsRest() {
		if mgr.propertyKeys.Has(p.Key) {
			zettel.Meta.Delete(p.Key)
		}
	}
	if err := mgr.boxes[0].UpdateZettel(ctx, zettel); err != nil {
		return err
	}
	mgr.idxUpdateZettel(ctx, zettel)
	return nil
}

// AllowRenameZettel returns true, if box will not disallow renaming the zettel.
func (mgr *Manager) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	if mgr.State() != box.StartStateStarted {
		return false
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for _, p := range mgr.boxes {
		if !p.AllowRenameZettel(ctx, zid) {
			return false
		}
	}
	return true
}

// RenameZettel changes the current zid to a new zid.
func (mgr *Manager) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	mgr.mgrLog.Debug().Zid(curZid).Zid(newZid).Msg("RenameZettel")
	if mgr.State() != box.StartStateStarted {
		return box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for i, p := range mgr.boxes {
		err := p.RenameZettel(ctx, curZid, newZid)
		var errZNF box.ErrZettelNotFound
		if err != nil && !errors.As(err, &errZNF) {
			for j := 0; j < i; j++ {
				mgr.boxes[j].RenameZettel(ctx, newZid, curZid)
			}
			return err
		}
	}
	mgr.idxRenameZettel(ctx, curZid, newZid)
	return nil
}

// CanDeleteZettel returns true, if box could possibly delete the given zettel.
func (mgr *Manager) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	if mgr.State() != box.StartStateStarted {
		return false
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for _, p := range mgr.boxes {
		if p.CanDeleteZettel(ctx, zid) {
			return true
		}
	}
	return false
}

// DeleteZettel removes the zettel from the box.
func (mgr *Manager) DeleteZettel(ctx context.Context, zid id.Zid) error {
	mgr.mgrLog.Debug().Zid(zid).Msg("DeleteZettel")
	if mgr.State() != box.StartStateStarted {
		return box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for _, p := range mgr.boxes {
		err := p.DeleteZettel(ctx, zid)
		if err == nil {
			mgr.idxDeleteZettel(ctx, zid)
			return nil
		}
		var errZNF box.ErrZettelNotFound
		if !errors.As(err, &errZNF) && !errors.Is(err, box.ErrReadOnly) {
			return err
		}
	}
	return box.ErrZettelNotFound{Zid: zid}
}

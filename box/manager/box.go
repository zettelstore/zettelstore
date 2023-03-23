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
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/query"
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
	return mgr.getState() == mgrStateStarted && mgr.boxes[0].CanCreateZettel(ctx)
}

// CreateZettel creates a new zettel.
func (mgr *Manager) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	mgr.mgrLog.Debug().Msg("CreateZettel")
	if mgr.getState() != mgrStateStarted {
		return id.Invalid, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	return mgr.boxes[0].CreateZettel(ctx, zettel)
}

// GetZettel retrieves a specific zettel.
func (mgr *Manager) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	mgr.mgrLog.Debug().Zid(zid).Msg("GetZettel")
	if mgr.getState() != mgrStateStarted {
		return domain.Zettel{}, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for i, p := range mgr.boxes {
		if z, err := p.GetZettel(ctx, zid); err != box.ErrNotFound {
			if err == nil {
				mgr.Enrich(ctx, z.Meta, i+1)
			}
			return z, err
		}
	}
	return domain.Zettel{}, box.ErrNotFound
}

// GetAllZettel retrieves a specific zettel from all managed boxes.
func (mgr *Manager) GetAllZettel(ctx context.Context, zid id.Zid) ([]domain.Zettel, error) {
	mgr.mgrLog.Debug().Zid(zid).Msg("GetAllZettel")
	if mgr.getState() != mgrStateStarted {
		return nil, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	var result []domain.Zettel
	for i, p := range mgr.boxes {
		if z, err := p.GetZettel(ctx, zid); err == nil {
			mgr.Enrich(ctx, z.Meta, i+1)
			result = append(result, z)
		}
	}
	return result, nil
}

// GetMeta retrieves just the meta data of a specific zettel.
func (mgr *Manager) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	mgr.mgrLog.Debug().Zid(zid).Msg("GetMeta")
	if mgr.getState() != mgrStateStarted {
		// panic(zid)
		return nil, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	return mgr.doGetMeta(ctx, zid)
}

func (mgr *Manager) doGetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	for i, p := range mgr.boxes {
		if m, err := p.GetMeta(ctx, zid); err != box.ErrNotFound {
			if err == nil {
				mgr.Enrich(ctx, m, i+1)
			}
			return m, err
		}
	}
	return nil, box.ErrNotFound
}

// GetAllMeta retrieves the meta data of a specific zettel from all managed boxes.
func (mgr *Manager) GetAllMeta(ctx context.Context, zid id.Zid) ([]*meta.Meta, error) {
	mgr.mgrLog.Debug().Zid(zid).Msg("GetAllMeta")
	if mgr.getState() != mgrStateStarted {
		return nil, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	var result []*meta.Meta
	for i, p := range mgr.boxes {
		if m, err := p.GetMeta(ctx, zid); err == nil {
			mgr.Enrich(ctx, m, i+1)
			result = append(result, m)
		}
	}
	return result, nil
}

// FetchZids returns the set of all zettel identifer managed by the box.
func (mgr *Manager) FetchZids(ctx context.Context) (id.Set, error) {
	mgr.mgrLog.Debug().Msg("FetchZids")
	if mgr.getState() != mgrStateStarted {
		return nil, box.ErrStopped
	}
	result := id.Set{}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for _, p := range mgr.boxes {
		err := p.ApplyZid(ctx, func(zid id.Zid) { result.Zid(zid) }, func(id.Zid) bool { return true })
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

type metaMap map[id.Zid]*meta.Meta

// SelectMeta returns all zettel meta data that match the selection
// criteria. The result is ordered by descending zettel id.
func (mgr *Manager) SelectMeta(ctx context.Context, q *query.Query) ([]*meta.Meta, error) {
	if msg := mgr.mgrLog.Debug(); msg.Enabled() {
		msg.Str("query", q.String()).Msg("SelectMeta")
	}
	if mgr.getState() != mgrStateStarted {
		return nil, box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()

	compSearch := q.RetrieveAndCompile(mgr)
	selected := metaMap{}
	for _, term := range compSearch.Terms {
		rejected := id.Set{}
		handleMeta := func(m *meta.Meta) {
			zid := m.Zid
			if rejected.Contains(zid) {
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
				rejected.Zid(zid)
				mgr.mgrLog.Trace().Zid(zid).Msg("SelectMeta/reject")
			}
		}
		for _, p := range mgr.boxes {
			if err := p.ApplyMeta(ctx, handleMeta, term.Retrieve); err != nil {
				return nil, err
			}
		}
	}
	result := make([]*meta.Meta, 0, len(selected))
	for _, m := range selected {
		result = append(result, m)
	}
	return q.Sort(result), nil
}

// CanUpdateZettel returns true, if box could possibly update the given zettel.
func (mgr *Manager) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	return mgr.getState() == mgrStateStarted && mgr.boxes[0].CanUpdateZettel(ctx, zettel)
}

// UpdateZettel updates an existing zettel.
func (mgr *Manager) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	mgr.mgrLog.Debug().Zid(zettel.Meta.Zid).Msg("UpdateZettel")
	if mgr.getState() != mgrStateStarted {
		return box.ErrStopped
	}
	// Remove all (computed) properties from metadata before storing the zettel.
	zettel.Meta = zettel.Meta.Clone()
	for _, p := range zettel.Meta.ComputedPairsRest() {
		if mgr.propertyKeys.Has(p.Key) {
			zettel.Meta.Delete(p.Key)
		}
	}
	return mgr.boxes[0].UpdateZettel(ctx, zettel)
}

// AllowRenameZettel returns true, if box will not disallow renaming the zettel.
func (mgr *Manager) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	if mgr.getState() != mgrStateStarted {
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
	if mgr.getState() != mgrStateStarted {
		return box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for i, p := range mgr.boxes {
		err := p.RenameZettel(ctx, curZid, newZid)
		if err != nil && !errors.Is(err, box.ErrNotFound) {
			for j := 0; j < i; j++ {
				mgr.boxes[j].RenameZettel(ctx, newZid, curZid)
			}
			return err
		}
	}
	return nil
}

// CanDeleteZettel returns true, if box could possibly delete the given zettel.
func (mgr *Manager) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	if mgr.getState() != mgrStateStarted {
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
	if mgr.getState() != mgrStateStarted {
		return box.ErrStopped
	}
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	for _, p := range mgr.boxes {
		err := p.DeleteZettel(ctx, zid)
		if err == nil {
			return nil
		}
		if !errors.Is(err, box.ErrNotFound) && !errors.Is(err, box.ErrReadOnly) {
			return err
		}
	}
	return box.ErrNotFound
}

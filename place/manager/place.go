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
	"errors"
	"sort"
	"strings"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/search"
)

// Conatains all place.Place related functions

// Location returns some information where the place is located.
func (mgr *Manager) Location() string {
	if len(mgr.subplaces) < 2 {
		return mgr.subplaces[0].Location()
	}
	var sb strings.Builder
	for i := 0; i < len(mgr.subplaces)-2; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(mgr.subplaces[i].Location())
	}
	return sb.String()
}

// CanCreateZettel returns true, if place could possibly create a new zettel.
func (mgr *Manager) CanCreateZettel(ctx context.Context) bool {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	return mgr.started && mgr.subplaces[0].CanCreateZettel(ctx)
}

// CreateZettel creates a new zettel.
func (mgr *Manager) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return id.Invalid, place.ErrStopped
	}
	return mgr.subplaces[0].CreateZettel(ctx, zettel)
}

// GetZettel retrieves a specific zettel.
func (mgr *Manager) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return domain.Zettel{}, place.ErrStopped
	}
	for _, p := range mgr.subplaces {
		if z, err := p.GetZettel(ctx, zid); err != place.ErrNotFound {
			if err == nil {
				mgr.Enrich(ctx, z.Meta)
			}
			return z, err
		}
	}
	return domain.Zettel{}, place.ErrNotFound
}

// GetMeta retrieves just the meta data of a specific zettel.
func (mgr *Manager) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return nil, place.ErrStopped
	}
	for _, p := range mgr.subplaces {
		if m, err := p.GetMeta(ctx, zid); err != place.ErrNotFound {
			if err == nil {
				mgr.Enrich(ctx, m)
			}
			return m, err
		}
	}
	return nil, place.ErrNotFound
}

// FetchZids returns the set of all zettel identifer managed by the place.
func (mgr *Manager) FetchZids(ctx context.Context) (result id.Set, err error) {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return nil, place.ErrStopped
	}
	for _, p := range mgr.subplaces {
		zids, err := p.FetchZids(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = zids
		} else if len(result) <= len(zids) {
			for zid := range result {
				zids[zid] = true
			}
			result = zids
		} else {
			for zid := range zids {
				result[zid] = true
			}
		}
	}
	return result, nil
}

// SelectMeta returns all zettel meta data that match the selection
// criteria. The result is ordered by descending zettel id.
func (mgr *Manager) SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error) {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return nil, place.ErrStopped
	}
	var result []*meta.Meta
	match := s.CompileMatch(startup.Indexer())
	for _, p := range mgr.subplaces {
		selected, err := p.SelectMeta(ctx, match)
		if err != nil {
			return nil, err
		}
		sort.Slice(selected, func(i, j int) bool { return selected[i].Zid > selected[j].Zid })
		if len(result) == 0 {
			result = selected
		} else {
			result = place.MergeSorted(result, selected)
		}
	}
	if s == nil {
		return result, nil
	}
	return s.Sort(result), nil
}

// CanUpdateZettel returns true, if place could possibly update the given zettel.
func (mgr *Manager) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	return mgr.started && mgr.subplaces[0].CanUpdateZettel(ctx, zettel)
}

// UpdateZettel updates an existing zettel.
func (mgr *Manager) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return place.ErrStopped
	}
	// Remove all (computed) properties from metadata before storing the zettel.
	zettel.Meta = zettel.Meta.Clone()
	for _, p := range zettel.Meta.PairsRest(true) {
		if mgr.propertyKeys[p.Key] {
			zettel.Meta.Delete(p.Key)
		}
	}
	return mgr.subplaces[0].UpdateZettel(ctx, zettel)
}

// AllowRenameZettel returns true, if place will not disallow renaming the zettel.
func (mgr *Manager) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return false
	}
	for _, p := range mgr.subplaces {
		if !p.AllowRenameZettel(ctx, zid) {
			return false
		}
	}
	return true
}

// RenameZettel changes the current zid to a new zid.
func (mgr *Manager) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return place.ErrStopped
	}
	for i, p := range mgr.subplaces {
		err := p.RenameZettel(ctx, curZid, newZid)
		if err != nil && !errors.Is(err, place.ErrNotFound) {
			for j := 0; j < i; j++ {
				mgr.subplaces[j].RenameZettel(ctx, newZid, curZid)
			}
			return err
		}
	}
	return nil
}

// CanDeleteZettel returns true, if place could possibly delete the given zettel.
func (mgr *Manager) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return false
	}
	for _, p := range mgr.subplaces {
		if p.CanDeleteZettel(ctx, zid) {
			return true
		}
	}
	return false
}

// DeleteZettel removes the zettel from the place.
func (mgr *Manager) DeleteZettel(ctx context.Context, zid id.Zid) error {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return place.ErrStopped
	}
	for _, p := range mgr.subplaces {
		err := p.DeleteZettel(ctx, zid)
		if err == nil {
			return nil
		}
		if !errors.Is(err, place.ErrNotFound) && !errors.Is(err, place.ErrReadOnly) {
			return err
		}
	}
	return place.ErrNotFound
}

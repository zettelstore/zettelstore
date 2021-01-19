//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various places of a Zettelstore.
package manager

import (
	"context"
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

// Connect returns a handle to the specified place
func Connect(rawURL string, readonlyMode bool, f MetaFilter, ob place.ObserverFunc) (place.Place, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "dir"
	}
	if readonlyMode {
		// TODO: the following is wrong under some circumstances:
		// 1. fragment is set
		if q := u.Query(); len(q) == 0 {
			rawURL += "?readonly"
		} else if _, ok := q["readonly"]; !ok {
			rawURL += "&readonly"
		}
		if u, err = url.Parse(rawURL); err != nil {
			return nil, err
		}
	}

	if create, ok := registry[u.Scheme]; ok {
		return create(u, f, ob)
	}
	return nil, &ErrInvalidScheme{u.Scheme}
}

// ErrInvalidScheme is returned if there is no place with the given scheme
type ErrInvalidScheme struct{ Scheme string }

func (err *ErrInvalidScheme) Error() string { return "Invalid scheme: " + err.Scheme }

type createFunc func(*url.URL, MetaFilter, place.ObserverFunc) (place.Place, error)

var registry = map[string]createFunc{}

// Register the encoder for later retrieval.
func Register(scheme string, create createFunc) {
	if _, ok := registry[scheme]; ok {
		log.Fatalf("Place with scheme %q already registered", scheme)
	}
	registry[scheme] = create
}

// GetSchemes returns all registered scheme, ordered by scheme string.
func GetSchemes() []string {
	result := make([]string, 0, len(registry))
	for scheme := range registry {
		result = append(result, scheme)
	}
	sort.Strings(result)
	return result
}

// Manager is a coordinating place.
type Manager struct {
	started    bool
	placeURIs  []url.URL
	subplaces  []place.Place
	filter     MetaFilter
	observers  []place.ObserverFunc
	mxObserver sync.RWMutex
}

// New creates a new managing place.
func New(placeURIs []string, readonlyMode bool) (*Manager, error) {
	filter := newFilter()
	mgr := &Manager{
		filter: filter,
	}
	subplaces := make([]place.Place, 0, len(placeURIs)+2)
	for _, uri := range placeURIs {
		p, err := Connect(uri, readonlyMode, filter, mgr.observe)
		if err != nil {
			return nil, err
		}
		subplaces = append(subplaces, p)
	}
	constplace, err := registry[" const"](nil, filter, mgr.observe)
	if err != nil {
		return nil, err
	}
	progplace, err := registry[" prog"](nil, filter, mgr.observe)
	if err != nil {
		return nil, err
	}
	subplaces = append(subplaces, constplace, progplace)
	mgr.subplaces = subplaces
	return mgr, nil
}

// RegisterChangeObserver registers an observer that will be notified
// if a zettel was found to be changed.
func (mgr *Manager) RegisterChangeObserver(f place.ObserverFunc) {
	if f != nil {
		mgr.mxObserver.Lock()
		mgr.observers = append(mgr.observers, f)
		mgr.mxObserver.Unlock()
	}
}

func (mgr *Manager) observe(reason place.ChangeReason, zid id.Zid) {
	mgr.mxObserver.RLock()
	observers := mgr.observers
	mgr.mxObserver.RUnlock()
	for _, ob := range observers {
		ob(reason, zid)
	}
}

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

// Start the place. Now all other functions of the place are allowed.
// Starting an already started place is not allowed.
func (mgr *Manager) Start(ctx context.Context) error {
	if mgr.started {
		return place.ErrStarted
	}
	for i := len(mgr.subplaces) - 1; i >= 0; i-- {
		if err := mgr.subplaces[i].Start(ctx); err != nil {
			for j := i + 1; j < len(mgr.subplaces); j++ {
				mgr.subplaces[j].Stop(ctx)
			}
			return err
		}
	}
	mgr.started = true
	return nil
}

// Stop the started place. Now only the Start() function is allowed.
func (mgr *Manager) Stop(ctx context.Context) error {
	if !mgr.started {
		return place.ErrStopped
	}
	var err error
	for _, p := range mgr.subplaces {
		if err1 := p.Stop(ctx); err1 != nil && err == nil {
			err = err1
		}
	}
	mgr.started = false
	return err
}

// CanCreateZettel returns true, if place could possibly create a new zettel.
func (mgr *Manager) CanCreateZettel(ctx context.Context) bool {
	return mgr.started && mgr.subplaces[0].CanCreateZettel(ctx)
}

// CreateZettel creates a new zettel.
func (mgr *Manager) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	if !mgr.started {
		return id.Invalid, place.ErrStopped
	}
	return mgr.subplaces[0].CreateZettel(ctx, zettel)
}

// GetZettel retrieves a specific zettel.
func (mgr *Manager) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	if !mgr.started {
		return domain.Zettel{}, place.ErrStopped
	}
	for _, p := range mgr.subplaces {
		if z, err := p.GetZettel(ctx, zid); err != place.ErrNotFound {
			mgr.filter.UpdateProperties(z.Meta)
			return z, err
		}
	}
	return domain.Zettel{}, place.ErrNotFound
}

// GetMeta retrieves just the meta data of a specific zettel.
func (mgr *Manager) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	if !mgr.started {
		return nil, place.ErrStopped
	}
	for _, p := range mgr.subplaces {
		if m, err := p.GetMeta(ctx, zid); err != place.ErrNotFound {
			mgr.filter.UpdateProperties(m)
			return m, err
		}
	}
	return nil, place.ErrNotFound
}

// FetchZids returns the set of all zettel identifer managed by the place.
func (mgr *Manager) FetchZids(ctx context.Context) (result map[id.Zid]bool, err error) {
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
func (mgr *Manager) SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error) {
	if !mgr.started {
		return nil, place.ErrStopped
	}
	var result []*meta.Meta
	for _, p := range mgr.subplaces {
		selected, err := p.SelectMeta(ctx, f, nil)
		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			result = selected
		} else {
			result = place.MergeSorted(result, selected)
		}
	}
	if s == nil {
		return result, nil
	}
	return place.ApplySorter(result, s), nil
}

// CanUpdateZettel returns true, if place could possibly update the given zettel.
func (mgr *Manager) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return mgr.started && mgr.subplaces[0].CanUpdateZettel(ctx, zettel)
}

// UpdateZettel updates an existing zettel.
func (mgr *Manager) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	if !mgr.started {
		return place.ErrStopped
	}
	zettel.Meta = zettel.Meta.Clone()
	mgr.filter.RemoveProperties(zettel.Meta)
	return mgr.subplaces[0].UpdateZettel(ctx, zettel)
}

// AllowRenameZettel returns true, if place will not disallow renaming the zettel.
func (mgr *Manager) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
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
	if !mgr.started {
		return place.ErrStopped
	}
	for i, p := range mgr.subplaces {
		if err := p.RenameZettel(ctx, curZid, newZid); err != nil && err != place.ErrNotFound {
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
	if !mgr.started {
		return place.ErrStopped
	}
	for _, p := range mgr.subplaces {
		if err := p.DeleteZettel(ctx, zid); err != place.ErrNotFound && err != place.ErrReadOnly {
			return err
		}
	}
	return place.ErrNotFound
}

// Reload clears all caches, reloads all internal data to reflect changes
// that were possibly undetected.
func (mgr *Manager) Reload(ctx context.Context) error {
	var err error
	for _, p := range mgr.subplaces {
		if err1 := p.Reload(ctx); err1 != nil && err == nil {
			err = err1
		}
	}
	return err
}

// ReadStats populates st with place statistics
func (mgr *Manager) ReadStats(st *place.Stats) {
	subStats := make([]place.Stats, len(mgr.subplaces))
	for i, p := range mgr.subplaces {
		p.ReadStats(&subStats[i])
	}

	st.ReadOnly = true
	sumZettel := 0
	for _, sst := range subStats {
		if !sst.ReadOnly {
			st.ReadOnly = false
		}
		sumZettel += sst.Zettel
	}
	st.Zettel = sumZettel
}

// NumPlaces returns the number of managed places.
func (mgr *Manager) NumPlaces() int { return len(mgr.subplaces) }

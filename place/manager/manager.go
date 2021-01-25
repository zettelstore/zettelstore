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
	"zettelstore.de/z/index"
	"zettelstore.de/z/place"
)

// ConnectData contains all administration related values.
type ConnectData struct {
	Filter index.MetaFilter
	Notify chan<- place.ChangeInfo
}

// Connect returns a handle to the specified place
func Connect(rawURL string, readonlyMode bool, cdata *ConnectData) (place.Place, error) {
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
		return create(u, cdata)
	}
	return nil, &ErrInvalidScheme{u.Scheme}
}

// ErrInvalidScheme is returned if there is no place with the given scheme
type ErrInvalidScheme struct{ Scheme string }

func (err *ErrInvalidScheme) Error() string { return "Invalid scheme: " + err.Scheme }

type createFunc func(*url.URL, *ConnectData) (place.Place, error)

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
	mx         sync.RWMutex
	started    bool
	placeURIs  []url.URL
	subplaces  []place.Place
	filter     index.MetaFilter
	observers  []func(place.ChangeInfo)
	mxObserver sync.RWMutex
	done       chan struct{}
	infos      chan place.ChangeInfo
}

// New creates a new managing place.
func New(placeURIs []string, readonlyMode bool, filter index.MetaFilter) (*Manager, error) {
	mgr := &Manager{
		filter: filter,
		infos:  make(chan place.ChangeInfo, len(placeURIs)*10),
	}
	cdata := ConnectData{Filter: filter, Notify: mgr.infos}
	subplaces := make([]place.Place, 0, len(placeURIs)+2)
	for _, uri := range placeURIs {
		p, err := Connect(uri, readonlyMode, &cdata)
		if err != nil {
			return nil, err
		}
		subplaces = append(subplaces, p)
	}
	constplace, err := registry[" const"](nil, &cdata)
	if err != nil {
		return nil, err
	}
	progplace, err := registry[" prog"](nil, &cdata)
	if err != nil {
		return nil, err
	}
	subplaces = append(subplaces, constplace, progplace)
	mgr.subplaces = subplaces
	return mgr, nil
}

// RegisterObserver registers an observer that will be notified
// if a zettel was found to be changed.
func (mgr *Manager) RegisterObserver(f func(place.ChangeInfo)) {
	if f != nil {
		mgr.mxObserver.Lock()
		mgr.observers = append(mgr.observers, f)
		mgr.mxObserver.Unlock()
	}
}

func (mgr *Manager) notifyObserver(ci place.ChangeInfo) {
	mgr.mxObserver.RLock()
	observers := mgr.observers
	mgr.mxObserver.RUnlock()
	for _, ob := range observers {
		ob(ci)
	}
}

func notifier(notify func(place.ChangeInfo), infos <-chan place.ChangeInfo, done <-chan struct{}) {
	// The call to notify may panic. Ensure a running notifier.
	defer func() {
		if err := recover(); err != nil {
			go notifier(notify, infos, done)
		}
	}()

	for {
		select {
		case ci, ok := <-infos:
			if ok {
				notify(ci)
			}
		case _, ok := <-done:
			if !ok {
				return
			}
		}
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
	mgr.mx.Lock()
	if mgr.started {
		mgr.mx.Unlock()
		return place.ErrStarted
	}
	for i := len(mgr.subplaces) - 1; i >= 0; i-- {
		if ssi, ok := mgr.subplaces[i].(place.StartStopper); ok {
			if err := ssi.Start(ctx); err != nil {
				for j := i + 1; j < len(mgr.subplaces); j++ {
					if ssj, ok := mgr.subplaces[j].(place.StartStopper); ok {
						ssj.Stop(ctx)
					}
				}
				mgr.mx.Unlock()
				return err
			}
		}
	}
	mgr.done = make(chan struct{})
	go notifier(mgr.notifyObserver, mgr.infos, mgr.done)
	mgr.started = true
	mgr.mx.Unlock()
	mgr.infos <- place.ChangeInfo{Reason: place.OnReload, Zid: id.Invalid}
	return nil
}

// Stop the started place. Now only the Start() function is allowed.
func (mgr *Manager) Stop(ctx context.Context) error {
	mgr.mx.Lock()
	defer mgr.mx.Unlock()
	if !mgr.started {
		return place.ErrStopped
	}
	close(mgr.done)
	mgr.done = nil
	var err error
	for _, p := range mgr.subplaces {
		if ss, ok := p.(place.StartStopper); ok {
			if err1 := ss.Stop(ctx); err1 != nil && err == nil {
				err = err1
			}
		}
	}
	mgr.started = false
	return err
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
			mgr.filter.Update(ctx, z.Meta)
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
			mgr.filter.Update(ctx, m)
			return m, err
		}
	}
	return nil, place.ErrNotFound
}

// FetchZids returns the set of all zettel identifer managed by the place.
func (mgr *Manager) FetchZids(ctx context.Context) (result map[id.Zid]bool, err error) {
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
func (mgr *Manager) SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error) {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
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
	zettel.Meta = zettel.Meta.Clone()
	mgr.filter.Remove(ctx, zettel.Meta)
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
		if err := p.DeleteZettel(ctx, zid); err != place.ErrNotFound && err != place.ErrReadOnly {
			return err
		}
	}
	return place.ErrNotFound
}

// Reload clears all caches, reloads all internal data to reflect changes
// that were possibly undetected.
func (mgr *Manager) Reload(ctx context.Context) error {
	mgr.mx.RLock()
	defer mgr.mx.RUnlock()
	if !mgr.started {
		return place.ErrStopped
	}
	var err error
	for _, p := range mgr.subplaces {
		if err1 := p.Reload(ctx); err1 != nil && err == nil {
			err = err1
		}
	}
	mgr.infos <- place.ChangeInfo{Reason: place.OnReload, Zid: id.Invalid}
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

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
	"io"
	"log"
	"net/url"
	"sort"
	"sync"
	"time"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/place/manager/memstore"
	"zettelstore.de/z/place/manager/store"
	"zettelstore.de/z/service"
)

// ConnectData contains all administration related values.
type ConnectData struct {
	Enricher place.Enricher
	Notify   chan<- change.Info
}

// Connect returns a handle to the specified place
func Connect(rawURL string, authManager auth.BaseManager, cdata *ConnectData) (place.ManagedPlace, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "dir"
	}
	if authManager.IsReadonly() {
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

type createFunc func(*url.URL, *ConnectData) (place.ManagedPlace, error)

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
	mgrMx        sync.RWMutex
	started      bool
	subplaces    []place.ManagedPlace
	observers    []change.Func
	mxObserver   sync.RWMutex
	done         chan struct{}
	infos        chan change.Info
	propertyKeys map[string]bool // Set of property key names

	// Indexer data
	idxStore store.Store
	idxAr    *anterooms
	idxReady chan struct{} // Signal a non-empty anteroom to background task

	// Indexer stats data
	idxMx           sync.RWMutex
	idxLastReload   time.Time
	idxSinceReload  uint64
	idxDurLastIndex time.Duration
}

// New creates a new managing place.
func New(placeURIs []string, cfg *meta.Meta, authManager auth.BaseManager) (*Manager, error) {
	propertyKeys := make(map[string]bool)
	for _, kd := range meta.GetSortedKeyDescriptions() {
		if kd.IsProperty() {
			propertyKeys[kd.Name] = true
		}
	}
	mgr := &Manager{
		infos:        make(chan change.Info, len(placeURIs)*10),
		propertyKeys: propertyKeys,

		idxStore: memstore.New(),
		idxAr:    newAnterooms(10),
		idxReady: make(chan struct{}, 1),
	}
	cdata := ConnectData{Enricher: mgr, Notify: mgr.infos}
	subplaces := make([]place.ManagedPlace, 0, len(placeURIs)+2)
	for _, uri := range placeURIs {
		p, err := Connect(uri, authManager, &cdata)
		if err != nil {
			return nil, err
		}
		if p != nil {
			subplaces = append(subplaces, p)
		}
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
func (mgr *Manager) RegisterObserver(f change.Func) {
	if f != nil {
		mgr.mxObserver.Lock()
		mgr.observers = append(mgr.observers, f)
		mgr.mxObserver.Unlock()
	}
}

func (mgr *Manager) notifyObserver(ci change.Info) {
	mgr.mxObserver.RLock()
	observers := mgr.observers
	mgr.mxObserver.RUnlock()
	for _, ob := range observers {
		ob(ci)
	}
}

func (mgr *Manager) notifier() {
	// The call to notify may panic. Ensure a running notifier.
	defer func() {
		if r := recover(); r != nil {
			service.Main.LogRecover("Notifier", r)
			go mgr.notifier()
		}
	}()

	for {
		select {
		case ci, ok := <-mgr.infos:
			if ok {
				mgr.idxEnqueue(ci)
				mgr.notifyObserver(ci)
			}
		case <-mgr.done:
			return
		}
	}
}

func (mgr *Manager) idxEnqueue(ci change.Info) {
	switch ci.Reason {
	case change.OnReload:
		mgr.idxAr.Reset()
	case change.OnUpdate:
		mgr.idxAr.Enqueue(ci.Zid, arUpdate)
	case change.OnDelete:
		mgr.idxAr.Enqueue(ci.Zid, arDelete)
	default:
		return
	}
	select {
	case mgr.idxReady <- struct{}{}:
	default:
	}
}

// Start the place. Now all other functions of the place are allowed.
// Starting an already started place is not allowed.
func (mgr *Manager) Start(ctx context.Context) error {
	mgr.mgrMx.Lock()
	if mgr.started {
		mgr.mgrMx.Unlock()
		return place.ErrStarted
	}
	for i := len(mgr.subplaces) - 1; i >= 0; i-- {
		ssi, ok := mgr.subplaces[i].(place.StartStopper)
		if !ok {
			continue
		}
		err := ssi.Start(ctx)
		if err == nil {
			continue
		}
		for j := i + 1; j < len(mgr.subplaces); j++ {
			if ssj, ok := mgr.subplaces[j].(place.StartStopper); ok {
				ssj.Stop(ctx)
			}
		}
		mgr.mgrMx.Unlock()
		return err
	}
	mgr.idxAr.Reset() // Ensure an initial index run
	mgr.done = make(chan struct{})
	go mgr.notifier()
	go mgr.idxIndexer()

	// mgr.startIndexer(mgr)
	mgr.started = true
	mgr.mgrMx.Unlock()
	mgr.infos <- change.Info{Reason: change.OnReload, Zid: id.Invalid}
	return nil
}

// Stop the started place. Now only the Start() function is allowed.
func (mgr *Manager) Stop(ctx context.Context) error {
	mgr.mgrMx.Lock()
	defer mgr.mgrMx.Unlock()
	if !mgr.started {
		return place.ErrStopped
	}
	close(mgr.done)
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

// ReadStats populates st with place statistics
func (mgr *Manager) ReadStats(st *place.Stats) {
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	subStats := make([]place.ManagedPlaceStats, len(mgr.subplaces))
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
	st.NumManagedPlaces = len(mgr.subplaces)
	st.ZettelTotal = sumZettel

	var storeSt store.Stats
	mgr.idxMx.RLock()
	defer mgr.idxMx.RUnlock()
	mgr.idxStore.ReadStats(&storeSt)

	st.LastReload = mgr.idxLastReload
	st.IndexesSinceReload = mgr.idxSinceReload
	st.DurLastIndex = mgr.idxDurLastIndex
	st.ZettelIndexed = storeSt.Zettel
	st.IndexUpdates = storeSt.Updates
	st.IndexedWords = storeSt.Words
	st.IndexedUrls = storeSt.Urls
}

// Dump internal data structures to a Writer.
func (mgr *Manager) Dump(w io.Writer) {
	mgr.idxStore.Dump(w)
}

//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various boxes and indexes of a Zettelstore.
package manager

import (
	"context"
	"io"
	"net/url"
	"sort"
	"sync"
	"time"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager/memstore"
	"zettelstore.de/z/box/manager/store"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/strfun"
)

// ConnectData contains all administration related values.
type ConnectData struct {
	Number   int // number of the box, starting with 1.
	Config   config.Config
	Enricher box.Enricher
	Notify   chan<- box.UpdateInfo
}

// Connect returns a handle to the specified box.
func Connect(u *url.URL, authManager auth.BaseManager, cdata *ConnectData) (box.ManagedBox, error) {
	if authManager.IsReadonly() {
		rawURL := u.String()
		// TODO: the following is wrong under some circumstances:
		// 1. fragment is set
		if q := u.Query(); len(q) == 0 {
			rawURL += "?readonly"
		} else if _, ok := q["readonly"]; !ok {
			rawURL += "&readonly"
		}
		var err error
		if u, err = url.Parse(rawURL); err != nil {
			return nil, err
		}
	}

	if create, ok := registry[u.Scheme]; ok {
		return create(u, cdata)
	}
	return nil, &ErrInvalidScheme{u.Scheme}
}

// ErrInvalidScheme is returned if there is no box with the given scheme.
type ErrInvalidScheme struct{ Scheme string }

func (err *ErrInvalidScheme) Error() string { return "Invalid scheme: " + err.Scheme }

type createFunc func(*url.URL, *ConnectData) (box.ManagedBox, error)

var registry = map[string]createFunc{}

// Register the encoder for later retrieval.
func Register(scheme string, create createFunc) {
	if _, ok := registry[scheme]; ok {
		panic(scheme)
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

// Manager is a coordinating box.
type Manager struct {
	mgrLog       *logger.Logger
	mgrMx        sync.RWMutex
	started      bool
	rtConfig     config.Config
	boxes        []box.ManagedBox
	observers    []box.UpdateFunc
	mxObserver   sync.RWMutex
	done         chan struct{}
	infos        chan box.UpdateInfo
	propertyKeys strfun.Set // Set of property key names

	// Indexer data
	idxLog   *logger.Logger
	idxStore store.Store
	idxAr    *anterooms
	idxReady chan struct{} // Signal a non-empty anteroom to background task

	// Indexer stats data
	idxMx          sync.RWMutex
	idxLastReload  time.Time
	idxDurReload   time.Duration
	idxSinceReload uint64
}

// New creates a new managing box.
func New(boxURIs []*url.URL, authManager auth.BaseManager, rtConfig config.Config) (*Manager, error) {
	descrs := meta.GetSortedKeyDescriptions()
	propertyKeys := make(strfun.Set, len(descrs))
	for _, kd := range descrs {
		if kd.IsProperty() {
			propertyKeys.Set(kd.Name)
		}
	}
	boxLog := kernel.Main.GetLogger(kernel.BoxService)
	mgr := &Manager{
		mgrLog:       boxLog.Clone().Str("box", "manager").Child(),
		rtConfig:     rtConfig,
		infos:        make(chan box.UpdateInfo, len(boxURIs)*10),
		propertyKeys: propertyKeys,

		idxLog:   boxLog.Clone().Str("box", "index").Child(),
		idxStore: memstore.New(),
		idxAr:    newAnterooms(10),
		idxReady: make(chan struct{}, 1),
	}
	cdata := ConnectData{Number: 1, Config: rtConfig, Enricher: mgr, Notify: mgr.infos}
	boxes := make([]box.ManagedBox, 0, len(boxURIs)+2)
	for _, uri := range boxURIs {
		p, err := Connect(uri, authManager, &cdata)
		if err != nil {
			return nil, err
		}
		if p != nil {
			boxes = append(boxes, p)
			cdata.Number++
		}
	}
	constbox, err := registry[" const"](nil, &cdata)
	if err != nil {
		return nil, err
	}
	cdata.Number++
	compbox, err := registry[" comp"](nil, &cdata)
	if err != nil {
		return nil, err
	}
	cdata.Number++
	boxes = append(boxes, constbox, compbox)
	mgr.boxes = boxes
	return mgr, nil
}

// RegisterObserver registers an observer that will be notified
// if a zettel was found to be changed.
func (mgr *Manager) RegisterObserver(f box.UpdateFunc) {
	if f != nil {
		mgr.mxObserver.Lock()
		mgr.observers = append(mgr.observers, f)
		mgr.mxObserver.Unlock()
	}
}

func (mgr *Manager) notifier() {
	// The call to notify may panic. Ensure a running notifier.
	defer func() {
		if r := recover(); r != nil {
			kernel.Main.LogRecover("Notifier", r)
			go mgr.notifier()
		}
	}()

	tsLastEvent := time.Now()
	cache := destutterCache{}
	for {
		select {
		case ci, ok := <-mgr.infos:
			if ok {
				now := time.Now()
				if len(cache) > 1 && tsLastEvent.Add(10*time.Second).Before(now) {
					// Cache contains entries and is definitely outdated
					mgr.mgrLog.Trace().Msg("clean destutter cache")
					cache = destutterCache{}
				}
				tsLastEvent = now

				reason, zid := ci.Reason, ci.Zid
				mgr.mgrLog.Debug().Uint("reason", uint64(reason)).Zid(zid).Msg("notifier")
				if ignoreUpdate(cache, now, reason, zid) {
					mgr.mgrLog.Trace().Uint("reason", uint64(reason)).Zid(zid).Msg("notifier ignored")
					continue
				}
				mgr.idxEnqueue(reason, zid)
				if ci.Box == nil {
					ci.Box = mgr
				}
				mgr.notifyObserver(&ci)
			}
		case <-mgr.done:
			return
		}
	}
}

type destutterData struct {
	deadAt time.Time
	reason box.UpdateReason
}
type destutterCache = map[id.Zid]destutterData

func ignoreUpdate(cache destutterCache, now time.Time, reason box.UpdateReason, zid id.Zid) bool {
	if dsd, found := cache[zid]; found {
		if dsd.reason == reason && dsd.deadAt.After(now) {
			return true
		}
	}
	cache[zid] = destutterData{
		deadAt: now.Add(500 * time.Millisecond),
		reason: reason,
	}
	return false
}

func (mgr *Manager) idxEnqueue(reason box.UpdateReason, zid id.Zid) {
	switch reason {
	case box.OnReload:
		mgr.idxAr.Reset()
	case box.OnUpdate:
		mgr.idxAr.Enqueue(zid, arUpdate)
	case box.OnDelete:
		mgr.idxAr.Enqueue(zid, arDelete)
	default:
		return
	}
	select {
	case mgr.idxReady <- struct{}{}:
	default:
	}
}

func (mgr *Manager) notifyObserver(ci *box.UpdateInfo) {
	mgr.mxObserver.RLock()
	observers := mgr.observers
	mgr.mxObserver.RUnlock()
	for _, ob := range observers {
		ob(*ci)
	}
}

// Start the box. Now all other functions of the box are allowed.
// Starting an already started box is not allowed.
func (mgr *Manager) Start(ctx context.Context) error {
	mgr.mgrMx.Lock()
	if mgr.started {
		mgr.mgrMx.Unlock()
		return box.ErrStarted
	}
	for i := len(mgr.boxes) - 1; i >= 0; i-- {
		ssi, ok := mgr.boxes[i].(box.StartStopper)
		if !ok {
			continue
		}
		err := ssi.Start(ctx)
		if err == nil {
			continue
		}
		for j := i + 1; j < len(mgr.boxes); j++ {
			if ssj, ok2 := mgr.boxes[j].(box.StartStopper); ok2 {
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

	mgr.started = true
	mgr.mgrMx.Unlock()
	return nil
}

// Stop the started box. Now only the Start() function is allowed.
func (mgr *Manager) Stop(ctx context.Context) {
	mgr.mgrMx.Lock()
	defer mgr.mgrMx.Unlock()
	if !mgr.started {
		return
	}
	close(mgr.done)
	for _, p := range mgr.boxes {
		if ss, ok := p.(box.StartStopper); ok {
			ss.Stop(ctx)
		}
	}
	mgr.started = false
}

// Refresh internal box data.
func (mgr *Manager) Refresh(ctx context.Context) error {
	mgr.mgrLog.Debug().Msg("Refresh")
	mgr.mgrMx.Lock()
	defer mgr.mgrMx.Unlock()
	if !mgr.started {
		return box.ErrStopped
	}
	mgr.infos <- box.UpdateInfo{Reason: box.OnReload, Zid: id.Invalid}
	for _, bx := range mgr.boxes {
		if rb, ok := bx.(box.Refresher); ok {
			rb.Refresh(ctx)
		}
	}
	return nil
}

// ReadStats populates st with box statistics.
func (mgr *Manager) ReadStats(st *box.Stats) {
	mgr.mgrLog.Debug().Msg("ReadStats")
	mgr.mgrMx.RLock()
	defer mgr.mgrMx.RUnlock()
	subStats := make([]box.ManagedBoxStats, len(mgr.boxes))
	for i, p := range mgr.boxes {
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
	st.NumManagedBoxes = len(mgr.boxes)
	st.ZettelTotal = sumZettel

	var storeSt store.Stats
	mgr.idxMx.RLock()
	defer mgr.idxMx.RUnlock()
	mgr.idxStore.ReadStats(&storeSt)

	st.LastReload = mgr.idxLastReload
	st.IndexesSinceReload = mgr.idxSinceReload
	st.DurLastReload = mgr.idxDurReload
	st.ZettelIndexed = storeSt.Zettel
	st.IndexUpdates = storeSt.Updates
	st.IndexedWords = storeSt.Words
	st.IndexedUrls = storeSt.Urls
}

// Dump internal data structures to a Writer.
func (mgr *Manager) Dump(w io.Writer) {
	mgr.idxStore.Dump(w)
}

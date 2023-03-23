//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
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
	"sync"
	"time"

	"zettelstore.de/c/maps"
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
func GetSchemes() []string { return maps.Keys(registry) }

type managerState uint8

const (
	mgrStateStopped managerState = iota
	mgrStateStarting
	mgrStateStarted
	mgrStateStopping
)

// Manager is a coordinating box.
type Manager struct {
	mgrLog       *logger.Logger
	stateMx      sync.RWMutex
	state        managerState
	boxReady     []bool
	mgrMx        sync.RWMutex
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

func (mgr *Manager) setState(newState managerState) {
	mgr.stateMx.Lock()
	mgr.state = newState
	mgr.stateMx.Unlock()
}

func (mgr *Manager) getState() managerState {
	mgr.stateMx.RLock()
	state := mgr.state
	mgr.stateMx.RUnlock()
	return state
}

func (mgr *Manager) clearBoxReady() {
	mgr.stateMx.Lock()
	for i := 0; i < len(mgr.boxReady); i++ {
		mgr.boxReady[i] = false
	}
	mgr.stateMx.Unlock()
}

func (mgr *Manager) setBoxReady(i int, b bool) {
	mgr.stateMx.Lock()
	mgr.boxReady[i] = b
	mgr.stateMx.Unlock()
}

func (mgr *Manager) allBoxReady() bool {
	mgr.stateMx.RLock()
	defer mgr.stateMx.RUnlock()
	for _, b := range mgr.boxReady {
		if !b {
			return false
		}
	}
	return true
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
		idxAr:    newAnterooms(1000),
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
	mgr.boxReady = make([]bool, len(boxes))
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

	doNotify := true
	queueNotify := make([]box.UpdateInfo, 0, 16384)
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

				doNotify, queueNotify = mgr.onStartingManager(&ci, queueNotify)

				mgr.idxEnqueue(reason, zid)
				if ci.Box == nil {
					ci.Box = mgr
				}
				if doNotify {
					mgr.notifyObserver(&ci)
				} else if reason != box.OnReady {
					mgr.mgrLog.Trace().Uint("reason", uint64(ci.Reason)).Zid(ci.Zid).Msg("queue info")
					queueNotify = append(queueNotify, ci)
				}
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

func (mgr *Manager) onStartingManager(ci *box.UpdateInfo, queueNotify []box.UpdateInfo) (bool, []box.UpdateInfo) {
	result := mgr.getState() == mgrStateStarting
	if ci.Reason != box.OnReady {
		return result, queueNotify
	}
	if i := mgr.boxIndex(ci.Box); i >= 0 {
		mgr.setBoxReady(i, true)
		if mgr.allBoxReady() {
			mgr.setState(mgrStateStarted)
			mgr.mgrLog.Debug().Msg("Manager started")
			mgr.notifyObserver(&box.UpdateInfo{Box: mgr, Reason: box.OnReady})
			for _, ci2 := range queueNotify {
				mgr.mgrLog.Trace().Uint("reason", uint64(ci2.Reason)).Zid(ci2.Zid).Msg("queued notifier")
				mgr.notifyObserver(&ci2)
			}
			return false, nil
		}
	}

	return false, queueNotify
}

func (mgr *Manager) boxIndex(bx box.BaseBox) int {
	boxes := mgr.boxes
	for i, b := range boxes {
		if mbox, ok := bx.(box.ManagedBox); ok && b == mbox {
			return i
		}
	}
	return -1
}

func (mgr *Manager) idxEnqueue(reason box.UpdateReason, zid id.Zid) {
	switch reason {
	case box.OnReady:
		return
	case box.OnReload:
		mgr.idxAr.Reset()
	case box.OnZettel:
		mgr.idxAr.EnqueueZettel(zid)
	default:
		mgr.mgrLog.Warn().Uint("reason", uint64(reason)).Zid(zid).Msg("Unknown notification reason")
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
	defer mgr.mgrMx.Unlock()
	if mgr.getState() != mgrStateStopped {
		return box.ErrStarted
	}
	mgr.setState(mgrStateStarting)
	mgr.clearBoxReady()
	noStartStopper := 0
	for i := len(mgr.boxes) - 1; i >= 0; i-- {
		ssi, ok := mgr.boxes[i].(box.StartStopper)
		if !ok {
			mgr.setBoxReady(i, true)
			noStartStopper++
			continue
		}
		err := ssi.Start(ctx)
		if err == nil {
			continue
		}
		mgr.setState(mgrStateStopping)
		for j := i + 1; j < len(mgr.boxes); j++ {
			if ssj, ok2 := mgr.boxes[j].(box.StartStopper); ok2 {
				ssj.Stop(ctx)
			}
		}
		mgr.clearBoxReady()
		mgr.setState(mgrStateStopped)
		return err
	}
	mgr.idxAr.Reset() // Ensure an initial index run
	mgr.done = make(chan struct{})
	go mgr.notifier()

	if noStartStopper == len(mgr.boxes) && mgr.allBoxReady() {
		mgr.setState(mgrStateStarted)
	}
	go mgr.idxIndexer()
	return nil
}

// Stop the started box. Now only the Start() function is allowed.
func (mgr *Manager) Stop(ctx context.Context) {
	mgr.mgrMx.Lock()
	defer mgr.mgrMx.Unlock()
	if mgr.getState() != mgrStateStarted {
		return
	}
	mgr.setState(mgrStateStopping)
	close(mgr.done)
	for _, p := range mgr.boxes {
		if ss, ok := p.(box.StartStopper); ok {
			ss.Stop(ctx)
		}
	}
	mgr.clearBoxReady()
	mgr.setState(mgrStateStopped)
}

// Refresh internal box data.
func (mgr *Manager) Refresh(ctx context.Context) error {
	mgr.mgrLog.Debug().Msg("Refresh")
	if mgr.getState() != mgrStateStarted {
		return box.ErrStopped
	}
	mgr.mgrMx.Lock()
	defer mgr.mgrMx.Unlock()
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

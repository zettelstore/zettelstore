//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the main internal service implementation.
package impl

import (
	"context"
	"fmt"
	"sync"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/place"
	"zettelstore.de/z/service"
)

type placeSub struct {
	subConfig
	mxService     sync.RWMutex
	manager       place.Manager
	createManager service.CreatePlaceManagerFunc
}

func (ps *placeSub) Initialize() {
	ps.descr = descriptionMap{
		service.PlaceDefaultDirType: {
			"Default directory place type",
			ps.noFrozen(func(val string) interface{} {
				switch val {
				case service.PlaceDirTypeNotify, service.PlaceDirTypeSimple:
					return val
				}
				return nil
			}),
			true,
		},
	}
	ps.next = interfaceMap{
		service.PlaceDefaultDirType: service.PlaceDirTypeNotify,
	}
}

func (ps *placeSub) Start(srv *myService) error {
	ps.mxService.Lock()
	defer ps.mxService.Unlock()
	mgr, err := ps.createManager()
	if err != nil {
		srv.doLog("Unable to create place manager:", err)
		return err
	}
	srv.doLog("Start Place Manager:", mgr.Location())
	if err := mgr.Start(context.Background()); err != nil {
		srv.doLog("Unable to start place manager:", err)
	}
	runtime.SetupConfiguration(mgr)
	ps.manager = mgr
	return nil
}

func (ps *placeSub) IsStarted() bool {
	ps.mxService.RLock()
	defer ps.mxService.RUnlock()
	return ps.manager != nil
}

func (ps *placeSub) Stop(srv *myService) error {
	srv.doLog("Stop Place Manager")
	ps.mxService.RLock()
	mgr := ps.manager
	ps.mxService.RUnlock()
	err := mgr.Stop(context.Background())
	ps.mxService.Lock()
	ps.manager = nil
	ps.mxService.Unlock()
	return err
}

func (ps *placeSub) GetStatistics() []service.KeyValue {
	var st place.Stats
	ps.mxService.RLock()
	ps.manager.ReadStats(&st)
	numPlaces := ps.manager.NumPlaces()
	ps.mxService.RUnlock()
	return []service.KeyValue{
		{Key: "Read-only", Value: fmt.Sprintf("%v", st.ReadOnly)},
		{Key: "Sub-places", Value: fmt.Sprintf("%v", numPlaces)},
		{Key: "Zettel (total)", Value: fmt.Sprintf("%v", st.ZettelTotal)},
		{Key: "Zettel (indexable)", Value: fmt.Sprintf("%v", st.ZettelIndexed)},
		{Key: "Last re-index", Value: st.LastReload.Format("2006-01-02 15:04:05 -0700 MST")},
		{Key: "Indexes since last re-index", Value: fmt.Sprintf("%v", st.IndexesSinceReload)},
		{Key: "Duration last index", Value: fmt.Sprintf("%vms", st.DurLastIndex.Milliseconds())},
		{Key: "Indexed words", Value: fmt.Sprintf("%v", st.IndexedWords)},
		{Key: "Indexed URLs", Value: fmt.Sprintf("%v", st.IndexedUrls)},
		{Key: "Zettel enrichments", Value: fmt.Sprintf("%v", st.IndexUpdates)},
	}
}

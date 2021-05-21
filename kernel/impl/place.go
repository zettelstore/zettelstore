//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the kernel implementation.
package impl

import (
	"context"
	"fmt"
	"sync"

	"zettelstore.de/z/kernel"
	"zettelstore.de/z/place"
)

type placeService struct {
	srvConfig
	mxService     sync.RWMutex
	manager       place.Manager
	createManager kernel.CreatePlaceManagerFunc
}

func (ps *placeService) Initialize() {
	ps.descr = descriptionMap{
		kernel.PlaceDefaultDirType: {
			"Default directory place type",
			ps.noFrozen(func(val string) interface{} {
				switch val {
				case kernel.PlaceDirTypeNotify, kernel.PlaceDirTypeSimple:
					return val
				}
				return nil
			}),
			true,
		},
	}
	ps.next = interfaceMap{
		kernel.PlaceDefaultDirType: kernel.PlaceDirTypeNotify,
	}
}

func (ps *placeService) Start(kern *myKernel) error {
	ps.mxService.Lock()
	defer ps.mxService.Unlock()
	mgr, err := ps.createManager(kern.auth.manager, kern.cfg.rtConfig)
	if err != nil {
		kern.doLog("Unable to create place manager:", err)
		return err
	}
	kern.doLog("Start Place Manager:", mgr.Location())
	if err := mgr.Start(context.Background()); err != nil {
		kern.doLog("Unable to start place manager:", err)
	}
	kern.cfg.setPlace(mgr)
	ps.manager = mgr
	return nil
}

func (ps *placeService) IsStarted() bool {
	ps.mxService.RLock()
	defer ps.mxService.RUnlock()
	return ps.manager != nil
}

func (ps *placeService) Stop(kern *myKernel) error {
	kern.doLog("Stop Place Manager")
	ps.mxService.RLock()
	mgr := ps.manager
	ps.mxService.RUnlock()
	err := mgr.Stop(context.Background())
	ps.mxService.Lock()
	ps.manager = nil
	ps.mxService.Unlock()
	return err
}

func (ps *placeService) GetStatistics() []kernel.KeyValue {
	var st place.Stats
	ps.mxService.RLock()
	ps.manager.ReadStats(&st)
	ps.mxService.RUnlock()
	return []kernel.KeyValue{
		{Key: "Read-only", Value: fmt.Sprintf("%v", st.ReadOnly)},
		{Key: "Sub-places", Value: fmt.Sprintf("%v", st.NumManagedPlaces)},
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

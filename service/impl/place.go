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

	"zettelstore.de/z/place"
	"zettelstore.de/z/service"
)

type placeSub struct {
	subConfig
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
	ps.mx.Lock()
	defer ps.mx.Unlock()
	ps.manager = ps.createManager()
	return nil
}

func (ps *placeSub) IsStarted() bool {
	ps.mx.RLock()
	defer ps.mx.RUnlock()
	return ps.manager != nil
}

func (ps *placeSub) Stop(srv *myService) error {
	ps.mx.RLock()
	mgr := ps.manager
	ps.mx.RUnlock()
	err := mgr.Stop(context.Background())
	ps.mx.Lock()
	ps.manager = nil
	ps.mx.Unlock()
	return err
}

func (ps *placeSub) GetStatistics() []service.KeyValue {
	var st place.Stats
	ps.mx.RLock()
	ps.manager.ReadStats(&st)
	numPlaces := ps.manager.NumPlaces()
	ps.mx.RUnlock()
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

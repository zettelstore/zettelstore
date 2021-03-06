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
	"io"
	"net/url"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/kernel"
)

type boxService struct {
	srvConfig
	mxService     sync.RWMutex
	manager       box.Manager
	createManager kernel.CreateBoxManagerFunc
}

func (ps *boxService) Initialize() {
	ps.descr = descriptionMap{
		kernel.BoxDefaultDirType: {
			"Default directory box type",
			ps.noFrozen(func(val string) interface{} {
				switch val {
				case kernel.BoxDirTypeNotify, kernel.BoxDirTypeSimple:
					return val
				}
				return nil
			}),
			true,
		},
		kernel.BoxURIs: {
			"Box URI",
			func(val string) interface{} {
				uVal, err := url.Parse(val)
				if err != nil {
					return nil
				}
				if uVal.Scheme == "" {
					uVal.Scheme = "dir"
				}
				return uVal
			},
			true,
		},
	}
	ps.next = interfaceMap{
		kernel.BoxDefaultDirType: kernel.BoxDirTypeNotify,
	}
}

func (ps *boxService) Start(kern *myKernel) error {
	boxURIs := make([]*url.URL, 0, 4)
	format := kernel.BoxURIs + "%d"
	for i := 1; ; i++ {
		u := ps.GetNextConfig(fmt.Sprintf(format, i))
		if u == nil {
			break
		}
		boxURIs = append(boxURIs, u.(*url.URL))
	}
	ps.mxService.Lock()
	defer ps.mxService.Unlock()
	mgr, err := ps.createManager(boxURIs, kern.auth.manager, kern.cfg.rtConfig)
	if err != nil {
		kern.doLog("Unable to create box manager:", err)
		return err
	}
	kern.doLog("Start Box Manager:", mgr.Location())
	if err := mgr.Start(context.Background()); err != nil {
		kern.doLog("Unable to start box manager:", err)
	}
	kern.cfg.setBox(mgr)
	ps.manager = mgr
	return nil
}

func (ps *boxService) IsStarted() bool {
	ps.mxService.RLock()
	defer ps.mxService.RUnlock()
	return ps.manager != nil
}

func (ps *boxService) Stop(kern *myKernel) error {
	kern.doLog("Stop Box Manager")
	ps.mxService.RLock()
	mgr := ps.manager
	ps.mxService.RUnlock()
	err := mgr.Stop(context.Background())
	ps.mxService.Lock()
	ps.manager = nil
	ps.mxService.Unlock()
	return err
}

func (ps *boxService) GetStatistics() []kernel.KeyValue {
	var st box.Stats
	ps.mxService.RLock()
	ps.manager.ReadStats(&st)
	ps.mxService.RUnlock()
	return []kernel.KeyValue{
		{Key: "Read-only", Value: fmt.Sprintf("%v", st.ReadOnly)},
		{Key: "Managed boxes", Value: fmt.Sprintf("%v", st.NumManagedBoxes)},
		{Key: "Zettel (total)", Value: fmt.Sprintf("%v", st.ZettelTotal)},
		{Key: "Zettel (indexed)", Value: fmt.Sprintf("%v", st.ZettelIndexed)},
		{Key: "Last re-index", Value: st.LastReload.Format("2006-01-02 15:04:05 -0700 MST")},
		{Key: "Duration last re-index", Value: fmt.Sprintf("%vms", st.DurLastReload.Milliseconds())},
		{Key: "Indexes since last re-index", Value: fmt.Sprintf("%v", st.IndexesSinceReload)},
		{Key: "Indexed words", Value: fmt.Sprintf("%v", st.IndexedWords)},
		{Key: "Indexed URLs", Value: fmt.Sprintf("%v", st.IndexedUrls)},
		{Key: "Zettel enrichments", Value: fmt.Sprintf("%v", st.IndexUpdates)},
	}
}

func (ps *boxService) DumpIndex(w io.Writer) {
	ps.manager.Dump(w)
}

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
	"sync"

	"zettelstore.de/z/service"
)

type authSub struct {
	subConfig
	mxService sync.RWMutex
	started   bool
}

func (as *authSub) Initialize() {
	as.descr = descriptionMap{
		service.AuthReadonly: {"Read-only mode", parseBool, true},
	}
	as.next = interfaceMap{
		service.AuthReadonly: false,
	}
}

func (as *authSub) Start(srv *myService) error {
	as.mxService.Lock()
	defer as.mxService.Unlock()
	as.started = true
	return nil
}

func (as *authSub) IsStarted() bool {
	as.mxService.RLock()
	defer as.mxService.RUnlock()
	return as.started
}

func (as *authSub) Stop(srv *myService) error {
	as.mxService.Lock()
	defer as.mxService.Unlock()
	as.started = false
	return nil
}

func (as *authSub) GetStatistics() []service.KeyValue {
	return nil
}

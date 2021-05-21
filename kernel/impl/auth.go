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
	"sync"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/kernel"
)

type authService struct {
	srvConfig
	mxService     sync.RWMutex
	manager       auth.Manager
	createManager kernel.CreateAuthManagerFunc
}

func (as *authService) Initialize() {
	as.descr = descriptionMap{
		kernel.AuthOwner: {
			"Owner's zettel id",
			func(val string) interface{} {
				if owner := as.cur[kernel.AuthOwner]; owner != nil && owner != id.Invalid {
					return nil
				}
				return parseZid(val)
			},
			false,
		},
		kernel.AuthReadonly: {
			"Read-only mode",
			func(val string) interface{} {
				if ro := as.cur[kernel.AuthReadonly]; ro == true {
					return nil
				}
				return parseBool(val)
			},
			true,
		},
	}
	as.next = interfaceMap{
		kernel.AuthOwner:    id.Invalid,
		kernel.AuthReadonly: false,
	}
}

func (as *authService) Start(kern *myKernel) error {
	as.mxService.Lock()
	defer as.mxService.Unlock()
	readonlyMode := as.GetNextConfig(kernel.AuthReadonly).(bool)
	owner := as.GetNextConfig(kernel.AuthOwner).(id.Zid)
	authMgr, err := as.createManager(readonlyMode, owner)
	if err != nil {
		kern.doLog("Unable to create auth manager:", err)
		return err
	}
	kern.doLog("Start Auth Manager")
	as.manager = authMgr
	return nil
}

func (as *authService) IsStarted() bool {
	as.mxService.RLock()
	defer as.mxService.RUnlock()
	return as.manager != nil
}

func (as *authService) Stop(kern *myKernel) error {
	kern.doLog("Stop Auth Manager")
	as.mxService.Lock()
	defer as.mxService.Unlock()
	as.manager = nil
	return nil
}

func (as *authService) GetStatistics() []kernel.KeyValue {
	return nil
}

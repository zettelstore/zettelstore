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
	"zettelstore.de/z/service"
)

type authSub struct {
	subConfig
}

func (as *authSub) Initialize() {
	as.descr = descriptionMap{
		service.AuthReadonly: {"Read-only mode", parseBool, true},
		service.AuthSimple:   {"Simple user mode", as.noFrozen(parseBool), true},
	}
	as.next = interfaceMap{
		service.AuthReadonly: false,
		service.AuthSimple:   false,
	}
}

func (as *authSub) Start(srv *myService) error { return nil }
func (as *authSub) Stop(srv *myService) error  { return nil }

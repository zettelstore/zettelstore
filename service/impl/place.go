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

type placeSub struct {
	subConfig
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

func (ps *placeSub) Start(srv *myService) error { return nil }
func (ps *placeSub) Stop(srv *myService) error  { return nil }

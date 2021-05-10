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

type indexSub struct {
	subConfig
}

func (is *indexSub) Initialize() {
	is.descr = descriptionMap{}
	is.next = interfaceMap{}
}

func (is *indexSub) Start(srv *myService) error { return nil }
func (is *indexSub) Stop(srv *myService) error  { return nil }

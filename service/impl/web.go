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
	"errors"

	"zettelstore.de/z/service"
	"zettelstore.de/z/web/server"
)

type webService struct {
	srvw          *server.Server
	createHandler service.CreateHandlerFunc
}

func (srv *myService) WebSetConfig(createHandler service.CreateHandlerFunc) {
	srv.web.createHandler = createHandler
}

var (
	errAlreadyStarted = errors.New("already started")
	errConfigMissing  = errors.New("no configuration")
	errAlreadyStopped = errors.New("already stopped")
)

func (srv *myService) WebStart() error {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	if srv.web.srvw != nil {
		return errAlreadyStarted
	}
	createHandler := srv.web.createHandler
	if createHandler == nil {
		return errConfigMissing
	}
	srvw := server.New(srv.config.GetConfig(service.SubWeb, service.WebListenAddress), createHandler())
	if srv.debug {
		srvw.SetDebug()
	}
	srvw.Run()
	srv.switchNextToCur(service.SubWeb)
	srv.web.srvw = srvw
	return nil
}

func (srv *myService) WebStop() error {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	srvw := srv.web.srvw
	if srvw == nil {
		return errAlreadyStopped
	}
	return srvw.Stop()
}

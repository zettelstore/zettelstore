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
	srv.mx.RLock()
	if srv.web.srvw != nil {
		srv.mx.RUnlock()
		return errAlreadyStarted
	}
	createHandler := srv.web.createHandler
	if createHandler == nil {
		srv.mx.RUnlock()
		return errConfigMissing
	}
	listenAddr := srv.config.GetConfig(service.SubWeb, service.WebListenAddress).(string)
	urlPrefix := srv.config.GetConfig(service.SubWeb, service.WebURLPrefix).(string)
	readonlyMode := srv.config.GetConfig(service.SubMain, service.MainReadonly).(bool)
	srv.mx.RUnlock()
	handler := createHandler(urlPrefix, readonlyMode)
	srvw := server.New(listenAddr, handler)
	if srv.debug {
		srvw.SetDebug()
	}
	srv.doLog("Start Zettelstore Web Service")
	srv.doLog("Listening on", listenAddr)
	srvw.Run()
	srv.mx.Lock()
	defer srv.mx.Unlock()
	srv.config.switchNextToCur(service.SubWeb)
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

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
	"net/http"

	"zettelstore.de/z/web/server"
)

type webService struct {
	srvw       *server.Server
	curConfig  *webServiceConfig
	nextConfig *webServiceConfig
}

type webServiceConfig struct {
	addrListen string
	handler    http.Handler
}

func (srv *myService) WebSetConfig(addrListen string, handler http.Handler) {
	srv.web.nextConfig = &webServiceConfig{addrListen, handler}
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
	config := srv.web.nextConfig
	if config == nil {
		return errConfigMissing
	}
	srvw := server.New(config.addrListen, config.handler)
	if srv.debug {
		srvw.SetDebug()
	}
	srvw.Run()
	srv.web.curConfig = config
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

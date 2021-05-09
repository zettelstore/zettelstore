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

type webSub struct {
	subConfig
	srvw          *server.Server
	createHandler service.CreateHandlerFunc
}

func (ws *webSub) Initialize() {
	ws.descr = descriptionMap{
		service.WebListenAddress: {"Listen address, format [IP_ADDRESS]:PORT", parseString, true},
		service.WebURLPrefix: {
			"URL prefix under which the web server runs",
			func(val string) interface{} {
				if val != "" && val[0] == '/' && val[len(val)-1] == '/' {
					return val
				}
				return nil
			},
			true,
		},
	}
	ws.next = interfaceMap{
		service.WebListenAddress: "127.0.0.1:23123",
		service.WebURLPrefix:     "/",
	}
}

func (ws *webSub) Start(srv *myService) error {
	if srv.web.srvw != nil {
		return errAlreadyStarted
	}
	createHandler := srv.web.createHandler
	if createHandler == nil {
		return errConfigMissing
	}
	listenAddr := ws.GetConfig(service.WebListenAddress).(string)
	urlPrefix := ws.GetConfig(service.WebURLPrefix).(string)
	simple := srv.auth.GetConfig(service.AuthSimple).(bool)
	readonlyMode := srv.auth.GetConfig(service.AuthReadonly).(bool)
	handler := createHandler(urlPrefix, simple, readonlyMode)
	srvw := server.New(listenAddr, handler)
	if srv.debug {
		srvw.SetDebug()
	}
	if err := srvw.Run(); err != nil {
		srv.doLog("Unable to start Zettelstore Web Service:", err)
		return err
	}
	srv.doLog("Start Zettelstore Web Service")
	srv.doLog("Listening on", listenAddr)
	ws.srvw = srvw
	return nil
}

func (ws *webSub) Stop(srv *myService) error {
	srvw := ws.srvw
	if srvw == nil {
		return errAlreadyStopped
	}
	srv.doLog("Stopping Zettelstore Web Service ...")
	return srvw.Stop()
}

func (srv *myService) WebSetConfig(createHandler service.CreateHandlerFunc) {
	srv.web.createHandler = createHandler
}

var (
	errAlreadyStarted = errors.New("already started")
	errConfigMissing  = errors.New("no configuration")
	errAlreadyStopped = errors.New("already stopped")
)

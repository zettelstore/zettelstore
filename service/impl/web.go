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
	"net"
	"sync"

	"zettelstore.de/z/service"
	"zettelstore.de/z/web/server"
)

type webSub struct {
	subConfig
	mxService     sync.RWMutex
	srvw          *server.Server
	createHandler service.CreateWebHandlerFunc
}

func (ws *webSub) Initialize() {
	ws.descr = descriptionMap{
		service.WebListenAddress: {
			"Listen address",
			func(val string) interface{} {
				host, port, err := net.SplitHostPort(val)
				if err != nil {
					return nil
				}
				if _, err := net.LookupPort("tcp", port); err != nil {
					return nil
				}
				return net.JoinHostPort(host, port)
			},
			true},
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
	listenAddr := ws.GetNextConfig(service.WebListenAddress).(string)
	urlPrefix := ws.GetNextConfig(service.WebURLPrefix).(string)

	readonlyMode := srv.auth.GetConfig(service.AuthReadonly).(bool)

	handler, err := srv.web.createHandler(urlPrefix, srv.place.manager, readonlyMode)
	if err != nil {
		srv.doLog("Unable to create handler for Web Service:", err)
		return err
	}
	srvw := server.New(listenAddr, handler)
	if srv.debug {
		srvw.SetDebug()
	}
	if err := srvw.Run(); err != nil {
		srv.doLog("Unable to start Web Service:", err)
		return err
	}
	srv.doLog("Start Web Service:", listenAddr)
	ws.mxService.Lock()
	ws.srvw = srvw
	ws.mxService.Unlock()
	return nil
}

func (ws *webSub) IsStarted() bool {
	ws.mxService.RLock()
	defer ws.mxService.RUnlock()
	return ws.srvw != nil
}

func (ws *webSub) Stop(srv *myService) error {
	srv.doLog("Stop Web Service")
	err := ws.srvw.Stop()
	ws.mxService.Lock()
	ws.srvw = nil
	ws.mxService.Unlock()
	return err
}

func (ws *webSub) GetStatistics() []service.KeyValue {
	return nil
}

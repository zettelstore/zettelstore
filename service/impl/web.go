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
	"strconv"
	"sync"
	"time"

	"zettelstore.de/z/service"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/web/server/impl"
)

type webSub struct {
	subConfig
	mxService   sync.RWMutex
	srvw        server.Server
	setupServer service.SetupWebServerFunc
}

// Constants for web subservice keys.
const (
	WebSecureCookie      = "secure"
	WebListenAddress     = "listen"
	WebPersistentCookie  = "persistent"
	WebTokenLifetimeAPI  = "api-lifetime"
	WebTokenLifetimeHTML = "html-lifetime"
	WebURLPrefix         = "prefix"
)

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
		service.WebPersistentCookie: {"Persistent cookie", parseBool, true},
		service.WebSecureCookie:     {"Secure cookie", parseBool, true},
		service.WebTokenLifetimeAPI: {
			"Token lifetime API",
			makeDurationParser(10*time.Minute, 0, 1*time.Hour),
			true,
		},
		service.WebTokenLifetimeHTML: {
			"Token lifetime HTML",
			makeDurationParser(1*time.Hour, 1*time.Minute, 30*24*time.Hour),
			true,
		},
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
		service.WebListenAddress:     "127.0.0.1:23123",
		service.WebPersistentCookie:  false,
		service.WebSecureCookie:      true,
		service.WebTokenLifetimeAPI:  1 * time.Hour,
		service.WebTokenLifetimeHTML: 10 * time.Minute,
		service.WebURLPrefix:         "/",
	}
}

func makeDurationParser(defDur, minDur, maxDur time.Duration) parseFunc {
	return func(val string) interface{} {
		if d, err := strconv.ParseUint(val, 10, 64); err == nil {
			secs := time.Duration(d) * time.Minute
			if secs < minDur {
				return minDur
			}
			if secs > maxDur {
				return maxDur
			}
			return secs
		}
		return defDur
	}
}

func (ws *webSub) Start(srv *myService) error {
	listenAddr := ws.GetNextConfig(service.WebListenAddress).(string)
	urlPrefix := ws.GetNextConfig(service.WebURLPrefix).(string)
	persistentCookie := ws.GetNextConfig(service.WebPersistentCookie).(bool)
	secureCookie := ws.GetNextConfig(service.WebSecureCookie).(bool)

	srvw := impl.New(listenAddr, urlPrefix, persistentCookie, secureCookie, srv.auth.manager)
	err := srv.web.setupServer(srvw, srv.cfg.rtConfig, srv.place.manager, srv.auth.manager)
	if err != nil {
		srv.doLog("Unable to create Web Server:", err)
		return err
	}
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

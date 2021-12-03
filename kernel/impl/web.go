//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package impl

import (
	"net"
	"strconv"
	"sync"
	"time"

	"zettelstore.de/z/kernel"
	"zettelstore.de/z/kernel/logger"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/web/server/impl"
)

type webService struct {
	srvConfig
	mxService   sync.RWMutex
	srvw        server.Server
	setupServer kernel.SetupWebServerFunc
}

// Constants for web service keys.
const (
	WebSecureCookie      = "secure"
	WebListenAddress     = "listen"
	WebPersistentCookie  = "persistent"
	WebTokenLifetimeAPI  = "api-lifetime"
	WebTokenLifetimeHTML = "html-lifetime"
	WebURLPrefix         = "prefix"
)

func (ws *webService) Initialize(logger *logger.Logger) {
	ws.logger = logger
	ws.descr = descriptionMap{
		kernel.WebListenAddress: {
			"Listen address",
			func(val string) interface{} {
				host, port, err := net.SplitHostPort(val)
				if err != nil {
					return nil
				}
				if _, err = net.LookupPort("tcp", port); err != nil {
					return nil
				}
				return net.JoinHostPort(host, port)
			},
			true},
		kernel.WebPersistentCookie: {"Persistent cookie", parseBool, true},
		kernel.WebSecureCookie:     {"Secure cookie", parseBool, true},
		kernel.WebTokenLifetimeAPI: {
			"Token lifetime API",
			makeDurationParser(10*time.Minute, 0, 1*time.Hour),
			true,
		},
		kernel.WebTokenLifetimeHTML: {
			"Token lifetime HTML",
			makeDurationParser(1*time.Hour, 1*time.Minute, 30*24*time.Hour),
			true,
		},
		kernel.WebURLPrefix: {
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
		kernel.WebListenAddress:     "127.0.0.1:23123",
		kernel.WebPersistentCookie:  false,
		kernel.WebSecureCookie:      true,
		kernel.WebTokenLifetimeAPI:  1 * time.Hour,
		kernel.WebTokenLifetimeHTML: 10 * time.Minute,
		kernel.WebURLPrefix:         "/",
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

func (ws *webService) GetLogger() *logger.Logger { return ws.logger }

func (ws *webService) Start(kern *myKernel) error {
	listenAddr := ws.GetNextConfig(kernel.WebListenAddress).(string)
	urlPrefix := ws.GetNextConfig(kernel.WebURLPrefix).(string)
	persistentCookie := ws.GetNextConfig(kernel.WebPersistentCookie).(bool)
	secureCookie := ws.GetNextConfig(kernel.WebSecureCookie).(bool)

	srvw := impl.New(listenAddr, urlPrefix, persistentCookie, secureCookie, kern.auth.manager)
	err := kern.web.setupServer(srvw, kern.box.manager, kern.auth.manager, kern.cfg.rtConfig)
	if err != nil {
		ws.logger.Fatal().Err(err).Msg("Unable to create")
		return err
	}
	if kern.core.GetConfig(kernel.CoreDebug).(bool) {
		srvw.SetDebug()
	}
	if err = srvw.Run(); err != nil {
		ws.logger.Fatal().Err(err).Msg("Unable to start")
		return err
	}
	ws.logger.Info().Str("listen", listenAddr).Msg("Start Service")
	ws.mxService.Lock()
	ws.srvw = srvw
	ws.mxService.Unlock()
	return nil
}

func (ws *webService) IsStarted() bool {
	ws.mxService.RLock()
	defer ws.mxService.RUnlock()
	return ws.srvw != nil
}

func (ws *webService) Stop(*myKernel) error {
	ws.logger.Info().Msg("Stop Service")
	err := ws.srvw.Stop()
	ws.mxService.Lock()
	ws.srvw = nil
	ws.mxService.Unlock()
	return err
}

func (*webService) GetStatistics() []kernel.KeyValue {
	return nil
}

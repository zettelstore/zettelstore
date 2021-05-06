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

import "zettelstore.de/z/service"

func (srv *myService) SetConfig(subsrv service.Subservice, key, value string) {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	switch subsrv {
	case service.SubMain:
		srv.config.sys[key] = value
	case service.SubWeb:
		srv.config.webNext[key] = value
	}
}

func (srv *myService) GetConfig(subsrv service.Subservice, key string) string {
	srv.mx.RLock()
	defer srv.mx.RUnlock()
	switch subsrv {
	case service.SubMain:
		return srv.config.sys[key]
	case service.SubWeb:
		if srv.config.webCur == nil {
			return srv.config.webNext[key]
		}
		return srv.config.webCur[key]
	}
	return ""
}

func (srv *myService) switchNextToCur(subsrv service.Subservice) {
	switch subsrv {
	case service.SubWeb:
		srv.config.webCur = srv.config.webNext.Clone()
	}
}

type stringMap map[string]string

func (m stringMap) Clone() stringMap {
	if m == nil {
		return nil
	}
	result := make(stringMap, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

type srvConfig struct {
	sys     stringMap
	webCur  stringMap
	webNext stringMap
}

func (cfg *srvConfig) Initialize() {
	cfg.sys = make(stringMap)
	cfg.webNext = stringMap{
		service.WebListenAddress: "127.0.0.1:23123",
	}
}
func (cfg *srvConfig) SetConfig(subsrv service.Subservice, key, value string) {
	switch subsrv {
	case service.SubMain:
		cfg.sys[key] = value
	case service.SubWeb:
		cfg.webNext[key] = value
	}
}

func (cfg *srvConfig) GetConfig(subsrv service.Subservice, key string) string {
	switch subsrv {
	case service.SubMain:
		return cfg.sys[key]
	case service.SubWeb:
		if cfg.webCur == nil {
			return cfg.webNext[key]
		}
		return cfg.webCur[key]
	}
	return ""
}

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
	"fmt"
	"os"
	"runtime"
	"sort"

	"zettelstore.de/z/service"
)

func (srv *myService) SetConfig(subsrv service.Subservice, key, value string) bool {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	return srv.config.SetConfig(subsrv, key, value)
}

func (srv *myService) GetConfig(subsrv service.Subservice, key string) interface{} {
	srv.mx.RLock()
	defer srv.mx.RUnlock()
	return srv.config.GetConfig(subsrv, key)
}

func (srv *myService) GetConfigList(subsrv service.Subservice) []service.KeyDescrValue {
	srv.mx.RLock()
	defer srv.mx.RUnlock()
	return srv.config.getConfigList(subsrv)
}

type descriptionMap map[string]struct {
	text    string
	parse   func(string) interface{}
	canList bool
}
type interfaceMap map[string]interface{}

func (m interfaceMap) Clone() interfaceMap {
	if m == nil {
		return nil
	}
	result := make(interfaceMap, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

type srvConfig struct {
	frozen    bool
	mainDescr descriptionMap
	mainCur   interfaceMap
	webDescr  descriptionMap
	webCur    interfaceMap
	webNext   interfaceMap
}

func (cfg *srvConfig) Initialize() {
	cfg.mainDescr = descriptionMap{
		service.MainGoArch:    {"Go processor architecture", nil, false},
		service.MainGoOS:      {"Go Operating System", nil, false},
		service.MainGoVersion: {"Go Version", nil, false},
		service.MainHostname:  {"Host name", nil, false},
		service.MainProgname:  {"Program name", nil, false},
		service.MainReadonly:  {"Read-only mode", parseBool, true},
		service.MainVersion: {"Version", func(val string) interface{} {
			if cfg.frozen {
				return nil
			}
			if val == "" {
				return "unknown"
			}
			return val
		}, false},
	}
	cfg.mainCur = interfaceMap{
		service.MainGoArch:    runtime.GOARCH,
		service.MainGoOS:      runtime.GOOS,
		service.MainGoVersion: runtime.Version(),
		service.MainHostname:  "*unknwon host*",
		service.MainReadonly:  false,
	}
	if hn, err := os.Hostname(); err == nil {
		cfg.mainCur[service.MainHostname] = hn
	}

	cfg.webDescr = descriptionMap{
		service.WebListenAddress: {"Listen address, format [IP_ADDRESS]:PORT", parseString, true},
		service.WebURLPrefix: {
			"URL prefix under which the web server runs",
			func(val string) interface{} {
				if val != "" && val[0] == '/' && val[len(val)-1] == '/' {
					return val
				}
				return nil
			}, true},
	}
	cfg.webNext = interfaceMap{
		service.WebListenAddress: "127.0.0.1:23123",
		service.WebURLPrefix:     "/",
	}
}

func parseString(val string) interface{} { return val }
func parseBool(val string) interface{} {
	if val == "" {
		return false
	}
	switch val[0] {
	case '0', 'f', 'F', 'n', 'N':
		return false
	}
	return true
}

func (cfg *srvConfig) SetConfig(subsrv service.Subservice, key, value string) bool {
	switch subsrv {
	case service.SubMain:
		return cfg.storeConfig(cfg.mainCur, key, value, cfg.mainDescr)
	case service.SubWeb:
		return cfg.storeConfig(cfg.webNext, key, value, cfg.webDescr)
	}
	return false
}

func (cfg *srvConfig) storeConfig(iMap interfaceMap, key, value string, dMap descriptionMap) bool {
	descr, ok := dMap[key]
	if !ok {
		return false
	}
	parse := descr.parse
	if parse == nil {
		if cfg.frozen {
			return false
		}
		iMap[key] = value
		return true
	}
	iVal := parse(value)
	if iVal == nil {
		return false
	}
	iMap[key] = iVal
	return true
}

func (cfg *srvConfig) GetConfig(subsrv service.Subservice, key string) interface{} {
	switch subsrv {
	case service.SubMain:
		return cfg.mainCur[key]
	case service.SubWeb:
		if cfg.webCur == nil {
			return cfg.webNext[key]
		}
		return cfg.webCur[key]
	}
	return ""
}

func (cfg *srvConfig) getConfigList(subsrv service.Subservice) []service.KeyDescrValue {
	var descrMap descriptionMap
	switch subsrv {
	case service.SubMain:
		descrMap = cfg.mainDescr
	case service.SubWeb:
		descrMap = cfg.webDescr
	}
	if len(descrMap) == 0 {
		return nil
	}
	keys := make([]string, 0, len(descrMap))
	for k, descr := range descrMap {
		if descr.canList {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	result := make([]service.KeyDescrValue, 0, len(keys))
	for _, k := range keys {
		val := cfg.GetConfig(subsrv, k)
		if val == nil {
			continue
		}
		result = append(result, service.KeyDescrValue{
			Key:   k,
			Descr: descrMap[k].text,
			Value: fmt.Sprintf("%v", val),
		})
	}
	return result
}

func (cfg *srvConfig) switchNextToCur(subsrv service.Subservice) {
	switch subsrv {
	case service.SubWeb:
		cfg.webCur = cfg.webNext.Clone()
	}
}

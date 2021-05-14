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
	"sort"
	"sync"

	"zettelstore.de/z/service"
)

type parseFunc func(string) interface{}
type descriptionMap map[string]struct {
	text    string
	parse   parseFunc
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

type subConfig struct {
	mx     sync.RWMutex
	frozen bool
	descr  descriptionMap
	cur    interfaceMap
	next   interfaceMap
}

func (cfg *subConfig) noFrozen(parse parseFunc) parseFunc {
	return func(val string) interface{} {
		if cfg.frozen {
			return nil
		}
		return parse(val)
	}
}

func (cfg *subConfig) SetConfig(key, value string) bool {
	cfg.mx.Lock()
	defer cfg.mx.Unlock()
	descr, ok := cfg.descr[key]
	if !ok {
		return false
	}
	parse := descr.parse
	if parse == nil {
		if cfg.frozen {
			return false
		}
		cfg.next[key] = value
		return true
	}
	iVal := parse(value)
	if iVal == nil {
		return false
	}
	cfg.next[key] = iVal
	return true
}

func (cfg *subConfig) GetConfig(key string) interface{} {
	cfg.mx.RLock()
	defer cfg.mx.RUnlock()
	if cfg.cur == nil {
		return cfg.next[key]
	}
	return cfg.cur[key]
}

func (cfg *subConfig) GetNextConfig(key string) interface{} {
	cfg.mx.RLock()
	defer cfg.mx.RUnlock()
	return cfg.next[key]
}

func (cfg *subConfig) GetConfigList(all bool) []service.KeyDescrValue {
	return cfg.getConfigList(all, cfg.GetConfig)
}
func (cfg *subConfig) GetNextConfigList() []service.KeyDescrValue {
	return cfg.getConfigList(true, cfg.GetNextConfig)
}
func (cfg *subConfig) getConfigList(all bool, getConfig func(string) interface{}) []service.KeyDescrValue {
	if len(cfg.descr) == 0 {
		return nil
	}
	keys := make([]string, 0, len(cfg.descr))
	for k, descr := range cfg.descr {
		if all || descr.canList {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	result := make([]service.KeyDescrValue, 0, len(keys))
	for _, k := range keys {
		val := getConfig(k)
		if val == nil {
			continue
		}
		result = append(result, service.KeyDescrValue{
			Key:   k,
			Descr: cfg.descr[k].text,
			Value: fmt.Sprintf("%v", val),
		})
	}
	return result
}

func (cfg *subConfig) Freeze() {
	cfg.mx.Lock()
	cfg.frozen = true
	cfg.mx.Unlock()
}

func (cfg *subConfig) SwitchNextToCur() {
	cfg.mx.Lock()
	defer cfg.mx.Unlock()
	cfg.cur = cfg.next.Clone()
}

// func parseString(val string) interface{} { return val }
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

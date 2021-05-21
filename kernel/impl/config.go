//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the kernel implementation.
package impl

import (
	"fmt"
	"sort"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/kernel"
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

type srvConfig struct {
	mxConfig sync.RWMutex
	frozen   bool
	descr    descriptionMap
	cur      interfaceMap
	next     interfaceMap
}

func (cfg *srvConfig) noFrozen(parse parseFunc) parseFunc {
	return func(val string) interface{} {
		if cfg.frozen {
			return nil
		}
		return parse(val)
	}
}

func (cfg *srvConfig) SetConfig(key, value string) bool {
	cfg.mxConfig.Lock()
	defer cfg.mxConfig.Unlock()
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

func (cfg *srvConfig) GetConfig(key string) interface{} {
	cfg.mxConfig.RLock()
	defer cfg.mxConfig.RUnlock()
	if cfg.cur == nil {
		return cfg.next[key]
	}
	return cfg.cur[key]
}

func (cfg *srvConfig) GetNextConfig(key string) interface{} {
	cfg.mxConfig.RLock()
	defer cfg.mxConfig.RUnlock()
	return cfg.next[key]
}

func (cfg *srvConfig) GetConfigList(all bool) []kernel.KeyDescrValue {
	return cfg.getConfigList(all, cfg.GetConfig)
}
func (cfg *srvConfig) GetNextConfigList() []kernel.KeyDescrValue {
	return cfg.getConfigList(true, cfg.GetNextConfig)
}
func (cfg *srvConfig) getConfigList(all bool, getConfig func(string) interface{}) []kernel.KeyDescrValue {
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
	result := make([]kernel.KeyDescrValue, 0, len(keys))
	for _, k := range keys {
		val := getConfig(k)
		if val == nil {
			continue
		}
		result = append(result, kernel.KeyDescrValue{
			Key:   k,
			Descr: cfg.descr[k].text,
			Value: fmt.Sprintf("%v", val),
		})
	}
	return result
}

func (cfg *srvConfig) Freeze() {
	cfg.mxConfig.Lock()
	cfg.frozen = true
	cfg.mxConfig.Unlock()
}

func (cfg *srvConfig) SwitchNextToCur() {
	cfg.mxConfig.Lock()
	defer cfg.mxConfig.Unlock()
	cfg.cur = cfg.next.Clone()
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

func parseZid(val string) interface{} {
	if zid, err := id.Parse(val); err == nil {
		return zid
	}
	return id.Invalid
}

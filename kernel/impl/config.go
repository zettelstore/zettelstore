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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/kernel/logger"
)

type parseFunc func(string) interface{}
type configDescription struct {
	text    string
	parse   parseFunc
	canList bool
}
type descriptionMap map[string]configDescription
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
	logger   *logger.Logger
	mxConfig sync.RWMutex
	frozen   bool
	descr    descriptionMap
	cur      interfaceMap
	next     interfaceMap
}

func (cfg *srvConfig) ConfigDescriptions() []serviceConfigDescription {
	cfg.mxConfig.RLock()
	defer cfg.mxConfig.RUnlock()
	keys := make([]string, 0, len(cfg.descr))
	for k := range cfg.descr {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]serviceConfigDescription, 0, len(keys))
	for _, k := range keys {
		text := cfg.descr[k].text
		if strings.HasSuffix(k, "-") {
			text = text + " (list)"
		}
		result = append(result, serviceConfigDescription{Key: k, Descr: text})
	}
	return result
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
		d, baseKey, num := cfg.getListDescription(key)
		if num < 0 {
			return false
		}
		format := baseKey + "%d"
		for i := num + 1; ; i++ {
			k := fmt.Sprintf(format, i)
			if _, ok = cfg.next[k]; !ok {
				break
			}
			delete(cfg.next, k)
		}
		if num == 0 {
			return true
		}
		descr = d
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

func (cfg *srvConfig) getListDescription(key string) (configDescription, string, int) {
	for k, d := range cfg.descr {
		if !strings.HasSuffix(k, "-") {
			continue
		}
		if !strings.HasPrefix(key, k) {
			continue
		}
		num, err := strconv.Atoi(key[len(k):])
		if err != nil || num < 0 {
			continue
		}
		return d, k, num
	}
	return configDescription{}, "", -1
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
	keys := cfg.getSortedConfigKeys(all, getConfig)
	result := make([]kernel.KeyDescrValue, 0, len(keys))
	for _, k := range keys {
		val := getConfig(k)
		if val == nil {
			continue
		}
		descr, ok := cfg.descr[k]
		if !ok {
			descr, _, _ = cfg.getListDescription(k)
		}
		result = append(result, kernel.KeyDescrValue{
			Key:   k,
			Descr: descr.text,
			Value: fmt.Sprintf("%v", val),
		})
	}
	return result
}

func (cfg *srvConfig) getSortedConfigKeys(all bool, getConfig func(string) interface{}) []string {
	keys := make([]string, 0, len(cfg.descr))
	for k, descr := range cfg.descr {
		if all || descr.canList {
			if !strings.HasSuffix(k, "-") {
				keys = append(keys, k)
				continue
			}
			format := k + "%d"
			for i := 1; ; i++ {
				key := fmt.Sprintf(format, i)
				val := getConfig(key)
				if val == nil {
					break
				}
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	return keys
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

func parseInt(val string) interface{} {
	i, err := strconv.Atoi(val)
	if err == nil {
		return i
	}
	return 0
}

func parseZid(val string) interface{} {
	if zid, err := id.Parse(val); err == nil {
		return zid
	}
	return id.Invalid
}

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
	"strconv"
	"strings"
	"sync"

	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/service"
)

type cfgSub struct {
	subConfig
	mxService sync.RWMutex
	rtConfig  *config.Config
}

func (cs *cfgSub) Initialize() {
	cs.descr = descriptionMap{
		meta.KeyCopyright:  {"Copyright", parseString, true},
		meta.KeyExpertMode: {"Expert mode", parseBool, true},
		meta.KeyFooterHTML: {"Footer HTML", parseString, true},
		meta.KeyHomeZettel: {"Home zettel", parseZid, true},
		meta.KeyListPageSize: {
			"List page size",
			func(val string) interface{} {
				iVal, err := strconv.Atoi(val)
				if err != nil {
					return nil
				}
				return iVal
			},
			true,
		},
		meta.KeyDefaultLang:    {"Language", parseString, true},
		meta.KeyMarkerExternal: {"Marker external URL", parseString, true},
		meta.KeyDefaultRole:    {"Default role", parseString, true},
		meta.KeySiteName:       {"Site name", parseString, true},
		meta.KeyDefaultSyntax:  {"Default syntax", parseString, true},
		meta.KeyDefaultTitle:   {"Default title", parseString, true},
		meta.KeyDefaultVisibility: {
			"Default zettel visibility",
			func(val string) interface{} {
				vis := meta.GetVisibility(val)
				if vis == meta.VisibilityUnknown {
					return nil
				}
				return vis
			},
			true,
		},
		meta.KeyYAMLHeader: {"YAML header", parseBool, true},
		meta.KeyZettelFileSyntax: {
			"Zettel file syntax",
			func(val string) interface{} { return strings.Fields(val) },
			true,
		},
	}
	cs.next = interfaceMap{
		meta.KeyCopyright:         "",
		meta.KeyExpertMode:        false,
		meta.KeyFooterHTML:        "",
		meta.KeyHomeZettel:        id.DefaultHomeZid,
		meta.KeyListPageSize:      0,
		meta.KeyDefaultLang:       meta.ValueLangEN,
		meta.KeyMarkerExternal:    "&#10138;",
		meta.KeyDefaultRole:       meta.ValueRoleZettel,
		meta.KeySiteName:          "Zettelstore",
		meta.KeyDefaultSyntax:     meta.ValueSyntaxZmk,
		meta.KeyDefaultTitle:      "Untitled",
		meta.KeyDefaultVisibility: meta.VisibilityLogin,
		meta.KeyYAMLHeader:        false,
		meta.KeyZettelFileSyntax:  nil,
	}
}

func (cs *cfgSub) Start(srv *myService) error {
	srv.doLog("Start Config Service")
	data := meta.New(id.ConfigurationZid)
	for _, kv := range cs.GetNextConfigList() {
		data.Set(kv.Key, fmt.Sprintf("%v", kv.Value))
	}
	rtConfig, err := config.New(data, srv.place.manager)
	if err != nil {
		return err
	}
	cs.mxService.Lock()
	cs.rtConfig = rtConfig
	cs.mxService.Unlock()
	return nil
}

func (cs *cfgSub) IsStarted() bool {
	cs.mxService.RLock()
	defer cs.mxService.RUnlock()
	return cs.rtConfig != nil
}

func (cs *cfgSub) Stop(srv *myService) error {
	srv.doLog("Stop Config Service")
	cs.mxService.Lock()
	cs.rtConfig = nil
	cs.mxService.Unlock()
	return nil
}

func (cs *cfgSub) GetStatistics() []service.KeyValue {
	return nil
}

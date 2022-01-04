//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package filebox provides boxes that are stored in a file.
package filebox

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
)

func init() {
	manager.Register("file", func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
		path := getFilepathFromURL(u)
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".zip" {
			return nil, errors.New("unknown extension '" + ext + "' in box URL: " + u.String())
		}
		return &zipBox{
			log: kernel.Main.GetLogger(kernel.BoxService).Clone().
				Str("box", "zip").Int("boxnum", int64(cdata.Number)).Child(),
			number:   cdata.Number,
			name:     path,
			enricher: cdata.Enricher,
			notify:   cdata.Notify,
		}, nil
	})
}

func getFilepathFromURL(u *url.URL) string {
	name := u.Opaque
	if name == "" {
		name = u.Path
	}
	components := strings.Split(name, "/")
	fileName := filepath.Join(components...)
	if len(components) > 0 && components[0] == "" {
		return "/" + fileName
	}
	return fileName
}

var alternativeSyntax = map[string]string{
	"htm": "html",
}

func calculateSyntax(ext string) string {
	ext = strings.ToLower(ext)
	if syntax, ok := alternativeSyntax[ext]; ok {
		return syntax
	}
	return ext
}

// CalcDefaultMeta returns metadata with default values for the given entry.
func CalcDefaultMeta(zid id.Zid, ext string) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, zid.String())
	m.Set(api.KeySyntax, calculateSyntax(ext))
	return m
}

// CleanupMeta enhances the given metadata.
func CleanupMeta(m *meta.Meta, zid id.Zid, ext string, inMeta bool, duplicates []string) {
	if title, ok := m.Get(api.KeyTitle); !ok || title == "" {
		m.Set(api.KeyTitle, zid.String())
	}

	if inMeta {
		if syntax, ok := m.Get(api.KeySyntax); !ok || syntax == "" {
			dm := CalcDefaultMeta(zid, ext)
			syntax, ok = dm.Get(api.KeySyntax)
			if !ok {
				panic("Default meta must contain syntax")
			}
			m.Set(api.KeySyntax, syntax)
		}
	}

	if len(duplicates) > 0 {
		m.Set(api.KeyDuplicates, strings.Join(duplicates, " "))
	}
}

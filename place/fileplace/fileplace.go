//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package fileplace provides places that are stored in a file.
package fileplace

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
)

func init() {
	manager.Register("file", func(u *url.URL, cdata *manager.ConnectData) (place.ManagedPlace, error) {
		path := getFilepathFromURL(u)
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".zip" {
			return nil, errors.New("unknown extension '" + ext + "' in place URL: " + u.String())
		}
		return &zipPlace{name: path, enricher: cdata.Enricher}, nil
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
	m.Set(meta.KeyTitle, zid.String())
	m.Set(meta.KeySyntax, calculateSyntax(ext))
	return m
}

// CleanupMeta enhances the given metadata.
func CleanupMeta(m *meta.Meta, zid id.Zid, ext string, inMeta, duplicates bool) {
	if title, ok := m.Get(meta.KeyTitle); !ok || title == "" {
		m.Set(meta.KeyTitle, zid.String())
	}

	if inMeta {
		if syntax, ok := m.Get(meta.KeySyntax); !ok || syntax == "" {
			dm := CalcDefaultMeta(zid, ext)
			syntax, ok = dm.Get(meta.KeySyntax)
			if !ok {
				panic("Default meta must contain syntax")
			}
			m.Set(meta.KeySyntax, syntax)
		}
	}

	if duplicates {
		m.Set(meta.KeyDuplicates, meta.ValueTrue)
	}
}

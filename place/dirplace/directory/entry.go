//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package directory manages the directory part of a dirstore.
package directory

import (
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// MetaSpec defines all possibilities where meta data can be stored.
type MetaSpec int

// Constants for MetaSpec
const (
	MetaSpecUnknown MetaSpec = iota
	MetaSpecNone             // no meta information
	MetaSpecFile             // meta information is in meta file
	MetaSpecHeader           // meta information is in header
)

// Entry stores everything for a directory entry.
type Entry struct {
	Zid         id.Zid
	MetaSpec    MetaSpec // location of meta information
	MetaPath    string   // file path of meta information
	ContentPath string   // file path of zettel content
	ContentExt  string   // (normalized) file extension of zettel content
	Duplicates  bool     // multiple content files
}

// IsValid checks whether the entry is valid.
func (e *Entry) IsValid() bool {
	return e.Zid.IsValid()
}

var alternativeSyntax = map[string]string{
	"htm": "html",
}

func (e *Entry) calculateSyntax() string {
	ext := strings.ToLower(e.ContentExt)
	if syntax, ok := alternativeSyntax[ext]; ok {
		return syntax
	}
	return ext
}

// CalcDefaultMeta returns metadata with default values for the given entry.
func (e *Entry) CalcDefaultMeta() *meta.Meta {
	m := meta.New(e.Zid)
	m.Set(meta.KeyTitle, e.Zid.String())
	m.Set(meta.KeySyntax, e.calculateSyntax())
	return m
}

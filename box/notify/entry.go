//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

package notify

import (
	"path/filepath"

	"t73f.de/r/zsc/api"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

const (
	extZettel = "zettel" // file contains metadata and content
	extBin    = "bin"    // file contains binary content
	extTxt    = "txt"    // file contains non-binary content
)

func extIsMetaAndContent(ext string) bool { return ext == extZettel }

// DirEntry stores everything for a directory entry.
type DirEntry struct {
	Zid          id.Zid
	MetaName     string   // file name of meta information
	ContentName  string   // file name of zettel content
	ContentExt   string   // (normalized) file extension of zettel content
	UselessFiles []string // list of other content files
}

// IsValid checks whether the entry is valid.
func (e *DirEntry) IsValid() bool {
	return e != nil && e.Zid.IsValid()
}

// HasMetaInContent returns true, if metadata will be stored in the content file.
func (e *DirEntry) HasMetaInContent() bool {
	return e.IsValid() && extIsMetaAndContent(e.ContentExt)
}

// SetupFromMetaContent fills entry data based on metadata and zettel content.
func (e *DirEntry) SetupFromMetaContent(m *meta.Meta, content zettel.Content, getZettelFileSyntax func() []string) {
	if e.Zid != m.Zid {
		panic("Zid differ")
	}
	if contentName := e.ContentName; contentName != "" {
		if !extIsMetaAndContent(e.ContentExt) && e.MetaName == "" {
			e.MetaName = e.calcBaseName(contentName)
		}
		return
	}

	syntax := m.GetDefault(api.KeySyntax, meta.DefaultSyntax)
	ext := calcContentExt(syntax, m.YamlSep, getZettelFileSyntax)
	metaName := e.MetaName
	eimc := extIsMetaAndContent(ext)
	if eimc {
		if metaName != "" {
			ext = contentExtWithMeta(syntax, content)
		}
		e.ContentName = e.calcBaseName(metaName) + "." + ext
		e.ContentExt = ext
	} else {
		if len(content.AsBytes()) > 0 {
			e.ContentName = e.calcBaseName(metaName) + "." + ext
			e.ContentExt = ext
		}
		if metaName == "" {
			e.MetaName = e.calcBaseName(e.ContentName)
		}
	}
}

func contentExtWithMeta(syntax string, content zettel.Content) string {
	p := parser.Get(syntax)
	if content.IsBinary() {
		if p.IsImageFormat {
			return syntax
		}
		return extBin
	}
	if p.IsImageFormat {
		return extTxt
	}
	return syntax
}

func calcContentExt(syntax string, yamlSep bool, getZettelFileSyntax func() []string) string {
	if yamlSep {
		return extZettel
	}
	switch syntax {
	case meta.SyntaxNone, meta.SyntaxZmk:
		return extZettel
	}
	for _, s := range getZettelFileSyntax() {
		if s == syntax {
			return extZettel
		}
	}
	return syntax

}

func (e *DirEntry) calcBaseName(name string) string {
	if name == "" {
		return e.Zid.String()
	}
	return name[0 : len(name)-len(filepath.Ext(name))]

}

//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlgen encodes the abstract syntax tree into HTML5 (deprecated).
// It is only used for the WebUI and will be deprecated by a software that
// uses the zettelstore-client HTML encoder.
package htmlgen

import (
	"bytes"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

// Create a new encoder.
func Create(extMarker string, newWindow bool) *Encoder {
	return &Encoder{extMarker: extMarker, newWindow: newWindow}
}

// Encoder encapsulates the encoder itself
type Encoder struct {
	extMarker   string
	newWindow   bool
	footnotes   []footnoteInfo // Stores footnotes detected while encoding
	footnoteNum int
}
type footnoteInfo struct {
	fn  *ast.FootnoteNode
	num int
}

// addFootnote adds a footnote node to the environment and returns the number of that footnote.
func (he *Encoder) addFootnote(fn *ast.FootnoteNode) int {
	he.footnoteNum++
	he.footnotes = append(he.footnotes, footnoteInfo{fn: fn, num: he.footnoteNum})
	return he.footnoteNum
}

// popFootnote returns the next footnote and removes it from the list.
func (he *Encoder) popFootnote() (*ast.FootnoteNode, int) {
	if len(he.footnotes) == 0 {
		he.footnotes = nil
		he.footnoteNum = 0
		return nil, -1
	}
	fni := he.footnotes[0]
	he.footnotes = he.footnotes[1:]
	return fni.fn, fni.num
}

// MetaString encodes meta data as HTML5.
func (he *Encoder) MetaString(m *meta.Meta, evalMeta encoder.EvalMetaFunc) (string, error) {
	var buf bytes.Buffer
	v := newVisitor(he, &buf)

	// Write title
	if title, ok := m.Get(api.KeyTitle); ok {
		v.b.WriteStrings("<meta name=\"zs-", api.KeyTitle, "\" content=\"")
		v.writeQuotedEscaped(v.evalValue(title, evalMeta))
		v.b.WriteString("\">")
	}

	// Write other metadata
	ignore := strfun.NewSet(api.KeyTitle, api.KeyLang)
	if tags, ok := m.Get(api.KeyAllTags); ok {
		v.writeTags(tags)
		ignore.Set(api.KeyAllTags)
		ignore.Set(api.KeyTags)
	} else if tags, ok = m.Get(api.KeyTags); ok {
		v.writeTags(tags)
		ignore.Set(api.KeyTags)
	}

	for _, p := range m.ComputedPairs() {
		key := p.Key
		if ignore.Has(key) {
			continue
		}
		value := p.Value
		if m.Type(key) == meta.TypeZettelmarkup {
			if v := v.evalValue(value, evalMeta); v != "" {
				value = v
			}
		}
		if mKey, ok := mapMetaKey[key]; ok {
			v.writeMeta("", mKey, value)
		} else {
			v.writeMeta("zs-", key, value)
		}
	}
	return v.makeResult(&buf)
}

var mapMetaKey = map[string]string{
	api.KeyCopyright: "copyright",
	api.KeyLicense:   "license",
}

// BlocksString encodes a block slice.
func (he *Encoder) BlocksString(bs *ast.BlockSlice) (string, error) {
	var buf bytes.Buffer
	v := newVisitor(he, &buf)
	ast.Walk(v, bs)
	v.writeEndnotes()
	return v.makeResult(&buf)
}

// InlinesString writes an inline slice to the writer
func (he *Encoder) InlinesString(is *ast.InlineSlice, noLink bool) (string, error) {
	if is == nil || len(*is) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	v := newVisitor(he, &buf)
	v.noLink = noLink
	ast.Walk(v, is)
	return v.makeResult(&buf)
}

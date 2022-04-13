//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"bytes"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/c/html"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/encoder/zjsonenc"
	"zettelstore.de/z/strfun"
)

type htmlGenerator struct {
	textEnc   *textenc.Encoder
	zjsonEnc  *zjsonenc.Encoder
	extMarker string
	newWindow bool
	enc       *html.Encoder
}

func createGenerator(extMarker string, newWindow bool) *htmlGenerator {
	return &htmlGenerator{
		textEnc:   textenc.Create(),
		zjsonEnc:  zjsonenc.Create(),
		extMarker: extMarker,
		newWindow: newWindow,
		enc:       html.NewEncoder(nil, 1),
	}
}

var mapMetaKey = map[string]string{
	api.KeyCopyright: "copyright",
	api.KeyLicense:   "license",
}

func (g *htmlGenerator) MetaString(m *meta.Meta, evalMeta encoder.EvalMetaFunc) (string, error) {
	ignore := strfun.Set{
		api.KeyTitle: struct{}{},
		api.KeyLang:  struct{}{},
	}
	var buf bytes.Buffer

	if tags, ok := m.Get(api.KeyAllTags); ok {
		g.writeTags(&buf, tags)
		ignore.Set(api.KeyAllTags)
		ignore.Set(api.KeyTags)
	} else if tags, ok = m.Get(api.KeyTags); ok {
		g.writeTags(&buf, tags)
		ignore.Set(api.KeyTags)
	}

	for _, p := range m.ComputedPairs() {
		key := p.Key
		if ignore.Has(key) {
			continue
		}
		if altKey, found := mapMetaKey[key]; found {
			buf.WriteString(`<meta name="`)
			buf.WriteString(altKey)
		} else {
			buf.WriteString(`<meta name="zs-`)
			buf.WriteString(key)
		}
		buf.WriteString(`" content="`)
		is := evalMeta(p.Value)
		var sb strings.Builder
		g.textEnc.WriteInlines(&sb, &is)
		html.AttributeEscape(&buf, sb.String())
		buf.WriteString("\">\n")
	}
	return buf.String(), nil
}
func (g *htmlGenerator) writeTags(buf *bytes.Buffer, tags string) {
	buf.WriteString(`<meta name="keywords" content="`)
	for i, val := range meta.ListFromValue(tags) {
		if i > 0 {
			buf.WriteString(", ")
		}
		html.AttributeEscape(buf, strings.TrimPrefix(val, "#"))
	}
	buf.WriteString("\">\n")
}

// BlocksString encodes a block slice.
func (g *htmlGenerator) BlocksString(bs *ast.BlockSlice) (string, error) {
	if bs == nil || len(*bs) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	if _, err := g.zjsonEnc.WriteBlocks(&buf, bs); err != nil {
		return "", err
	}

	val, err := zjson.Decode(&buf)
	if err != nil {
		return "", err
	}
	buf.Reset()
	g.enc.ReplaceWriter(&buf)
	g.enc.TraverseBlock(zjson.MakeArray(val))
	g.enc.WriteEndnotes()
	g.enc.ReplaceWriter(nil)
	return buf.String(), nil
}

// InlinesString writes an inline slice to the writer
func (g *htmlGenerator) InlinesString(is *ast.InlineSlice, noLink bool) (string, error) {
	if is == nil || len(*is) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	if _, err := g.zjsonEnc.WriteInlines(&buf, is); err != nil {
		return "", err
	}

	val, err := zjson.Decode(&buf)
	if err != nil {
		return "", err
	}
	buf = bytes.Buffer{} // free all encoded zjaon data
	return html.EncodeInline(g.enc, zjson.MakeArray(val), !noLink, noLink), nil
}

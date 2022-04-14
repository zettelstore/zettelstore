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
	"log"
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

// Builder allows to build new URLs for the web service.
type urlBuilder interface {
	GetURLPrefix() string
	NewURLBuilder(key byte) *api.URLBuilder
}

type htmlGenerator struct {
	builder   urlBuilder
	textEnc   *textenc.Encoder
	zjsonEnc  *zjsonenc.Encoder
	extMarker string
	newWindow bool
	enc       *html.Encoder
}

func createGenerator(builder urlBuilder, extMarker string, newWindow bool) *htmlGenerator {
	enc := html.NewEncoder(nil, 1)
	gen := &htmlGenerator{
		builder:   builder,
		textEnc:   textenc.Create(),
		zjsonEnc:  zjsonenc.Create(),
		extMarker: extMarker,
		newWindow: newWindow,
		enc:       enc,
	}
	enc.SetTypeFunc(zjson.TypeTag, gen.generateTag)
	enc.SetTypeFunc(zjson.TypeLink, gen.generateLink)
	return gen
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

func (g *htmlGenerator) generateTag(enc *html.Encoder, obj zjson.Object, pos int) (bool, zjson.CloseFunc) {
	if s := zjson.GetString(obj, zjson.NameString); s != "" {
		if enc.IgnoreLinks() {
			enc.WriteString(s)
		} else {
			u := g.builder.NewURLBuilder('h').AppendQuery(api.KeyAllTags, "#"+strings.ToLower(s))
			enc.WriteString(`<a href="`)
			enc.WriteString(u.String())
			enc.WriteString(`">#`)
			enc.WriteString(s)
			enc.WriteString("</a>")
		}
	}
	return false, nil
}

func (g *htmlGenerator) generateLink(enc *html.Encoder, obj zjson.Object, pos int) (bool, zjson.CloseFunc) {
	if enc.IgnoreLinks() {
		return enc.MustGetTypeFunc(zjson.TypeFormatSpan)(enc, obj, pos)
	}
	ref := zjson.GetString(obj, zjson.NameString)
	in := zjson.GetArray(obj, zjson.NameInline)
	if ref == "" {
		return len(in) > 0, nil
	}
	a := zjson.GetAttributes(obj)
	suffix := ""
	switch q := zjson.GetString(obj, zjson.NameString2); q {
	case zjson.RefStateExternal:
		a = a.Set("href", ref).
			AddClass("external").
			Set("target", "_blank").
			Set("rel", "noopener noreferrer")
		suffix = g.extMarker
	case zjson.RefStateZettel:
		u := g.builder.NewURLBuilder('h').SetZid(api.ZettelID(ref))
		a = a.Set("href", u.String())
	case zjson.RefStateHosted:
		a = a.Set("href", ref)
	case zjson.RefStateBased:
		u := g.builder.NewURLBuilder('/').SetRawLocal(ref)
		a = a.Set("href", u.String())
	case zjson.RefStateSelf:
		a = a.Set("href", ref)
	case zjson.RefStateBroken:
		a = a.AddClass("broken")
	default:
		log.Println("LINK", q, ref)
	}

	if len(a) > 0 {
		enc.WriteString("<a")
		enc.WriteAttributes(a)
		enc.WriteByte('>')
	}

	children := true
	if len(in) == 0 {
		enc.WriteString(ref)
		children = false
	}
	return children, func() {
		enc.WriteString("</a>")
		enc.WriteString(suffix)
	}
}

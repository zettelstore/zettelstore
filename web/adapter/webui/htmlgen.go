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

	"github.com/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/html"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/sexprenc"
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
	env       *html.EncEnvironment
}

func createGenerator(builder urlBuilder, extMarker string, newWindow bool) *htmlGenerator {
	enc := html.NewEncoder(nil, 1)
	env := html.NewEncEnvironment(nil, 1)
	gen := &htmlGenerator{
		builder:   builder,
		textEnc:   textenc.Create(),
		zjsonEnc:  zjsonenc.Create(),
		extMarker: extMarker,
		newWindow: newWindow,
		enc:       enc,
		env:       env,
	}
	enc.SetTypeFunc(zjson.TypeTag, gen.zgenerateTag)
	enc.SetTypeFunc(zjson.TypeLink, gen.zgenerateLink)
	enc.ChangeTypeFunc(zjson.TypeEmbed, gen.zmakeGenerateEmbed)

	env.Builtins.Set(sexpr.SymTag, sxpf.NewBuiltin("tag", true, 0, -1, gen.generateTag))
	env.Builtins.Set(sexpr.SymLink, sxpf.NewBuiltin("link", true, 2, -1, gen.generateLink))
	return gen
}

var mapMetaKey = map[string]string{
	api.KeyCopyright: "copyright",
	api.KeyLicense:   "license",
}

func (g *htmlGenerator) MetaString(m *meta.Meta, evalMeta encoder.EvalMetaFunc) string {
	ignore := strfun.NewSet(api.KeyTitle, api.KeyLang)
	var buf bytes.Buffer

	if tags, ok := m.Get(api.KeyAllTags); ok {
		writeMetaTags(&buf, tags)
		ignore.Set(api.KeyAllTags)
		ignore.Set(api.KeyTags)
	} else if tags, ok = m.Get(api.KeyTags); ok {
		writeMetaTags(&buf, tags)
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
	return buf.String()
}
func writeMetaTags(buf *bytes.Buffer, tags string) {
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
	lst := sexprenc.GetSexpr(bs)
	var buf bytes.Buffer
	g.env.ReplaceWriter(&buf)
	sxpf.Evaluate(g.env, lst)
	if g.env.GetError() == nil {
		g.env.WriteEndnotes()
	}
	g.env.ReplaceWriter(nil)
	return buf.String(), g.env.GetError()
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

func (g *htmlGenerator) zgenerateTag(enc *html.Encoder, obj zjson.Object, _ int) (bool, zjson.CloseFunc) {
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

func (g *htmlGenerator) generateTag(senv sxpf.Environment, args []sxpf.Value) (sxpf.Value, error) {
	if len(args) > 0 {
		env := senv.(*html.EncEnvironment)
		s := env.GetString(args, 0)
		if env.IgnoreLinks() {
			env.WriteEscaped(s)
		} else {
			u := g.builder.NewURLBuilder('h').AppendQuery(api.KeyAllTags, "#"+strings.ToLower(s))
			env.WriteStrings(`<a href="`, u.String(), `">#`)
			env.WriteEscaped(s)
			env.WriteString("</a>")
		}
	}
	return nil, nil
}

func (g *htmlGenerator) zgenerateLink(enc *html.Encoder, obj zjson.Object, pos int) (bool, zjson.CloseFunc) {
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
		zid, fragment, hasFragment := strings.Cut(ref, "#")
		u := g.builder.NewURLBuilder('h').SetZid(api.ZettelID(zid))
		if hasFragment {
			u = u.SetFragment(fragment)
		}
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

func (g *htmlGenerator) generateLink(senv sxpf.Environment, args []sxpf.Value) (sxpf.Value, error) {
	env := senv.(*html.EncEnvironment)
	if env.IgnoreLinks() {
		spanList := sxpf.NewArray(sexpr.SymFormatSpan)
		spanList.Append(args...)
		sxpf.Evaluate(env, spanList)
		return nil, nil
	}
	a := env.GetAttributes(args, 0)
	ref := env.GetList(args, 1)
	if ref == nil {
		return nil, nil
	}
	refPair := ref.GetValue()
	refKind := env.GetSymbol(refPair, 0)
	if refKind == nil {
		return nil, nil
	}
	refValue := env.GetString(refPair, 1)
	suffix := ""
	switch {
	case sexpr.SymRefStateExternal.Equal(refKind):
		a = a.Set("href", refValue).
			AddClass("external").
			Set("target", "_blank").
			Set("rel", "noopener noreferrer")
		suffix = g.extMarker
	case sexpr.SymRefStateZettel.Equal(refKind):
		zid, fragment, hasFragment := strings.Cut(refValue, "#")
		u := g.builder.NewURLBuilder('h').SetZid(api.ZettelID(zid))
		if hasFragment {
			u = u.SetFragment(fragment)
		}
		a = a.Set("href", u.String())
	case sexpr.SymRefStateBased.Equal(refKind):
		u := g.builder.NewURLBuilder('/').SetRawLocal(refValue)
		a = a.Set("href", u.String())
	case sexpr.SymRefStateHosted.Equal(refKind), sexpr.SymRefStateSelf.Equal(refKind):
		a = a.Set("href", refValue)
	case sexpr.SymRefStateBroken.Equal(refKind):
		a = a.AddClass("broken")
	default:
		log.Println("LINK", sxpf.NewArray(args...))
	}
	env.WriteString("<a")
	env.WriteAttributes(a)
	env.WriteString(">")

	if in := args[2:]; len(in) == 0 {
		env.WriteEscaped(refValue)
	} else {
		sxpf.EvaluateSlice(env, in)
	}
	env.WriteStrings("</a>", suffix)
	return nil, nil
}

func (g *htmlGenerator) zmakeGenerateEmbed(oldF html.TypeFunc) html.TypeFunc {
	return func(enc *html.Encoder, obj zjson.Object, pos int) (bool, zjson.CloseFunc) {
		src := zjson.GetString(obj, zjson.NameString)
		zid := api.ZettelID(src)
		if !zid.IsValid() {
			return oldF(enc, obj, pos)
		}
		u := g.builder.NewURLBuilder('z').SetZid(zid)
		enc.WriteImage(obj, u.String())
		return false, nil
	}
}

func (g *htmlGenerator) makeGenerateEmbed(oldFn sxpf.BuiltinFn) sxpf.BuiltinFn {
	return func(senv sxpf.Environment, args []sxpf.Value) (sxpf.Value, error) {
		env := senv.(*html.EncEnvironment)
		ref := env.GetList(args, 1)
		refPair := ref.GetValue()
		refValue := env.GetString(refPair, 1)
		zid := api.ZettelID(refValue)
		if !zid.IsValid() {
			return oldFn(senv, args)
		}
		// u := g.builder.NewURLBuilder('z').SetZid(zid)
		// env.WriteImage(obj, u.String())
		return nil, nil
	}
}

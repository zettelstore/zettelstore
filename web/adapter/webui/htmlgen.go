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

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/html"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/sexprenc"
	"zettelstore.de/z/encoder/textenc"
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
	extMarker string
	newWindow bool
	env       *html.EncEnvironment
}

func createGenerator(builder urlBuilder, extMarker string, newWindow bool) *htmlGenerator {
	env := html.NewEncEnvironment(nil, 1)
	gen := &htmlGenerator{
		builder:   builder,
		textEnc:   textenc.Create(),
		extMarker: extMarker,
		newWindow: newWindow,
		env:       env,
	}

	env.Builtins.Set(sexpr.SymTag, sxpf.NewBuiltin("tag", true, 0, -1, gen.generateTag))
	env.Builtins.Set(sexpr.SymLinkZettel, sxpf.NewBuiltin("linkZ", true, 2, -1, gen.generateLinkZettel))
	env.Builtins.Set(sexpr.SymLinkFound, sxpf.NewBuiltin("linkZ", true, 2, -1, gen.generateLinkZettel))
	env.Builtins.Set(sexpr.SymLinkBased, sxpf.NewBuiltin("linkB", true, 2, -1, gen.generateLinkBased))
	env.Builtins.Set(sexpr.SymLinkExternal, sxpf.NewBuiltin("linkE", true, 2, -1, gen.generateLinkExternal))

	f, err := env.Builtins.LookupForm(sexpr.SymEmbed)
	if err != nil {
		panic(err)
	}
	b := f.(*sxpf.Builtin)
	env.Builtins.Set(sexpr.SymEmbed, sxpf.NewBuiltin(b.Name(), true, 3, -1, gen.makeGenerateEmbed(b.GetValue())))
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
	sxpf.Eval(g.env, lst)
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
	return html.EvaluateInline(g.env, sexprenc.GetSexpr(is), !noLink, noLink), nil
}

func (g *htmlGenerator) generateTag(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
	if !sxpf.IsNil(args) {
		env := senv.(*html.EncEnvironment)
		s := env.GetString(args.GetSlice(), 0)
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

func (g *htmlGenerator) generateLinkZettel(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
	env := senv.(*html.EncEnvironment)
	sArgs := args.GetSlice()
	if a, refValue, ok := html.PrepareLink(env, sArgs); ok {
		zid, fragment, hasFragment := strings.Cut(refValue, "#")
		u := g.builder.NewURLBuilder('h').SetZid(api.ZettelID(zid))
		if hasFragment {
			u = u.SetFragment(fragment)
		}
		html.WriteLink(env, sArgs, a.Set("href", u.String()), refValue, "")
	}
	return nil, nil
}

func (g *htmlGenerator) generateLinkBased(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
	env := senv.(*html.EncEnvironment)
	sArgs := args.GetSlice()
	if a, refValue, ok := html.PrepareLink(env, sArgs); ok {
		u := g.builder.NewURLBuilder('/').SetRawLocal(refValue)
		html.WriteLink(env, sArgs, a.Set("href", u.String()), refValue, "")
	}
	return nil, nil
}

func (g *htmlGenerator) generateLinkExternal(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
	env := senv.(*html.EncEnvironment)
	sArgs := args.GetSlice()
	if a, refValue, ok := html.PrepareLink(env, sArgs); ok {
		a = a.Set("href", refValue).
			AddClass("external").
			Set("target", "_blank").
			Set("rel", "noopener noreferrer")
		html.WriteLink(env, sArgs, a, refValue, g.extMarker)
	}
	return nil, nil
}

func (g *htmlGenerator) makeGenerateEmbed(oldFn sxpf.BuiltinFn) sxpf.BuiltinFn {
	return func(senv sxpf.Environment, args *sxpf.Pair, arity int) (sxpf.Value, error) {
		env := senv.(*html.EncEnvironment)
		sArgs := args.GetSlice()
		ref := env.GetPair(sArgs, 1)
		refPair := ref.GetSlice()
		refValue := env.GetString(refPair, 1)
		zid := api.ZettelID(refValue)
		if !zid.IsValid() {
			return oldFn(senv, args, arity)
		}
		u := g.builder.NewURLBuilder('z').SetZid(zid)
		env.WriteImageWithSource(sArgs, u.String())
		return nil, nil
	}
}

//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"io"
	"strings"

	"codeberg.org/t73fde/sxhtml"
	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/attrs"
	"zettelstore.de/c/maps"
	"zettelstore.de/c/shtml"
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
	tx        *sexprenc.Transformer
	th        *shtml.Transformer
	builder   urlBuilder
	textEnc   *textenc.Encoder
	extMarker string
}

func createGenerator(builder urlBuilder, extMarker string) *htmlGenerator {
	gen := &htmlGenerator{
		tx:        sexprenc.NewTransformer(),
		th:        shtml.NewTransformer(1),
		builder:   builder,
		textEnc:   textenc.Create(),
		extMarker: extMarker,
	}

	// env.Builtins.Set(sexpr.SymLinkZettel, sxpf.NewBuiltin("linkZ", true, 2, -1, gen.generateLinkZettel))
	// env.Builtins.Set(sexpr.SymLinkFound, sxpf.NewBuiltin("linkZ", true, 2, -1, gen.generateLinkZettel))
	// env.Builtins.Set(sexpr.SymLinkBased, sxpf.NewBuiltin("linkB", true, 2, -1, gen.generateLinkBased))
	// env.Builtins.Set(sexpr.SymLinkQuery, sxpf.NewBuiltin("linkQ", true, 2, -1, gen.generateLinkQuery))
	// env.Builtins.Set(sexpr.SymLinkExternal, sxpf.NewBuiltin("linkE", true, 2, -1, gen.generateLinkExternal))

	// f, err := env.Builtins.LookupForm(sexpr.SymEmbed)
	// if err != nil {
	// 	panic(err)
	// }
	// b := f.(*sxpf.Builtin)
	// env.Builtins.Set(sexpr.SymEmbed, sxpf.NewBuiltin(b.Name(), true, 3, -1, gen.makeGenerateEmbed(b.GetValue())))
	return gen
}

var mapMetaKey = map[string]string{
	api.KeyCopyright: "copyright",
	api.KeyLicense:   "license",
}

func (g *htmlGenerator) MetaString(m *meta.Meta, evalMeta encoder.EvalMetaFunc) string {
	tm := g.tx.GetMeta(m, evalMeta)
	hm, err := g.th.Transform(tm)
	if err != nil {
		return ""
	}

	ignore := strfun.NewSet(api.KeyTitle, api.KeyLang)
	metaMap := make(map[string]*sxpf.List, m.Length())
	if tags, ok := m.Get(api.KeyTags); ok {
		metaMap[api.KeyTags] = g.transformMetaTags(tags)
		ignore.Set(api.KeyTags)
	}

	for elem := hm; elem != nil; elem = elem.Tail() {
		mlst, ok := elem.Car().(*sxpf.List)
		if !ok {
			continue
		}
		att, ok := mlst.Tail().Car().(*sxpf.List)
		if !ok {
			continue
		}
		if !att.Car().IsEqual(g.th.Make("@")) {
			continue
		}
		a := make(attrs.Attributes, 32)
		for aelem := att.Tail(); aelem != nil; aelem = aelem.Tail() {
			if p, ok2 := aelem.Car().(*sxpf.List); ok2 {
				key := p.Car()
				val := p.Cdr()
				if tail, ok3 := val.(*sxpf.List); ok3 {
					val = tail.Car()
				}
				a = a.Set(key.String(), val.String())
			}
		}
		name, found := a.Get("name")
		if !found || ignore.Has(name) {
			continue
		}
		var newName string
		if altName, found2 := mapMetaKey[name]; found2 {
			newName = altName
		} else {
			newName = "zs-" + name
		}
		a = a.Set("name", newName)
		metaMap[newName] = g.th.TransformMeta(a)
	}
	result := sxpf.Nil()
	keys := maps.Keys(metaMap)
	for i := len(keys) - 1; i >= 0; i-- {
		result = result.Cons(metaMap[keys[i]])
	}

	var sb strings.Builder
	_ = generateHTML(&sb, result)
	return sb.String()
}
func (g *htmlGenerator) transformMetaTags(tags string) *sxpf.List {
	var sb strings.Builder
	for i, val := range meta.ListFromValue(tags) {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strings.TrimPrefix(val, "#"))
	}
	metaTags := sb.String()
	if len(metaTags) == 0 {
		return sxpf.Nil()
	}
	return g.th.TransformMeta(attrs.Attributes{"name": "keywords", "content": metaTags})
}

// BlocksString encodes a block slice.
func (g *htmlGenerator) BlocksString(bs *ast.BlockSlice) (string, error) {
	if bs == nil || len(*bs) == 0 {
		return "", nil
	}
	sx := g.tx.GetSexpr(bs)
	sh, err := g.th.Transform(sx)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	err = generateHTML(&sb, sh)
	if err != nil {
		return sb.String(), err
	}

	// WriteEndnotes
	return sb.String(), nil
}

// InlinesString writes an inline slice to the writer
func (g *htmlGenerator) InlinesString(is *ast.InlineSlice) string {
	if is == nil || len(*is) == 0 {
		return ""
	}
	sx := g.tx.GetSexpr(is)
	sh, err := g.th.Transform(sx)
	if err != nil {
		return ""
	}
	var sb strings.Builder
	_ = generateHTML(&sb, sh)
	return sb.String()
}

func generateHTML(w io.Writer, hval *sxpf.List) error {
	gen := sxhtml.NewGenerator(sxpf.FindSymbolFactory(hval))
	for elem := hval; elem != nil; elem = elem.Tail() {
		_, err := gen.WriteHTML(w, elem.Car())
		if err != nil {
			return err
		}
	}
	return nil
}

// func (g *htmlGenerator) generateLinkZettel(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
// 	env := senv.(*html.EncEnvironment)
// 	if a, refValue, ok := html.PrepareLink(env, args); ok {
// 		zid, fragment, hasFragment := strings.Cut(refValue, "#")
// 		u := g.builder.NewURLBuilder('h').SetZid(api.ZettelID(zid))
// 		if hasFragment {
// 			u = u.SetFragment(fragment)
// 		}
// 		html.WriteLink(env, args, a.Set("href", u.String()), refValue, "")
// 	}
// 	return nil, nil
// }

// func (g *htmlGenerator) generateLinkBased(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
// 	env := senv.(*html.EncEnvironment)
// 	if a, refValue, ok := html.PrepareLink(env, args); ok {
// 		u := g.builder.NewURLBuilder('/').SetRawLocal(refValue)
// 		html.WriteLink(env, args, a.Set("href", u.String()), refValue, "")
// 	}
// 	return nil, nil
// }

// func (g *htmlGenerator) generateLinkQuery(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
// 	env := senv.(*html.EncEnvironment)
// 	if a, refValue, ok := html.PrepareLink(env, args); ok {
// 		queryExpr := query.Parse(refValue).String()
// 		u := g.builder.NewURLBuilder('h').AppendQuery(queryExpr)
// 		html.WriteLink(env, args, a.Set("href", u.String()), refValue, "")
// 	}
// 	return nil, nil
// }

// func (g *htmlGenerator) generateLinkExternal(senv sxpf.Environment, args *sxpf.Pair, _ int) (sxpf.Value, error) {
// 	env := senv.(*html.EncEnvironment)
// 	if a, refValue, ok := html.PrepareLink(env, args); ok {
// 		a = a.Set("href", refValue).
// 			AddClass("external").
// 			Set("target", "_blank").
// 			Set("rel", "noopener noreferrer")
// 		html.WriteLink(env, args, a, refValue, g.extMarker)
// 	}
// 	return nil, nil
// }

// func (g *htmlGenerator) makeGenerateEmbed(oldFn sxpf.BuiltinFn) sxpf.BuiltinFn {
// 	return func(senv sxpf.Environment, args *sxpf.Pair, arity int) (sxpf.Value, error) {
// 		env := senv.(*html.EncEnvironment)
// 		ref := env.GetPair(args.GetTail())
// 		refValue := env.GetString(ref.GetTail())
// 		zid := api.ZettelID(refValue)
// 		if !zid.IsValid() {
// 			return oldFn(senv, args, arity)
// 		}
// 		u := g.builder.NewURLBuilder('z').SetZid(zid)
// 		env.WriteImageWithSource(args, u.String())
// 		return nil, nil
// 	}
// }

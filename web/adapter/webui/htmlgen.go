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
	"net/url"
	"strings"

	"codeberg.org/t73fde/sxhtml"
	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/attrs"
	"zettelstore.de/c/maps"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/c/shtml"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/sexprenc"
	"zettelstore.de/z/strfun"
)

// Builder allows to build new URLs for the web service.
type urlBuilder interface {
	GetURLPrefix() string
	NewURLBuilder(key byte) *api.URLBuilder
}

type htmlGenerator struct {
	tx    *sexprenc.Transformer
	th    *shtml.Transformer
	symAt *sxpf.Symbol
}

func (wui *WebUI) createGenerator(builder urlBuilder) *htmlGenerator {
	th := shtml.NewTransformer(1, wui.sf)
	symA := th.Make("a")
	symImg := th.Make("img")
	symAt := th.Make(sxhtml.NameSymAttr)

	symHref := th.Make("href")
	symClass := th.Make("class")
	symTarget := th.Make("target")
	symRel := th.Make("rel")

	findA := func(obj sxpf.Object) (attr, assoc, rest *sxpf.List) {
		lst, ok := obj.(*sxpf.List)
		if !ok || !symA.IsEqual(lst.Car()) {
			return nil, nil, nil
		}
		rest = lst.Tail()
		if rest == nil {
			return nil, nil, nil
		}
		objA := rest.Car()
		attr, ok = objA.(*sxpf.List)
		if !ok || !symAt.IsEqual(attr.Car()) {
			return nil, nil, nil
		}
		return attr, attr.Tail(), rest.Tail()
	}
	linkZettel := func(_ sxpf.Environment, args *sxpf.List, prevFn sxpf.Callable) sxpf.Object {
		obj, err := prevFn.Call(nil, args)
		if err != nil {
			return sxpf.Nil()
		}
		attr, assoc, rest := findA(obj)
		if attr == nil {
			return obj
		}

		hrefP := assoc.Assoc(symHref)
		if hrefP == nil {
			return obj
		}
		href, ok := hrefP.Cdr().(sxpf.String)
		if !ok {
			return obj
		}
		zid, fragment, hasFragment := strings.Cut(href.String(), "#")
		u := builder.NewURLBuilder('h').SetZid(api.ZettelID(zid))
		if hasFragment {
			u = u.SetFragment(fragment)
		}
		assoc = assoc.Cons(sxpf.Cons(symHref, sxpf.MakeString(u.String())))
		return rest.Cons(assoc.Cons(symAt)).Cons(symA)
	}

	th.SetRebinder(func(te *shtml.TransformEnv) {
		te.Rebind(sexpr.NameSymLinkZettel, linkZettel)
		te.Rebind(sexpr.NameSymLinkFound, linkZettel)
		te.Rebind(sexpr.NameSymLinkBased, func(_ sxpf.Environment, args *sxpf.List, prevFn sxpf.Callable) sxpf.Object {
			obj, err := prevFn.Call(nil, args)
			if err != nil {
				return sxpf.Nil()
			}
			attr, assoc, rest := findA(obj)
			if attr == nil {
				return obj
			}
			hrefP := assoc.Assoc(symHref)
			if hrefP == nil {
				return obj
			}
			href, ok := hrefP.Cdr().(sxpf.String)
			if !ok {
				return obj
			}
			u := builder.NewURLBuilder('/').SetRawLocal(href.String())
			assoc = assoc.Cons(sxpf.Cons(symHref, sxpf.MakeString(u.String())))
			return rest.Cons(assoc.Cons(symAt)).Cons(symA)
		})
		te.Rebind(sexpr.NameSymLinkQuery, func(_ sxpf.Environment, args *sxpf.List, prevFn sxpf.Callable) sxpf.Object {
			obj, err := prevFn.Call(nil, args)
			if err != nil {
				return sxpf.Nil()
			}
			attr, assoc, rest := findA(obj)
			if attr == nil {
				return obj
			}
			hrefP := assoc.Assoc(symHref)
			if hrefP == nil {
				return obj
			}
			href, ok := hrefP.Cdr().(sxpf.String)
			if !ok {
				return obj
			}
			ur, err := url.Parse(href.String())
			if err != nil {
				return obj
			}
			q := ur.Query().Get(api.QueryKeyQuery)
			if q == "" {
				return obj
			}
			u := builder.NewURLBuilder('h').AppendQuery(q)
			assoc = assoc.Cons(sxpf.Cons(symHref, sxpf.MakeString(u.String())))
			return rest.Cons(assoc.Cons(symAt)).Cons(symA)
		})
		te.Rebind(sexpr.NameSymLinkExternal, func(_ sxpf.Environment, args *sxpf.List, prevFn sxpf.Callable) sxpf.Object {
			obj, err := prevFn.Call(nil, args)
			if err != nil {
				return sxpf.Nil()
			}
			attr, assoc, rest := findA(obj)
			if attr == nil {
				return obj
			}
			assoc = assoc.Cons(sxpf.Cons(symClass, sxpf.MakeString("external"))).
				Cons(sxpf.Cons(symTarget, sxpf.MakeString("_blank"))).
				Cons(sxpf.Cons(symRel, sxpf.MakeString("noopener noreferrer")))
			return rest.Cons(assoc.Cons(symAt)).Cons(symA)
		})
		te.Rebind(sexpr.NameSymEmbed, func(_ sxpf.Environment, args *sxpf.List, prevFn sxpf.Callable) sxpf.Object {
			obj, err := prevFn.Call(nil, args)
			if err != nil {
				return sxpf.Nil()
			}
			lst, ok := obj.(*sxpf.List)
			if !ok || !symImg.IsEqual(lst.Car()) {
				return obj
			}
			attr, ok := lst.Tail().Car().(*sxpf.List)
			if !ok || !symAt.IsEqual(attr.Car()) {
				return obj
			}
			symSrc := th.Make("src")
			srcP := attr.Tail().Assoc(symSrc)
			if srcP == nil {
				return obj
			}
			src, ok := srcP.Cdr().(sxpf.String)
			if !ok {
				return obj
			}
			zid := api.ZettelID(src)
			if !zid.IsValid() {
				return obj
			}
			u := builder.NewURLBuilder('z').SetZid(zid)
			imgAttr := attr.Tail().Cons(sxpf.Cons(symSrc, sxpf.MakeString(u.String()))).Cons(symAt)
			return lst.Tail().Tail().Cons(imgAttr).Cons(symImg)
		})
	})

	return &htmlGenerator{
		tx:    sexprenc.NewTransformer(),
		th:    th,
		symAt: symAt,
	}
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
		if !att.Car().IsEqual(g.symAt) {
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
	gen := sxhtml.NewGenerator(sxpf.FindSymbolFactory(result))
	_, _ = gen.WriteListHTML(&sb, result)
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
	gen := sxhtml.NewGenerator(sxpf.FindSymbolFactory(sh), sxhtml.WithNewline)
	_, err = gen.WriteListHTML(&sb, sh)
	if err != nil {
		return sb.String(), err
	}

	_, err = gen.WriteHTML(&sb, g.th.Endnotes())
	return sb.String(), err
}

// InlinesSxHTML returns an inline slice, encoded as a SxHTML object.
func (g *htmlGenerator) InlinesSxHTML(is *ast.InlineSlice) *sxpf.List {
	if is == nil || len(*is) == 0 {
		return sxpf.Nil()
	}
	sx := g.tx.GetSexpr(is)
	sh, err := g.th.Transform(sx)
	if err != nil {
		return sxpf.Nil()
	}
	return sh
}

// InlinesString returns an inline slice, encoded as a HTML string.
func (g *htmlGenerator) InlinesString(is *ast.InlineSlice) string {
	if sh := g.InlinesSxHTML(is); !sxpf.IsNil(sh) {
		var sb strings.Builder
		gen := sxhtml.NewGenerator(sxpf.FindSymbolFactory(sh))
		_, _ = gen.WriteListHTML(&sb, sh)
		return sb.String()
	}
	return ""
}

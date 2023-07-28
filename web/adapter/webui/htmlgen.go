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

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/client.fossil/attrs"
	"zettelstore.de/client.fossil/maps"
	"zettelstore.de/client.fossil/shtml"
	"zettelstore.de/client.fossil/sz"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/sx.fossil/sxeval"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/szenc"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/zettel/meta"
)

// Builder allows to build new URLs for the web service.
type urlBuilder interface {
	GetURLPrefix() string
	NewURLBuilder(key byte) *api.URLBuilder
}

type htmlGenerator struct {
	tx    *szenc.Transformer
	th    *shtml.Transformer
	symAt *sx.Symbol
}

func (wui *WebUI) createGenerator(builder urlBuilder) *htmlGenerator {
	th := shtml.NewTransformer(1, wui.sf)
	symA := wui.symA
	symImg := th.Make("img")
	symAttr := wui.symAttr

	symHref := wui.symHref
	symClass := th.Make("class")
	symTarget := th.Make("target")
	symRel := th.Make("rel")

	findA := func(obj sx.Object) (attr, assoc, rest *sx.Pair) {
		pair, isPair := sx.GetPair(obj)
		if !isPair || !symA.IsEqual(pair.Car()) {
			return nil, nil, nil
		}
		rest = pair.Tail()
		if rest == nil {
			return nil, nil, nil
		}
		objA := rest.Car()
		attr, isPair = sx.GetPair(objA)
		if !isPair || !symAttr.IsEqual(attr.Car()) {
			return nil, nil, nil
		}
		return attr, attr.Tail(), rest.Tail()
	}
	linkZettel := func(args []sx.Object, prevFn sxeval.Callable) sx.Object {
		obj, err := prevFn.Call(nil, nil, args)
		if err != nil {
			return sx.Nil()
		}
		attr, assoc, rest := findA(obj)
		if attr == nil {
			return obj
		}

		hrefP := assoc.Assoc(symHref)
		if hrefP == nil {
			return obj
		}
		href, ok := sx.GetString(hrefP.Cdr())
		if !ok {
			return obj
		}
		zid, fragment, hasFragment := strings.Cut(href.String(), "#")
		u := builder.NewURLBuilder('h').SetZid(api.ZettelID(zid))
		if hasFragment {
			u = u.SetFragment(fragment)
		}
		assoc = assoc.Cons(sx.Cons(symHref, sx.MakeString(u.String())))
		return rest.Cons(assoc.Cons(symAttr)).Cons(symA)
	}

	th.SetRebinder(func(te *shtml.TransformEnv) {
		te.Rebind(sz.NameSymLinkZettel, linkZettel)
		te.Rebind(sz.NameSymLinkFound, linkZettel)
		te.Rebind(sz.NameSymLinkBased, func(args []sx.Object, prevFn sxeval.Callable) sx.Object {
			obj, err := prevFn.Call(nil, nil, args)
			if err != nil {
				return sx.Nil()
			}
			attr, assoc, rest := findA(obj)
			if attr == nil {
				return obj
			}
			hrefP := assoc.Assoc(symHref)
			if hrefP == nil {
				return obj
			}
			href, ok := sx.GetString(hrefP.Cdr())
			if !ok {
				return obj
			}
			u := builder.NewURLBuilder('/').SetRawLocal(href.String())
			assoc = assoc.Cons(sx.Cons(symHref, sx.MakeString(u.String())))
			return rest.Cons(assoc.Cons(symAttr)).Cons(symA)
		})
		te.Rebind(sz.NameSymLinkQuery, func(args []sx.Object, prevFn sxeval.Callable) sx.Object {
			obj, err := prevFn.Call(nil, nil, args)
			if err != nil {
				return sx.Nil()
			}
			attr, assoc, rest := findA(obj)
			if attr == nil {
				return obj
			}
			hrefP := assoc.Assoc(symHref)
			if hrefP == nil {
				return obj
			}
			href, ok := sx.GetString(hrefP.Cdr())
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
			assoc = assoc.Cons(sx.Cons(symHref, sx.MakeString(u.String())))
			return rest.Cons(assoc.Cons(symAttr)).Cons(symA)
		})
		te.Rebind(sz.NameSymLinkExternal, func(args []sx.Object, prevFn sxeval.Callable) sx.Object {
			obj, err := prevFn.Call(nil, nil, args)
			if err != nil {
				return sx.Nil()
			}
			attr, assoc, rest := findA(obj)
			if attr == nil {
				return obj
			}
			assoc = assoc.Cons(sx.Cons(symClass, sx.MakeString("external"))).
				Cons(sx.Cons(symTarget, sx.MakeString("_blank"))).
				Cons(sx.Cons(symRel, sx.MakeString("noopener noreferrer")))
			return rest.Cons(assoc.Cons(symAttr)).Cons(symA)
		})
		te.Rebind(sz.NameSymEmbed, func(args []sx.Object, prevFn sxeval.Callable) sx.Object {
			obj, err := prevFn.Call(nil, nil, args)
			if err != nil {
				return sx.Nil()
			}
			pair, isPair := sx.GetPair(obj)
			if !isPair || !symImg.IsEqual(pair.Car()) {
				return obj
			}
			attr, isPair := sx.GetPair(pair.Tail().Car())
			if !isPair || !symAttr.IsEqual(attr.Car()) {
				return obj
			}
			symSrc := th.Make("src")
			srcP := attr.Tail().Assoc(symSrc)
			if srcP == nil {
				return obj
			}
			src, isString := sx.GetString(srcP.Cdr())
			if !isString {
				return obj
			}
			zid := api.ZettelID(src)
			if !zid.IsValid() {
				return obj
			}
			u := builder.NewURLBuilder('z').SetZid(zid)
			imgAttr := attr.Tail().Cons(sx.Cons(symSrc, sx.MakeString(u.String()))).Cons(symAttr)
			return pair.Tail().Tail().Cons(imgAttr).Cons(symImg)
		})
	})

	return &htmlGenerator{
		tx:    szenc.NewTransformer(),
		th:    th,
		symAt: symAttr,
	}
}

// SetUnique sets a prefix to make several HTML ids unique.
func (g *htmlGenerator) SetUnique(s string) *htmlGenerator { g.th.SetUnique(s); return g }

var mapMetaKey = map[string]string{
	api.KeyCopyright: "copyright",
	api.KeyLicense:   "license",
}

func (g *htmlGenerator) MetaSxn(m *meta.Meta, evalMeta encoder.EvalMetaFunc) *sx.Pair {
	tm := g.tx.GetMeta(m, evalMeta)
	hm, err := g.th.Transform(tm)
	if err != nil {
		return nil
	}

	ignore := strfun.NewSet(api.KeyTitle, api.KeyLang)
	metaMap := make(map[string]*sx.Pair, m.Length())
	if tags, ok := m.Get(api.KeyTags); ok {
		metaMap[api.KeyTags] = g.transformMetaTags(tags)
		ignore.Set(api.KeyTags)
	}

	for elem := hm; elem != nil; elem = elem.Tail() {
		mlst, isPair := sx.GetPair(elem.Car())
		if !isPair {
			continue
		}
		att, isPair := sx.GetPair(mlst.Tail().Car())
		if !isPair {
			continue
		}
		if !att.Car().IsEqual(g.symAt) {
			continue
		}
		a := make(attrs.Attributes, 32)
		for aelem := att.Tail(); aelem != nil; aelem = aelem.Tail() {
			if p, ok := sx.GetPair(aelem.Car()); ok {
				key := p.Car()
				val := p.Cdr()
				if tail, isTail := sx.GetPair(val); isTail {
					val = tail.Car()
				}
				a = a.Set(key.String(), val.String())
			}
		}
		name, found := a.Get("name")
		if !found || ignore.Has(name) {
			continue
		}

		newName, found := mapMetaKey[name]
		if !found {
			continue
		}
		a = a.Set("name", newName)
		metaMap[newName] = g.th.TransformMeta(a)
	}
	result := sx.Nil()
	keys := maps.Keys(metaMap)
	for i := len(keys) - 1; i >= 0; i-- {
		result = result.Cons(metaMap[keys[i]])
	}
	return result
}

func (g *htmlGenerator) transformMetaTags(tags string) *sx.Pair {
	var sb strings.Builder
	for i, val := range meta.ListFromValue(tags) {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strings.TrimPrefix(val, "#"))
	}
	metaTags := sb.String()
	if len(metaTags) == 0 {
		return nil
	}
	return g.th.TransformMeta(attrs.Attributes{"name": "keywords", "content": metaTags})
}

func (g *htmlGenerator) BlocksSxn(bs *ast.BlockSlice) (content, endnotes *sx.Pair, _ error) {
	if bs == nil || len(*bs) == 0 {
		return nil, nil, nil
	}
	sx := g.tx.GetSz(bs)
	sh, err := g.th.Transform(sx)
	if err != nil {
		return nil, nil, err
	}
	return sh, g.th.Endnotes(), nil
}

// InlinesSxHTML returns an inline slice, encoded as a SxHTML object.
func (g *htmlGenerator) InlinesSxHTML(is *ast.InlineSlice) *sx.Pair {
	if is == nil || len(*is) == 0 {
		return nil
	}
	sx := g.tx.GetSz(is)
	sh, err := g.th.Transform(sx)
	if err != nil {
		return nil
	}
	return sh
}

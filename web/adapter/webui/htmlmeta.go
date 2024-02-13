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

package webui

import (
	"context"
	"errors"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/client.fossil/shtml"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/sx.fossil/sxhtml"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func (wui *WebUI) writeHTMLMetaValue(
	key, value string,
	getTextTitle getTextTitleFunc,
	evalMetadata evalMetadataFunc,
	gen *htmlGenerator,
) sx.Object {
	switch kt := meta.Type(key); kt {
	case meta.TypeCredential:
		return sx.String(value)
	case meta.TypeEmpty:
		return sx.String(value)
	case meta.TypeID:
		return wui.transformIdentifier(value, getTextTitle)
	case meta.TypeIDSet:
		return wui.transformIdentifierSet(meta.ListFromValue(value), getTextTitle)
	case meta.TypeNumber:
		return wui.transformLink(key, value, value)
	case meta.TypeString:
		return sx.String(value)
	case meta.TypeTagSet:
		return wui.transformTagSet(key, meta.ListFromValue(value))
	case meta.TypeTimestamp:
		if ts, ok := meta.TimeValue(value); ok {
			return sx.MakeList(
				sx.MakeSymbol("time"),
				sx.MakeList(
					sxhtml.SymAttr,
					sx.Cons(sx.MakeSymbol("datetime"), sx.String(ts.Format("2006-01-02T15:04:05"))),
				),
				sx.MakeList(sxhtml.SymNoEscape, sx.String(ts.Format("2006-01-02&nbsp;15:04:05"))),
			)
		}
		return sx.Nil()
	case meta.TypeURL:
		return wui.url2html(sx.String(value))
	case meta.TypeWord:
		return wui.transformLink(key, value, value)
	case meta.TypeWordSet:
		return wui.transformWordSet(key, meta.ListFromValue(value))
	case meta.TypeZettelmarkup:
		return wui.transformZmkMetadata(value, evalMetadata, gen)
	default:
		return sx.MakeList(shtml.SymSTRONG, sx.String("Unhandled type: "), sx.String(kt.Name))
	}
}

func (wui *WebUI) transformIdentifier(val string, getTextTitle getTextTitleFunc) sx.Object {
	text := sx.String(val)
	zid, err := id.Parse(val)
	if err != nil {
		return text
	}
	title, found := getTextTitle(zid)
	switch {
	case found > 0:
		ub := wui.NewURLBuilder('h').SetZid(zid.ZettelID())
		attrs := sx.Nil()
		if title != "" {
			attrs = attrs.Cons(sx.Cons(shtml.SymAttrTitle, sx.String(title)))
		}
		attrs = attrs.Cons(sx.Cons(shtml.SymAttrHref, sx.String(ub.String()))).Cons(sxhtml.SymAttr)
		return sx.Nil().Cons(sx.String(zid.String())).Cons(attrs).Cons(shtml.SymA)
	case found == 0:
		return sx.MakeList(sx.MakeSymbol("s"), text)
	default: // case found < 0:
		return text
	}
}

func (wui *WebUI) transformIdentifierSet(vals []string, getTextTitle getTextTitleFunc) *sx.Pair {
	if len(vals) == 0 {
		return nil
	}
	space := sx.String(" ")
	text := make(sx.Vector, 0, 2*len(vals))
	for _, val := range vals {
		text = append(text, space, wui.transformIdentifier(val, getTextTitle))
	}
	return sx.MakeList(text[1:]...).Cons(shtml.SymSPAN)
}

func (wui *WebUI) transformTagSet(key string, tags []string) *sx.Pair {
	if len(tags) == 0 {
		return nil
	}
	space := sx.String(" ")
	text := make(sx.Vector, 0, 2*len(tags))
	for _, tag := range tags {
		text = append(text, space, wui.transformLink(key, tag, tag))
	}
	return sx.MakeList(text[1:]...).Cons(shtml.SymSPAN)
}

func (wui *WebUI) transformWordSet(key string, words []string) sx.Object {
	if len(words) == 0 {
		return sx.Nil()
	}
	space := sx.String(" ")
	text := make(sx.Vector, 0, 2*len(words))
	for _, word := range words {
		text = append(text, space, wui.transformLink(key, word, word))
	}
	return sx.MakeList(text[1:]...).Cons(shtml.SymSPAN)
}

func (wui *WebUI) transformLink(key, value, text string) *sx.Pair {
	return sx.MakeList(
		shtml.SymA,
		sx.MakeList(
			sxhtml.SymAttr,
			sx.Cons(shtml.SymAttrHref, sx.String(wui.NewURLBuilder('h').AppendQuery(key+api.SearchOperatorHas+value).String())),
		),
		sx.String(text),
	)
}

type evalMetadataFunc = func(string) ast.InlineSlice

func createEvalMetadataFunc(ctx context.Context, evaluate *usecase.Evaluate) evalMetadataFunc {
	return func(value string) ast.InlineSlice { return evaluate.RunMetadata(ctx, value) }
}

type getTextTitleFunc func(id.Zid) (string, int)

func (wui *WebUI) makeGetTextTitle(ctx context.Context, getZettel usecase.GetZettel) getTextTitleFunc {
	return func(zid id.Zid) (string, int) {
		z, err := getZettel.Run(box.NoEnrichContext(ctx), zid)
		if err != nil {
			if errors.Is(err, &box.ErrNotAllowed{}) {
				return "", -1
			}
			return "", 0
		}
		return parser.NormalizedSpacedText(z.Meta.GetTitle()), 1
	}
}

func (wui *WebUI) transformZmkMetadata(value string, evalMetadata evalMetadataFunc, gen *htmlGenerator) sx.Object {
	is := evalMetadata(value)
	return gen.InlinesSxHTML(&is).Cons(shtml.SymSPAN)
}

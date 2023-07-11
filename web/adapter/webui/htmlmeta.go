//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"context"
	"errors"

	"codeberg.org/t73fde/sxhtml"
	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
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
) sxpf.Object {
	switch kt := meta.Type(key); kt {
	case meta.TypeCredential:
		return sxpf.MakeString(value)
	case meta.TypeEmpty:
		return sxpf.MakeString(value)
	case meta.TypeID:
		return wui.transformIdentifier(value, getTextTitle)
	case meta.TypeIDSet:
		return wui.transformIdentifierSet(meta.ListFromValue(value), getTextTitle)
	case meta.TypeNumber:
		return wui.transformLink(key, value, value)
	case meta.TypeString:
		return sxpf.MakeString(value)
	case meta.TypeTagSet:
		return wui.transformTagSet(key, meta.ListFromValue(value))
	case meta.TypeTimestamp:
		if ts, ok := meta.TimeValue(value); ok {
			return sxpf.MakeList(
				wui.sf.MustMake("time"),
				sxpf.MakeList(
					wui.symAttr,
					sxpf.Cons(wui.sf.MustMake("datetime"), sxpf.MakeString(ts.Format("2006-01-02T15:04:05"))),
				),
				sxpf.MakeList(wui.sf.MustMake(sxhtml.NameSymNoEscape), sxpf.MakeString(ts.Format("2006-01-02&nbsp;15:04:05"))),
			)
		}
		return sxpf.Nil()
	case meta.TypeURL:
		text := sxpf.MakeString(value)
		if res, err := wui.url2html([]sxpf.Object{text}); err == nil {
			return res
		}
		return text
	case meta.TypeWord:
		return wui.transformLink(key, value, value)
	case meta.TypeWordSet:
		return wui.transformWordSet(key, meta.ListFromValue(value))
	case meta.TypeZettelmarkup:
		return wui.transformZmkMetadata(value, evalMetadata, gen)
	default:
		return sxpf.MakeList(wui.sf.MustMake("b"), sxpf.MakeString("Unhandled type: "), sxpf.MakeString(kt.Name))
	}
}

func (wui *WebUI) transformIdentifier(val string, getTextTitle getTextTitleFunc) sxpf.Object {
	text := sxpf.MakeString(val)
	zid, err := id.Parse(val)
	if err != nil {
		return text
	}
	title, found := getTextTitle(zid)
	switch {
	case found > 0:
		ub := wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String()))
		attrs := sxpf.Nil()
		if title != "" {
			attrs = attrs.Cons(sxpf.Cons(wui.sf.MustMake("title"), sxpf.MakeString(title)))
		}
		attrs = attrs.Cons(sxpf.Cons(wui.symHref, sxpf.MakeString(ub.String()))).Cons(wui.symAttr)
		return sxpf.Nil().Cons(sxpf.MakeString(zid.String())).Cons(attrs).Cons(wui.symA)
	case found == 0:
		return sxpf.MakeList(wui.sf.MustMake("s"), text)
	default: // case found < 0:
		return text
	}
}

func (wui *WebUI) transformIdentifierSet(vals []string, getTextTitle getTextTitleFunc) *sxpf.Pair {
	if len(vals) == 0 {
		return nil
	}
	space := sxpf.MakeString(" ")
	text := make([]sxpf.Object, 0, 2*len(vals))
	for _, val := range vals {
		text = append(text, space, wui.transformIdentifier(val, getTextTitle))
	}
	return sxpf.MakeList(text[1:]...).Cons(wui.symSpan)
}

func (wui *WebUI) transformTagSet(key string, tags []string) *sxpf.Pair {
	if len(tags) == 0 {
		return nil
	}
	space := sxpf.MakeString(" ")
	text := make([]sxpf.Object, 0, 2*len(tags))
	for _, tag := range tags {
		text = append(text, space, wui.transformLink(key, tag, tag))
	}
	return sxpf.MakeList(text[1:]...).Cons(wui.symSpan)
}

func (wui *WebUI) transformWordSet(key string, words []string) sxpf.Object {
	if len(words) == 0 {
		return sxpf.Nil()
	}
	space := sxpf.MakeString(" ")
	text := make([]sxpf.Object, 0, 2*len(words))
	for _, word := range words {
		text = append(text, space, wui.transformLink(key, word, word))
	}
	return sxpf.MakeList(text[1:]...).Cons(wui.symSpan)
}

func (wui *WebUI) transformLink(key, value, text string) *sxpf.Pair {
	return sxpf.MakeList(
		wui.symA,
		sxpf.MakeList(
			wui.symAttr,
			sxpf.Cons(wui.symHref, sxpf.MakeString(wui.NewURLBuilder('h').AppendQuery(key+api.SearchOperatorHas+value).String())),
		),
		sxpf.MakeString(text),
	)
}

type evalMetadataFunc = func(string) ast.InlineSlice

func createEvalMetadataFunc(ctx context.Context, evaluate *usecase.Evaluate) evalMetadataFunc {
	return func(value string) ast.InlineSlice { return evaluate.RunMetadata(ctx, value) }
}

type getTextTitleFunc func(id.Zid) (string, int)

func (wui *WebUI) makeGetTextTitle(ctx context.Context, getMeta usecase.GetMeta) getTextTitleFunc {
	return func(zid id.Zid) (string, int) {
		m, err := getMeta.Run(box.NoEnrichContext(ctx), zid)
		if err != nil {
			if errors.Is(err, &box.ErrNotAllowed{}) {
				return "", -1
			}
			return "", 0
		}
		return parser.NormalizedSpacedText(m.GetTitle()), 1
	}
}

func (wui *WebUI) transformZmkMetadata(value string, evalMetadata evalMetadataFunc, gen *htmlGenerator) sxpf.Object {
	is := evalMetadata(value)
	return gen.InlinesSxHTML(&is).Cons(wui.symSpan)
}

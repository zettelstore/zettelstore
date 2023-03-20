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
	"fmt"
	"net/url"
	"time"

	"codeberg.org/t73fde/sxhtml"
	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
)

func (wui *WebUI) writeHTMLMetaValue(
	key, value string,
	getTextTitle getTextTitleFunc,
	evalMetadata evalMetadataFunc,
	gen *htmlGenerator,
) sxpf.Object {
	var sval sxpf.Object = sxpf.Nil()
	switch kt := meta.Type(key); kt {
	case meta.TypeCredential:
		sval = sxpf.MakeString(value)
	case meta.TypeEmpty:
		sval = sxpf.MakeString(value)
	case meta.TypeID:
		sval = wui.transformIdentifier(value, getTextTitle)
	case meta.TypeIDSet:
		sval = wui.transformIdentifierSet(meta.ListFromValue(value), getTextTitle)
	case meta.TypeNumber:
		sval = wui.transformLink(key, value, value)
	case meta.TypeString:
		sval = sxpf.MakeString(value)
	case meta.TypeTagSet:
		sval = wui.transformTagSet(key, meta.ListFromValue(value))
	case meta.TypeTimestamp:
		if ts, ok := meta.TimeValue(value); ok {
			sval = wui.transformTimestamp(ts)
		}
	case meta.TypeURL:
		sval = wui.transformURL(value)
	case meta.TypeWord:
		sval = wui.transformWord(key, value)
	case meta.TypeWordSet:
		sval = wui.transformWordSet(key, meta.ListFromValue(value))
	case meta.TypeZettelmarkup:
		sval = wui.transformZmkMetadata(value, evalMetadata, gen)
	default:
		sval = sxpf.Nil().Cons(sxpf.MakeString(fmt.Sprintf(" <b>(Unhandled type: %v, key: %v)</b>", kt, key))).Cons(wui.sf.MustMake("b"))
	}
	return sval
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
		attrs = attrs.Cons(sxpf.Cons(wui.sf.MustMake("href"), sxpf.MakeString(ub.String()))).Cons(wui.sf.MustMake(sxhtml.NameSymAttr))
		return sxpf.Nil().Cons(sxpf.MakeString(zid.String())).Cons(attrs).Cons(wui.sf.MustMake("a"))
	case found == 0:
		return sxpf.Nil().Cons(text).Cons(wui.sf.MustMake("s"))
	default: // case found < 0:
		return text
	}
}

func (wui *WebUI) transformIdentifierSet(vals []string, getTextTitle getTextTitleFunc) sxpf.Object {
	if len(vals) == 0 {
		return sxpf.Nil()
	}
	space := sxpf.MakeString(" ")
	text := make([]sxpf.Object, 0, 2*len(vals))
	for _, val := range vals {
		text = append(text, space, wui.transformIdentifier(val, getTextTitle))
	}
	return sxpf.MakeList(text[1:]...).Cons(wui.sf.MustMake("span"))
}

func (wui *WebUI) transformTagSet(key string, tags []string) sxpf.Object {
	if len(tags) == 0 {
		return sxpf.Nil()
	}
	space := sxpf.MakeString(" ")
	text := make([]sxpf.Object, 0, 2*len(tags))
	for _, tag := range tags {
		text = append(text, space, wui.transformLink(key, tag, tag))
	}
	return sxpf.MakeList(text[1:]...).Cons(wui.sf.MustMake("span"))
}

func (wui *WebUI) transformTimestamp(ts time.Time) sxpf.Object {
	return sxpf.MakeList(
		wui.sf.MustMake("time"),
		sxpf.MakeList(
			wui.sf.MustMake(sxhtml.NameSymAttr),
			sxpf.Cons(wui.sf.MustMake("datetime"), sxpf.MakeString(ts.Format("2006-01-02T15:04:05"))),
		),
		sxpf.MakeList(wui.sf.MustMake(sxhtml.NameSymNoEscape), sxpf.MakeString(ts.Format("2006-01-02&nbsp;15:04:05"))),
	)
}

func (wui *WebUI) transformURL(val string) sxpf.Object {
	text := sxpf.MakeString(val)
	u, err := url.Parse(val)
	if err == nil {
		if us := u.String(); us != "" {
			return sxpf.MakeList(
				wui.sf.MustMake("a"),
				sxpf.MakeList(
					wui.sf.MustMake(sxhtml.NameSymAttr),
					sxpf.Cons(wui.sf.MustMake("href"), sxpf.MakeString(val)),
					sxpf.Cons(wui.sf.MustMake("target"), sxpf.MakeString("_blank")),
					sxpf.Cons(wui.sf.MustMake("rel"), sxpf.MakeString("noopener noreferrer")),
				),
				text,
			)
		}
	}
	return text
}

func (wui *WebUI) transformWord(key, word string) sxpf.Object {
	return wui.transformLink(key, word, word)
}

func (wui *WebUI) transformWordSet(key string, words []string) sxpf.Object {
	if len(words) == 0 {
		return sxpf.Nil()
	}
	space := sxpf.MakeString(" ")
	text := make([]sxpf.Object, 0, 2*len(words))
	for _, tag := range words {
		text = append(text, space, wui.transformWord(key, tag))
	}
	return sxpf.MakeList(text[1:]...).Cons(wui.sf.MustMake("span"))
}

func (wui *WebUI) transformLink(key, value, text string) sxpf.Object {
	return sxpf.MakeList(
		wui.sf.MustMake("a"),
		sxpf.MakeList(
			wui.sf.MustMake(sxhtml.NameSymAttr),
			sxpf.Cons(wui.sf.MustMake("href"), sxpf.MakeString(wui.NewURLBuilder('h').AppendQuery(key+api.SearchOperatorHas+value).String())),
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
	return gen.InlinesSxHTML(&is).Cons(wui.sf.MustMake("span"))
}

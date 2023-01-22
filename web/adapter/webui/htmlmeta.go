//-----------------------------------------------------------------------------
// Copyright (c) 2020-2023 Detlef Stern
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
	"io"
	"net/url"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/c/html"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
)

var space = []byte{' '}

func (wui *WebUI) writeHTMLMetaValue(
	w io.Writer,
	key, value string,
	getTextTitle getTextTitleFunc,
	evalMetadata evalMetadataFunc,
	gen *htmlGenerator,
) {
	switch kt := meta.Type(key); kt {
	case meta.TypeCredential:
		writeCredential(w, value)
	case meta.TypeEmpty:
		writeEmpty(w, value)
	case meta.TypeID:
		wui.writeIdentifier(w, value, getTextTitle)
	case meta.TypeIDSet:
		wui.writeIdentifierSet(w, meta.ListFromValue(value), getTextTitle)
	case meta.TypeNumber:
		wui.writeNumber(w, key, value)
	case meta.TypeString:
		writeString(w, value)
	case meta.TypeTagSet:
		wui.writeTagSet(w, key, meta.ListFromValue(value))
	case meta.TypeTimestamp:
		if ts, ok := meta.TimeValue(value); ok {
			writeTimestamp(w, ts)
		}
	case meta.TypeURL:
		writeURL(w, value)
	case meta.TypeWord:
		wui.writeWord(w, key, value)
	case meta.TypeWordSet:
		wui.writeWordSet(w, key, meta.ListFromValue(value))
	case meta.TypeZettelmarkup:
		io.WriteString(w, encodeZmkMetadata(value, evalMetadata, gen))
	default:
		html.Escape(w, value)
		fmt.Fprintf(w, " <b>(Unhandled type: %v, key: %v)</b>", kt, key)
	}
}

func writeCredential(w io.Writer, val string) { html.Escape(w, val) }
func writeEmpty(w io.Writer, val string)      { html.Escape(w, val) }

func (wui *WebUI) writeIdentifier(w io.Writer, val string, getTextTitle getTextTitleFunc) {
	zid, err := id.Parse(val)
	if err != nil {
		html.Escape(w, val)
		return
	}
	title, found := getTextTitle(zid)
	switch {
	case found > 0:
		ub := wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String()))
		if title == "" {
			fmt.Fprintf(w, "<a href=\"%v\">%v</a>", ub, zid)
		} else {
			fmt.Fprintf(w, "<a href=\"%v\" title=\"%v\">%v</a>", ub, title, zid)
		}
	case found == 0:
		fmt.Fprintf(w, "<s>%v</s>", val)
	case found < 0:
		io.WriteString(w, val)
	}
}

func (wui *WebUI) writeIdentifierSet(w io.Writer, vals []string, getTextTitle getTextTitleFunc) {
	for i, val := range vals {
		if i > 0 {
			w.Write(space)
		}
		wui.writeIdentifier(w, val, getTextTitle)
	}
}

func (wui *WebUI) writeNumber(w io.Writer, key, val string) {
	wui.writeLink(w, key, val, val)
}

func writeString(w io.Writer, val string) { html.Escape(w, val) }

func (wui *WebUI) writeTagSet(w io.Writer, key string, tags []string) {
	for i, tag := range tags {
		if i > 0 {
			w.Write(space)
		}
		wui.writeLink(w, key, tag, tag)
	}
}

func writeTimestamp(w io.Writer, ts time.Time) {
	io.WriteString(w, `<time datetime="`)
	io.WriteString(w, ts.Format("2006-01-02T15:04:05"))
	io.WriteString(w, `">`)
	io.WriteString(w, ts.Format("2006-01-02&nbsp;15:04:05"))
	io.WriteString(w, `</time>`)
}

func writeURL(w io.Writer, val string) {
	u, err := url.Parse(val)
	if err != nil {
		html.Escape(w, val)
		return
	}
	if us := u.String(); us != "" {
		io.WriteString(w, `<a href="`)
		html.AttributeEscape(w, us)
		io.WriteString(w, `" target="_blank" rel="noopener noreferrer">`)
		html.Escape(w, val)
		io.WriteString(w, "</a>")
	}
}

func (wui *WebUI) writeWord(w io.Writer, key, word string) {
	wui.writeLink(w, key, word, word)
}

func (wui *WebUI) writeWordSet(w io.Writer, key string, words []string) {
	for i, word := range words {
		if i > 0 {
			w.Write(space)
		}
		wui.writeWord(w, key, word)
	}
}

func (wui *WebUI) writeLink(w io.Writer, key, value, text string) {
	fmt.Fprintf(w, `<a href="%v">`, wui.NewURLBuilder('h').AppendQuery(key+api.SearchOperatorHas+value))
	html.Escape(w, text)
	io.WriteString(w, "</a>")
}

type getMetadataFunc func(id.Zid) (*meta.Meta, error)

func createGetMetadataFunc(ctx context.Context, getMeta usecase.GetMeta) getMetadataFunc {
	return func(zid id.Zid) (*meta.Meta, error) { return getMeta.Run(box.NoEnrichContext(ctx), zid) }
}

type evalMetadataFunc = func(string) ast.InlineSlice

func createEvalMetadataFunc(ctx context.Context, evaluate *usecase.Evaluate) evalMetadataFunc {
	return func(value string) ast.InlineSlice { return evaluate.RunMetadata(ctx, value) }
}

type getTextTitleFunc func(id.Zid) (string, int)

func (wui *WebUI) makeGetTextTitle(getMetadata getMetadataFunc, evalMetadata evalMetadataFunc) getTextTitleFunc {
	return func(zid id.Zid) (string, int) {
		m, err := getMetadata(zid)
		if err != nil {
			if errors.Is(err, &box.ErrNotAllowed{}) {
				return "", -1
			}
			return "", 0
		}
		return encodeEvaluatedTitleText(m, evalMetadata, wui.gentext), 1
	}
}

func encodeZmkMetadata(value string, evalMetadata evalMetadataFunc, gen *htmlGenerator) string {
	is := evalMetadata(value)
	return gen.InlinesString(&is)
}

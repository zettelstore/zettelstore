//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides web-UI handlers for web requests.
package webui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/usecase"
)

var space = []byte{' '}

type evalMetadataFunc = func(string) *ast.InlineListNode

func (wui *WebUI) writeHTMLMetaValue(
	w io.Writer,
	key, value string,
	getTextTitle getTextTitleFunc,
	evalMetadata evalMetadataFunc,
	envEnc *encoder.Environment,
) {
	switch kt := meta.Type(key); kt {
	case meta.TypeBool:
		wui.writeHTMLBool(w, key, value)
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
		io.WriteString(w, encodeZmkMetadata(value, evalMetadata, api.EncoderHTML, envEnc))
	default:
		strfun.HTMLEscape(w, value, false)
		fmt.Fprintf(w, " <b>(Unhandled type: %v, key: %v)</b>", kt, key)
	}
}

func (wui *WebUI) writeHTMLBool(w io.Writer, key, value string) {
	if meta.BoolValue(value) {
		wui.writeLink(w, key, "true", "True")
	} else {
		wui.writeLink(w, key, "false", "False")
	}
}

func writeCredential(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func writeEmpty(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func (wui *WebUI) writeIdentifier(w io.Writer, val string, getTextTitle getTextTitleFunc) {
	zid, err := id.Parse(val)
	if err != nil {
		strfun.HTMLEscape(w, val, false)
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

func writeString(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func (wui *WebUI) writeTagSet(w io.Writer, key string, tags []string) {
	for i, tag := range tags {
		if i > 0 {
			w.Write(space)
		}
		wui.writeLink(w, key, tag, tag)
	}
}

func writeTimestamp(w io.Writer, ts time.Time) {
	io.WriteString(w, ts.Format("2006-01-02&nbsp;15:04:05"))
}

func writeURL(w io.Writer, val string) {
	u, err := url.Parse(val)
	if err != nil {
		strfun.HTMLEscape(w, val, false)
		return
	}
	fmt.Fprintf(w, "<a href=\"%v\"%v>", u, htmlAttrNewWindow(true))
	strfun.HTMLEscape(w, val, false)
	io.WriteString(w, "</a>")
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
	fmt.Fprintf(w, "<a href=\"%v?%v=%v\">", wui.NewURLBuilder('h'), url.QueryEscape(key), url.QueryEscape(value))
	strfun.HTMLEscape(w, text, false)
	io.WriteString(w, "</a>")
}

type getTextTitleFunc func(id.Zid) (string, int)

func (wui *WebUI) makeGetTextTitle(
	ctx context.Context,
	getMeta usecase.GetMeta, evaluate *usecase.Evaluate,
) getTextTitleFunc {
	return func(zid id.Zid) (string, int) {
		m, err := getMeta.Run(box.NoEnrichContext(ctx), zid)
		if err != nil {
			if errors.Is(err, &box.ErrNotAllowed{}) {
				return "", -1
			}
			return "", 0
		}
		return wui.encodeTitleAsText(ctx, m, evaluate), 1
	}
}

func (wui *WebUI) encodeTitleAsHTML(
	ctx context.Context, m *meta.Meta,
	evaluate *usecase.Evaluate, envEval *evaluator.Environment,
	envHTML *encoder.Environment,
) string {
	plainTitle := config.GetTitle(m, wui.rtConfig)
	return encodeZmkMetadata(
		plainTitle,
		func(val string) *ast.InlineListNode {
			return evaluate.RunMetadata(ctx, plainTitle, envEval)
		},
		api.EncoderHTML, envHTML)
}

func (wui *WebUI) encodeTitleAsText(
	ctx context.Context, m *meta.Meta, evaluate *usecase.Evaluate,
) string {
	plainTitle := config.GetTitle(m, wui.rtConfig)
	return encodeZmkMetadata(
		plainTitle,
		func(val string) *ast.InlineListNode {
			return evaluate.RunMetadata(ctx, plainTitle, nil)
		},
		api.EncoderHTML, nil)
}

func encodeZmkMetadata(
	value string, evalMetadata evalMetadataFunc,
	enc api.EncodingEnum, envHTML *encoder.Environment,
) string {
	iln := evalMetadata(value)
	if iln.IsEmpty() {
		return ""
	}
	result, err := encodeInlines(iln, enc, envHTML)
	if err != nil {
		return err.Error()
	}
	return result
}

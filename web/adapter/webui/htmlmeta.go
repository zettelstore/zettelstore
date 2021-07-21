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

	"zettelstore.de/z/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

var space = []byte{' '}

func (wui *WebUI) writeHTMLMetaValue(w io.Writer, m *meta.Meta, key string, getTitle getTitleFunc, env *encoder.Environment) {
	switch kt := m.Type(key); kt {
	case meta.TypeBool:
		wui.writeHTMLBool(w, key, m.GetBool(key))
	case meta.TypeCredential:
		writeCredential(w, m.GetDefault(key, "???c"))
	case meta.TypeEmpty:
		writeEmpty(w, m.GetDefault(key, "???e"))
	case meta.TypeID:
		wui.writeIdentifier(w, m.GetDefault(key, "???i"), getTitle)
	case meta.TypeIDSet:
		if l, ok := m.GetList(key); ok {
			wui.writeIdentifierSet(w, l, getTitle)
		}
	case meta.TypeNumber:
		wui.writeNumber(w, key, m.GetDefault(key, "???n"))
	case meta.TypeString:
		writeString(w, m.GetDefault(key, "???s"))
	case meta.TypeTagSet:
		if l, ok := m.GetList(key); ok {
			wui.writeTagSet(w, key, l)
		}
	case meta.TypeTimestamp:
		if ts, ok := m.GetTime(key); ok {
			writeTimestamp(w, ts)
		}
	case meta.TypeURL:
		writeURL(w, m.GetDefault(key, "???u"))
	case meta.TypeWord:
		wui.writeWord(w, key, m.GetDefault(key, "???w"))
	case meta.TypeWordSet:
		if l, ok := m.GetList(key); ok {
			wui.writeWordSet(w, key, l)
		}
	case meta.TypeZettelmarkup:
		writeZettelmarkup(w, m.GetDefault(key, "???z"), env)
	default:
		strfun.HTMLEscape(w, m.GetDefault(key, "???w"), false)
		fmt.Fprintf(w, " <b>(Unhandled type: %v, key: %v)</b>", kt, key)
	}
}

func (wui *WebUI) writeHTMLBool(w io.Writer, key string, val bool) {
	if val {
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

func (wui *WebUI) writeIdentifier(w io.Writer, val string, getTitle getTitleFunc) {
	zid, err := id.Parse(val)
	if err != nil {
		strfun.HTMLEscape(w, val, false)
		return
	}
	title, found := getTitle(zid, api.EncoderText)
	switch {
	case found > 0:
		if title == "" {
			fmt.Fprintf(w, "<a href=\"%v\">%v</a>", wui.NewURLBuilder('h').SetZid(zid), zid)
		} else {
			fmt.Fprintf(w, "<a href=\"%v\" title=\"%v\">%v</a>", wui.NewURLBuilder('h').SetZid(zid), title, zid)
		}
	case found == 0:
		fmt.Fprintf(w, "<s>%v</s>", val)
	case found < 0:
		io.WriteString(w, val)
	}
}

func (wui *WebUI) writeIdentifierSet(w io.Writer, vals []string, getTitle getTitleFunc) {
	for i, val := range vals {
		if i > 0 {
			w.Write(space)
		}
		wui.writeIdentifier(w, val, getTitle)
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
func writeZettelmarkup(w io.Writer, val string, env *encoder.Environment) {
	title, err := adapter.FormatInlines(parser.ParseMetadata(val), api.EncoderHTML, env)
	if err != nil {
		strfun.HTMLEscape(w, val, false)
		return
	}
	io.WriteString(w, title)
}

func (wui *WebUI) writeLink(w io.Writer, key, value, text string) {
	fmt.Fprintf(w, "<a href=\"%v?%v=%v\">", wui.NewURLBuilder('h'), url.QueryEscape(key), url.QueryEscape(value))
	strfun.HTMLEscape(w, text, false)
	io.WriteString(w, "</a>")
}

type getTitleFunc func(id.Zid, api.EncodingEnum) (string, int)

func makeGetTitle(ctx context.Context, getMeta usecase.GetMeta, env *encoder.Environment) getTitleFunc {
	return func(zid id.Zid, format api.EncodingEnum) (string, int) {
		m, err := getMeta.Run(box.NoEnrichContext(ctx), zid)
		if err != nil {
			if errors.Is(err, &box.ErrNotAllowed{}) {
				return "", -1
			}
			return "", 0
		}
		astTitle := parser.ParseMetadata(m.GetDefault(meta.KeyTitle, ""))
		title, err := adapter.FormatInlines(astTitle, format, env)
		if err == nil {
			return title, 1
		}
		return "", 1
	}
}

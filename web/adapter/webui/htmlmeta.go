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
	"fmt"
	"io"
	"net/url"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

var space = []byte{' '}

func writeHTMLMetaValue(w io.Writer, builder server.URLBuilderFunc, m *meta.Meta, key string, getTitle getTitleFunc, env *encoder.Environment) {
	switch kt := m.Type(key); kt {
	case meta.TypeBool:
		writeHTMLBool(w, builder, key, m.GetBool(key))
	case meta.TypeCredential:
		writeCredential(w, m.GetDefault(key, "???c"))
	case meta.TypeEmpty:
		writeEmpty(w, m.GetDefault(key, "???e"))
	case meta.TypeID:
		writeIdentifier(w, builder, m.GetDefault(key, "???i"), getTitle)
	case meta.TypeIDSet:
		if l, ok := m.GetList(key); ok {
			writeIdentifierSet(w, builder, l, getTitle)
		}
	case meta.TypeNumber:
		writeNumber(w, m.GetDefault(key, "???n"))
	case meta.TypeString:
		writeString(w, m.GetDefault(key, "???s"))
	case meta.TypeTagSet:
		if l, ok := m.GetList(key); ok {
			writeTagSet(w, builder, key, l)
		}
	case meta.TypeTimestamp:
		if ts, ok := m.GetTime(key); ok {
			writeTimestamp(w, ts)
		}
	case meta.TypeURL:
		writeURL(w, m.GetDefault(key, "???u"))
	case meta.TypeWord:
		writeWord(w, builder, key, m.GetDefault(key, "???w"))
	case meta.TypeWordSet:
		if l, ok := m.GetList(key); ok {
			writeWordSet(w, builder, key, l)
		}
	case meta.TypeZettelmarkup:
		writeZettelmarkup(w, m.GetDefault(key, "???z"), env)
	case meta.TypeUnknown:
		writeUnknown(w, m.GetDefault(key, "???u"))
	default:
		strfun.HTMLEscape(w, m.GetDefault(key, "???w"), false)
		fmt.Fprintf(w, " <b>(Unhandled type: %v, key: %v)</b>", kt, key)
	}
}

func writeHTMLBool(w io.Writer, builder server.URLBuilderFunc, key string, val bool) {
	if val {
		writeLink(w, builder, key, "true", "True")
	} else {
		writeLink(w, builder, key, "false", "False")
	}
}

func writeCredential(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func writeEmpty(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func writeIdentifier(w io.Writer, builder server.URLBuilderFunc, val string, getTitle func(id.Zid, string) (string, int)) {
	zid, err := id.Parse(val)
	if err != nil {
		strfun.HTMLEscape(w, val, false)
		return
	}
	title, found := getTitle(zid, "text")
	switch {
	case found > 0:
		if title == "" {
			fmt.Fprintf(w, "<a href=\"%v\">%v</a>", builder('h').SetZid(zid), zid)
		} else {
			fmt.Fprintf(w, "<a href=\"%v\" title=\"%v\">%v</a>", builder('h').SetZid(zid), title, zid)
		}
	case found == 0:
		fmt.Fprintf(w, "<s>%v</s>", val)
	case found < 0:
		io.WriteString(w, val)
	}
}

func writeIdentifierSet(w io.Writer, builder server.URLBuilderFunc, vals []string, getTitle func(id.Zid, string) (string, int)) {
	for i, val := range vals {
		if i > 0 {
			w.Write(space)
		}
		writeIdentifier(w, builder, val, getTitle)
	}
}

func writeNumber(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func writeString(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func writeUnknown(w io.Writer, val string) {
	strfun.HTMLEscape(w, val, false)
}

func writeTagSet(w io.Writer, builder server.URLBuilderFunc, key string, tags []string) {
	for i, tag := range tags {
		if i > 0 {
			w.Write(space)
		}
		writeLink(w, builder, key, tag, tag)
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
	fmt.Fprintf(w, "<a href=\"%v\">", u)
	strfun.HTMLEscape(w, val, false)
	io.WriteString(w, "</a>")
}

func writeWord(w io.Writer, builder server.URLBuilderFunc, key, word string) {
	writeLink(w, builder, key, word, word)
}

func writeWordSet(w io.Writer, builder server.URLBuilderFunc, key string, words []string) {
	for i, word := range words {
		if i > 0 {
			w.Write(space)
		}
		writeWord(w, builder, key, word)
	}
}
func writeZettelmarkup(w io.Writer, val string, env *encoder.Environment) {
	title, err := adapter.FormatInlines(parser.ParseMetadata(val), "html", env)
	if err != nil {
		strfun.HTMLEscape(w, val, false)
		return
	}
	io.WriteString(w, title)
}

func writeLink(w io.Writer, builder server.URLBuilderFunc, key, value, text string) {
	fmt.Fprintf(w, "<a href=\"%v?%v=%v\">", builder('h'), url.QueryEscape(key), url.QueryEscape(value))
	strfun.HTMLEscape(w, text, false)
	io.WriteString(w, "</a>")
}

type getTitleFunc func(id.Zid, string) (string, int)

func makeGetTitle(ctx context.Context, getMeta usecase.GetMeta, env *encoder.Environment) getTitleFunc {
	return func(zid id.Zid, format string) (string, int) {
		m, err := getMeta.Run(place.NoEnrichContext(ctx), zid)
		if err != nil {
			if place.IsErrNotAllowed(err) {
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

//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

type jsonIDURL struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type jsonZettel struct {
	ID       string            `json:"id"`
	URL      string            `json:"url"`
	Meta     map[string]string `json:"meta"`
	Encoding string            `json:"encoding"`
	Content  interface{}       `json:"content"`
}
type jsonMeta struct {
	ID   string            `json:"id"`
	URL  string            `json:"url"`
	Meta map[string]string `json:"meta"`
}
type jsonMetaList struct {
	ID   string            `json:"id"`
	URL  string            `json:"url"`
	Meta map[string]string `json:"meta"`
	List []jsonMeta        `json:"list"`
}
type jsonContent struct {
	ID       string      `json:"id"`
	URL      string      `json:"url"`
	Encoding string      `json:"encoding"`
	Content  interface{} `json:"content"`
}

func writeJSONZettel(w http.ResponseWriter, z *ast.ZettelNode, part partType) error {
	var outData interface{}
	idData := jsonIDURL{
		ID:  z.Zid.String(),
		URL: adapter.NewURLBuilder('z').SetZid(z.Zid).String(),
	}

	switch part {
	case partZettel:
		encoding, content := encodedContent(z.Content)
		outData = jsonZettel{
			ID:       idData.ID,
			URL:      idData.URL,
			Meta:     z.InhMeta.Map(),
			Encoding: encoding,
			Content:  content,
		}
	case partMeta:
		outData = jsonMeta{
			ID:   idData.ID,
			URL:  idData.URL,
			Meta: z.InhMeta.Map(),
		}
	case partContent:
		encoding, content := encodedContent(z.Content)
		outData = jsonContent{
			ID:       idData.ID,
			URL:      idData.URL,
			Encoding: encoding,
			Content:  content,
		}
	case partID:
		outData = idData
	default:
		panic(part)
	}
	return encodeJSONData(w, outData, false)
}

func encodedContent(content domain.Content) (string, interface{}) {
	if content.IsBinary() {
		return "base64", content.AsBytes()
	}
	return "", content.AsString()
}

func writeDJSONZettel(
	ctx context.Context,
	w http.ResponseWriter,
	z *ast.ZettelNode,
	urlPrefix string,
	part, defPart partType,
	getMeta usecase.GetMeta,
) (err error) {
	switch part {
	case partZettel:
		err = writeDJSONHeader(w, z.Zid)
		if err == nil {
			err = writeDJSONMeta(w, z)
		}
		if err == nil {
			err = writeDJSONContent(ctx, w, z, urlPrefix, part, defPart, getMeta)
		}
	case partMeta:
		err = writeDJSONHeader(w, z.Zid)
		if err == nil {
			err = writeDJSONMeta(w, z)
		}
	case partContent:
		err = writeDJSONHeader(w, z.Zid)
		if err == nil {
			err = writeDJSONContent(ctx, w, z, urlPrefix, part, defPart, getMeta)
		}
	case partID:
		writeDJSONHeader(w, z.Zid)
	default:
		panic(part)
	}
	if err == nil {
		_, err = w.Write(djsonFooter)
	}
	return err
}

var (
	djsonMetaHeader    = []byte(",\"meta\":")
	djsonContentHeader = []byte(",\"content\":")
	djsonHeader1       = []byte("{\"id\":\"")
	djsonHeader2       = []byte("\",\"url\":\"")
	djsonHeader3       = []byte("?_format=")
	djsonHeader4       = []byte("\"")
	djsonFooter        = []byte("}")
)

func writeDJSONHeader(w http.ResponseWriter, zid id.Zid) error {
	_, err := w.Write(djsonHeader1)
	if err == nil {
		_, err = w.Write(zid.Bytes())
	}
	if err == nil {
		_, err = w.Write(djsonHeader2)
	}
	if err == nil {
		_, err = io.WriteString(w, adapter.NewURLBuilder('z').SetZid(zid).String())
	}
	if err == nil {
		_, err = w.Write(djsonHeader3)
		if err == nil {
			_, err = io.WriteString(w, "djson")
		}
	}
	if err == nil {
		_, err = w.Write(djsonHeader4)
	}
	return err
}

func writeDJSONMeta(w io.Writer, z *ast.ZettelNode) error {
	_, err := w.Write(djsonMetaHeader)
	if err == nil {
		err = writeMeta(w, z.InhMeta, "djson", nil)
	}
	return err
}

func writeDJSONContent(
	ctx context.Context,
	w io.Writer,
	z *ast.ZettelNode,
	urlPrefix string,
	part, defPart partType,
	getMeta usecase.GetMeta,
) (err error) {
	_, err = w.Write(djsonContentHeader)
	if err == nil {
		err = writeContent(w, z, "djson", &encoder.Environment{
			LinkAdapter:  adapter.MakeLinkAdapter(ctx, urlPrefix, 'z', getMeta, part.DefString(defPart), "djson"),
			ImageAdapter: adapter.MakeImageAdapter(ctx, getMeta)})
	}
	return err
}

var (
	jsonListHeader = []byte("{\"list\":[")
	jsonListSep    = []byte{','}
	jsonListFooter = []byte("]}")
)

var setJSON = map[string]bool{"json": true}

func renderListMetaXJSON(
	ctx context.Context,
	w http.ResponseWriter,
	metaList []*meta.Meta,
	urlPrefix string,
	format string,
	part, defPart partType,
	getMeta usecase.GetMeta,
	parseZettel usecase.ParseZettel,
) {
	var readZettel bool
	switch part {
	case partZettel, partContent:
		readZettel = true
	case partMeta, partID:
		readZettel = false
	default:
		adapter.BadRequest(w, "Unknown _part parameter")
		return
	}
	isJSON := setJSON[format]
	_, err := w.Write(jsonListHeader)
	for i, m := range metaList {
		if err != nil {
			break
		}
		if i > 0 {
			_, err = w.Write(jsonListSep)
		}
		if err != nil {
			break
		}
		var zn *ast.ZettelNode
		if readZettel {
			z, err1 := parseZettel.Run(ctx, m.Zid, "")
			if err1 != nil {
				err = err1
				break
			}
			zn = z
		} else {
			zn = &ast.ZettelNode{
				Meta:    m,
				Content: "",
				Zid:     m.Zid,
				InhMeta: runtime.AddDefaultValues(m),
				Ast:     nil,
			}
		}
		if isJSON {
			err = writeJSONZettel(w, zn, part)
		} else {
			err = writeDJSONZettel(ctx, w, zn, urlPrefix, part, defPart, getMeta)
		}
	}
	if err == nil {
		_, err = w.Write(jsonListFooter)
	}
	if err != nil {
		adapter.InternalServerError(w, "Get list", err)
	}
}

func writeContent(
	w io.Writer, zn *ast.ZettelNode, format string, env *encoder.Environment) error {
	enc := encoder.Create(format, env)
	if enc == nil {
		return adapter.ErrNoSuchFormat
	}

	_, err := enc.WriteContent(w, zn)
	return err
}

func writeMeta(
	w io.Writer, m *meta.Meta, format string, env *encoder.Environment) error {
	enc := encoder.Create(format, env)
	if enc == nil {
		return adapter.ErrNoSuchFormat
	}

	_, err := enc.WriteMeta(w, m)
	return err
}

func encodeJSONData(w http.ResponseWriter, data interface{}, addHeader bool) error {
	w.Header().Set(adapter.ContentType, format2ContentType("json"))
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

func writeMetaList(w http.ResponseWriter, m *meta.Meta, metaList []*meta.Meta) error {
	outData := jsonMetaList{
		ID:   m.Zid.String(),
		URL:  adapter.NewURLBuilder('z').SetZid(m.Zid).String(),
		Meta: m.Map(),
		List: make([]jsonMeta, len(metaList)),
	}
	for i, m := range metaList {
		outData.List[i].ID = m.Zid.String()
		outData.List[i].URL = adapter.NewURLBuilder('z').SetZid(m.Zid).String()
		outData.List[i].Meta = m.Map()
	}
	return encodeJSONData(w, outData, true)
}

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
	"zettelstore.de/z/web/server"
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

func encodedContent(content domain.Content) (string, interface{}) {
	if content.IsBinary() {
		return "base64", content.AsBytes()
	}
	return "", content.AsString()
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

func writeDJSONHeader(w io.Writer, builder server.URLBuilderFunc, zid id.Zid) error {
	_, err := w.Write(djsonHeader1)
	if err == nil {
		_, err = w.Write(zid.Bytes())
	}
	if err == nil {
		_, err = w.Write(djsonHeader2)
	}
	if err == nil {
		_, err = io.WriteString(w, builder('z').SetZid(zid).String())
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

func renderListMetaXJSON(
	ctx context.Context,
	w http.ResponseWriter,
	metaList []*meta.Meta,
	format string,
	part, defPart partType,
	getMeta usecase.GetMeta,
	parseZettel usecase.ParseZettel,
) {
	prepareZettel := getPrepareZettelFunc(ctx, parseZettel, part)
	writeZettel := getWriteMetaZettelFunc(ctx, format, part, defPart, getMeta)
	err := writeListXJSON(w, metaList, prepareZettel, writeZettel)
	if err != nil {
		adapter.InternalServerError(w, "Get list", err)
	}
}

type prepareZettelFunc func(m *meta.Meta) (*ast.ZettelNode, error)

func getPrepareZettelFunc(ctx context.Context, parseZettel usecase.ParseZettel, part partType) prepareZettelFunc {
	switch part {
	case partZettel, partContent:
		return func(m *meta.Meta) (*ast.ZettelNode, error) {
			return parseZettel.Run(ctx, m.Zid, "")
		}
	case partMeta, partID:
		return func(m *meta.Meta) (*ast.ZettelNode, error) {
			return &ast.ZettelNode{
				Meta:    m,
				Content: "",
				Zid:     m.Zid,
				InhMeta: runtime.AddDefaultValues(m),
				Ast:     nil,
			}, nil
		}
	}
	return nil
}

type writeZettelFunc func(io.Writer, *ast.ZettelNode) error

func getWriteMetaZettelFunc(ctx context.Context, format string,
	part, defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
	switch part {
	case partZettel:
		return getWriteZettelFunc(ctx, format, defPart, getMeta)
	case partMeta:
		return getWriteMetaFunc(ctx, format)
	case partContent:
		return getWriteContentFunc(ctx, format, defPart, getMeta)
	case partID:
		return getWriteIDFunc(ctx, format)
	default:
		panic(part)
	}
}

func getWriteZettelFunc(ctx context.Context, format string,
	defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
	builder := server.GetURLBuilderFunc(ctx)
	if format == "json" {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			encoding, content := encodedContent(zn.Content)
			return encodeJSONData(w, jsonZettel{
				ID:       zn.Zid.String(),
				URL:      builder('z').SetZid(zn.Zid).String(),
				Meta:     zn.InhMeta.Map(),
				Encoding: encoding,
				Content:  content,
			})
		}
	}
	enc := encoder.Create("djson", nil)
	if enc == nil {
		panic("no DJSON encoder found")
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := writeDJSONHeader(w, builder, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonMetaHeader)
		if err != nil {
			return err
		}
		_, err = enc.WriteMeta(w, zn.InhMeta)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonContentHeader)
		if err != nil {
			return err
		}
		err = writeContent(w, zn, "djson", &encoder.Environment{
			LinkAdapter:  adapter.MakeLinkAdapter(ctx, 'z', getMeta, partZettel.DefString(defPart), "djson"),
			ImageAdapter: adapter.MakeImageAdapter(ctx, getMeta)})
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func getWriteMetaFunc(ctx context.Context, format string) writeZettelFunc {
	builder := server.GetURLBuilderFunc(ctx)
	if format == "json" {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			return encodeJSONData(w, jsonMeta{
				ID:   zn.Zid.String(),
				URL:  builder('z').SetZid(zn.Zid).String(),
				Meta: zn.InhMeta.Map(),
			})
		}
	}
	enc := encoder.Create("djson", nil)
	if enc == nil {
		panic("no DJSON encoder found")
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := writeDJSONHeader(w, builder, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonMetaHeader)
		if err != nil {
			return err
		}
		_, err = enc.WriteMeta(w, zn.InhMeta)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func getWriteContentFunc(ctx context.Context, format string,
	defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
	builder := server.GetURLBuilderFunc(ctx)
	if format == "json" {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			encoding, content := encodedContent(zn.Content)
			return encodeJSONData(w, jsonContent{
				ID:       zn.Zid.String(),
				URL:      builder('z').SetZid(zn.Zid).String(),
				Encoding: encoding,
				Content:  content,
			})
		}
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := writeDJSONHeader(w, builder, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonContentHeader)
		if err != nil {
			return err
		}
		err = writeContent(w, zn, "djson", &encoder.Environment{
			LinkAdapter:  adapter.MakeLinkAdapter(ctx, 'z', getMeta, partContent.DefString(defPart), "djson"),
			ImageAdapter: adapter.MakeImageAdapter(ctx, getMeta)})
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func getWriteIDFunc(ctx context.Context, format string) writeZettelFunc {
	builder := server.GetURLBuilderFunc(ctx)
	if format == "json" {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			return encodeJSONData(w, jsonIDURL{
				ID:  zn.Zid.String(),
				URL: builder('z').SetZid(zn.Zid).String(),
			})
		}
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := writeDJSONHeader(w, builder, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}

var (
	jsonListHeader = []byte("{\"list\":[")
	jsonListSep    = []byte{','}
	jsonListFooter = []byte("]}")
)

func writeListXJSON(w http.ResponseWriter, metaList []*meta.Meta, prepareZettel prepareZettelFunc, writeZettel writeZettelFunc) error {
	_, err := w.Write(jsonListHeader)
	for i, m := range metaList {
		if err != nil {
			return err
		}
		if i > 0 {
			_, err = w.Write(jsonListSep)
			if err != nil {
				return err
			}
		}
		zn, err1 := prepareZettel(m)
		if err1 != nil {
			return err1
		}
		err = writeZettel(w, zn)
	}
	if err == nil {
		_, err = w.Write(jsonListFooter)
	}
	return err
}

func writeContent(w io.Writer, zn *ast.ZettelNode, format string, env *encoder.Environment) error {
	enc := encoder.Create(format, env)
	if enc == nil {
		return adapter.ErrNoSuchFormat
	}

	_, err := enc.WriteContent(w, zn)
	return err
}

func encodeJSONData(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

func writeMetaList(w http.ResponseWriter, builder server.URLBuilderFunc, m *meta.Meta, metaList []*meta.Meta) error {
	outData := jsonMetaList{
		ID:   m.Zid.String(),
		URL:  builder('z').SetZid(m.Zid).String(),
		Meta: m.Map(),
		List: make([]jsonMeta, len(metaList)),
	}
	for i, m := range metaList {
		outData.List[i].ID = m.Zid.String()
		outData.List[i].URL = builder('z').SetZid(m.Zid).String()
		outData.List[i].Meta = m.Map()
	}
	w.Header().Set(adapter.ContentType, format2ContentType("json"))
	return encodeJSONData(w, outData)
}

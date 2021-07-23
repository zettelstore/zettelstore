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

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

type jsonContent struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

var (
	djsonMetaHeader    = []byte(",\"meta\":")
	djsonContentHeader = []byte(",\"content\":")
	djsonHeader1       = []byte("{\"id\":\"")
	djsonHeader2       = []byte("\",\"url\":\"")
	djsonHeader3       = []byte("?" + zsapi.QueryKeyEncoding + "=")
	djsonHeader4       = []byte("\"")
	djsonFooter        = []byte("}")
)

func (api *API) writeDJSONHeader(w io.Writer, zid id.Zid) error {
	_, err := w.Write(djsonHeader1)
	if err == nil {
		_, err = w.Write(zid.Bytes())
	}
	if err == nil {
		_, err = w.Write(djsonHeader2)
	}
	if err == nil {
		_, err = io.WriteString(w, api.NewURLBuilder('z').SetZid(zid).String())
	}
	if err == nil {
		_, err = w.Write(djsonHeader3)
		if err == nil {
			_, err = io.WriteString(w, zsapi.EncodingDJSON)
		}
	}
	if err == nil {
		_, err = w.Write(djsonHeader4)
	}
	return err
}

func (api *API) renderListMetaXJSON(
	ctx context.Context,
	w http.ResponseWriter,
	metaList []*meta.Meta,
	enc zsapi.EncodingEnum,
	part, defPart partType,
	getMeta usecase.GetMeta,
	parseZettel usecase.ParseZettel,
) {
	prepareZettel := api.getPrepareZettelFunc(ctx, parseZettel, part)
	writeZettel := api.getWriteMetaZettelFunc(ctx, enc, part, defPart, getMeta)
	err := writeListXJSON(w, metaList, prepareZettel, writeZettel)
	if err != nil {
		adapter.InternalServerError(w, "Get list", err)
	}
}

type prepareZettelFunc func(m *meta.Meta) (*ast.ZettelNode, error)

func (api *API) getPrepareZettelFunc(ctx context.Context, parseZettel usecase.ParseZettel, part partType) prepareZettelFunc {
	switch part {
	case partZettel, partContent:
		return func(m *meta.Meta) (*ast.ZettelNode, error) {
			return parseZettel.Run(ctx, m.Zid, "")
		}
	case partMeta, partID:
		return func(m *meta.Meta) (*ast.ZettelNode, error) {
			return &ast.ZettelNode{
				Meta:    m,
				Content: domain.NewContent(""),
				Zid:     m.Zid,
				InhMeta: api.rtConfig.AddDefaultValues(m),
				Ast:     nil,
			}, nil
		}
	}
	return nil
}

type writeZettelFunc func(io.Writer, *ast.ZettelNode) error

func (api *API) getWriteMetaZettelFunc(ctx context.Context, enc zsapi.EncodingEnum,
	part, defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
	switch part {
	case partZettel:
		return api.getWriteZettelFunc(ctx, enc, defPart, getMeta)
	case partMeta:
		return api.getWriteMetaFunc(ctx, enc)
	case partContent:
		return api.getWriteContentFunc(ctx, enc, defPart, getMeta)
	case partID:
		return api.getWriteIDFunc(ctx, enc)
	default:
		panic(part)
	}
}

func (api *API) getWriteZettelFunc(ctx context.Context, enc zsapi.EncodingEnum,
	defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
	if enc == zsapi.EncoderJSON {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			content, encoding := zn.Content.Encode()
			return encodeJSONData(w, zsapi.ZettelJSON{
				ID:       zn.Zid.String(),
				URL:      api.NewURLBuilder('z').SetZid(zn.Zid).String(),
				Meta:     zn.InhMeta.Map(),
				Encoding: encoding,
				Content:  content,
			})
		}
	}
	encdr := encoder.Create(zsapi.EncoderDJSON, nil)
	if encdr == nil {
		panic("no DJSON encoder found")
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := api.writeDJSONHeader(w, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonMetaHeader)
		if err != nil {
			return err
		}
		_, err = encdr.WriteMeta(w, zn.InhMeta)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonContentHeader)
		if err != nil {
			return err
		}
		err = writeContent(w, zn, zsapi.EncoderDJSON, &encoder.Environment{
			LinkAdapter: adapter.MakeLinkAdapter(
				ctx, api, 'z', getMeta, partZettel.DefString(defPart), zsapi.EncoderDJSON),
			ImageAdapter: adapter.MakeImageAdapter(ctx, api, getMeta)})
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func (api *API) getWriteMetaFunc(ctx context.Context, enc zsapi.EncodingEnum) writeZettelFunc {
	if enc == zsapi.EncoderJSON {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			return encodeJSONData(w, zsapi.ZidMetaJSON{
				ID:   zn.Zid.String(),
				URL:  api.NewURLBuilder('z').SetZid(zn.Zid).String(),
				Meta: zn.InhMeta.Map(),
			})
		}
	}
	encdr := encoder.Create(zsapi.EncoderDJSON, nil)
	if encdr == nil {
		panic("no DJSON encoder found")
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := api.writeDJSONHeader(w, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonMetaHeader)
		if err != nil {
			return err
		}
		_, err = encdr.WriteMeta(w, zn.InhMeta)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func (api *API) getWriteContentFunc(ctx context.Context, enc zsapi.EncodingEnum,
	defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
	if enc == zsapi.EncoderJSON {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			content, encoding := zn.Content.Encode()
			return encodeJSONData(w, jsonContent{
				ID:       zn.Zid.String(),
				URL:      api.NewURLBuilder('z').SetZid(zn.Zid).String(),
				Encoding: encoding,
				Content:  content,
			})
		}
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := api.writeDJSONHeader(w, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonContentHeader)
		if err != nil {
			return err
		}
		err = writeContent(w, zn, zsapi.EncoderDJSON, &encoder.Environment{
			LinkAdapter: adapter.MakeLinkAdapter(
				ctx, api, 'z', getMeta, partContent.DefString(defPart), zsapi.EncoderDJSON),
			ImageAdapter: adapter.MakeImageAdapter(ctx, api, getMeta)})
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func (api *API) getWriteIDFunc(ctx context.Context, enc zsapi.EncodingEnum) writeZettelFunc {
	if enc == zsapi.EncoderJSON {
		return func(w io.Writer, zn *ast.ZettelNode) error {
			return encodeJSONData(w, zsapi.ZidJSON{
				ID:  zn.Zid.String(),
				URL: api.NewURLBuilder('z').SetZid(zn.Zid).String(),
			})
		}
	}
	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := api.writeDJSONHeader(w, zn.Zid)
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

func writeContent(w io.Writer, zn *ast.ZettelNode, enc zsapi.EncodingEnum, env *encoder.Environment) error {
	encdr := encoder.Create(enc, env)
	if encdr == nil {
		return adapter.ErrNoSuchEncoding
	}

	_, err := encdr.WriteContent(w, zn)
	return err
}

func encodeJSONData(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

func (api *API) writeMetaList(w http.ResponseWriter, m *meta.Meta, metaList []*meta.Meta) error {
	outList := make([]zsapi.ZidMetaJSON, len(metaList))
	for i, m := range metaList {
		outList[i].ID = m.Zid.String()
		outList[i].URL = api.NewURLBuilder('z').SetZid(m.Zid).String()
		outList[i].Meta = m.Map()
	}
	w.Header().Set(zsapi.HeaderContentType, encoding2ContentType(zsapi.EncoderJSON))
	return encodeJSONData(w, zsapi.ZidMetaRelatedList{
		ID:   m.Zid.String(),
		URL:  api.NewURLBuilder('z').SetZid(m.Zid).String(),
		Meta: m.Map(),
		List: outList,
	})
}

func buildZettelFromData(r *http.Request, zid id.Zid) (domain.Zettel, error) {
	var zettel domain.Zettel
	dec := json.NewDecoder(r.Body)
	var zettelData zsapi.ZettelDataJSON
	if err := dec.Decode(&zettelData); err != nil {
		return zettel, err
	}
	m := meta.New(zid)
	for k, v := range zettelData.Meta {
		m.Set(k, v)
	}
	zettel.Meta = m
	if err := zettel.Content.SetDecoded(zettelData.Content, zettelData.Encoding); err != nil {
		return zettel, err
	}
	return zettel, nil
}

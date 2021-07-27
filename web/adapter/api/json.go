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
		_, err = io.WriteString(w, api.NewURLBuilder('v').SetZid(zid).String())
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

type writeZettelFunc func(io.Writer, *ast.ZettelNode) error

func (api *API) getWriteMetaZettelFunc(ctx context.Context,
	part, defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
	switch part {
	case partZettel:
		return api.getWriteZettelFunc(ctx, defPart, getMeta)
	case partMeta:
		return api.getWriteMetaFunc(ctx)
	case partContent:
		return api.getWriteContentFunc(ctx, defPart, getMeta)
	case partID:
		return api.getWriteIDFunc(ctx)
	default:
		panic(part)
	}
}

func (api *API) getWriteZettelFunc(ctx context.Context,
	defPart partType, getMeta usecase.GetMeta) writeZettelFunc {
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
				ctx, api, 'v', getMeta, partZettel.DefString(defPart), zsapi.EncoderDJSON),
			ImageAdapter: adapter.MakeImageAdapter(ctx, api, getMeta)})
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func (api *API) getWriteMetaFunc(ctx context.Context) writeZettelFunc {

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
func (api *API) getWriteContentFunc(ctx context.Context,
	defPart partType, getMeta usecase.GetMeta) writeZettelFunc {

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
				ctx, api, 'v', getMeta, partContent.DefString(defPart), zsapi.EncoderDJSON),
			ImageAdapter: adapter.MakeImageAdapter(ctx, api, getMeta)})
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
}
func (api *API) getWriteIDFunc(ctx context.Context) writeZettelFunc {

	return func(w io.Writer, zn *ast.ZettelNode) error {
		err := api.writeDJSONHeader(w, zn.Zid)
		if err != nil {
			return err
		}
		_, err = w.Write(djsonFooter)
		return err
	}
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
	w.Header().Set(zsapi.HeaderContentType, ctJSON)
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

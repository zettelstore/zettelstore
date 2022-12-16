//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"bytes"
	"context"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/content"
)

// MakeGetZettelHandler creates a new HTTP handler to return a zettel in various encodings.
func (a *API) MakeGetZettelHandler(getMeta usecase.GetMeta, getZettel usecase.GetZettel, parseZettel usecase.ParseZettel, evaluate usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		part := getPart(q, partContent)
		ctx := r.Context()
		switch enc, encStr := getEncoding(r, q, api.EncoderPlain); enc {
		case api.EncoderPlain:
			a.writePlainData(w, ctx, zid, part, getMeta, getZettel)

		case api.EncoderJson:
			a.writeJSONData(w, ctx, zid, part, getMeta, getZettel)

		default:
			var zn *ast.ZettelNode
			var em func(value string) ast.InlineSlice
			if q.Has(api.QueryKeyEval) {
				zn, err = evaluate.Run(ctx, zid, q.Get(api.KeySyntax))
				em = func(value string) ast.InlineSlice {
					return evaluate.RunMetadata(ctx, value)
				}
			} else {
				zn, err = parseZettel.Run(ctx, zid, q.Get(api.KeySyntax))
				em = parser.ParseMetadata
			}
			if err != nil {
				a.reportUsecaseError(w, err)
				return
			}
			a.writeEncodedZettelPart(w, zn, em, enc, encStr, part)
		}
	}
}

func (a *API) writePlainData(w http.ResponseWriter, ctx context.Context, zid id.Zid, part partType, getMeta usecase.GetMeta, getZettel usecase.GetZettel) {
	var buf bytes.Buffer
	var contentType string
	var err error

	switch part {
	case partZettel:
		z, err2 := getZettel.Run(box.NoEnrichContext(ctx), zid)
		if err2 != nil {
			a.reportUsecaseError(w, err2)
			return
		}
		_, err2 = z.Meta.Write(&buf)
		if err2 == nil {
			err2 = buf.WriteByte('\n')
		}
		if err2 == nil {
			_, err = z.Content.Write(&buf)
		}

	case partMeta:
		m, err2 := getMeta.Run(box.NoEnrichContext(ctx), zid)
		if err2 != nil {
			a.reportUsecaseError(w, err2)
			return
		}
		contentType = content.PlainText
		_, err = m.Write(&buf)

	case partContent:
		z, err2 := getZettel.Run(box.NoEnrichContext(ctx), zid)
		if err2 != nil {
			a.reportUsecaseError(w, err2)
			return
		}
		contentType = content.MIMEFromSyntax(z.Meta.GetDefault(api.KeySyntax, ""))
		_, err = z.Content.Write(&buf)
	}

	if err != nil {
		a.log.Fatal().Err(err).Zid(zid).Msg("Unable to store plain zettel/part in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = writeBuffer(w, &buf, contentType)
	a.log.IfErr(err).Zid(zid).Msg("Write Plain data")
}

func (a *API) writeJSONData(w http.ResponseWriter, ctx context.Context, zid id.Zid, part partType, getMeta usecase.GetMeta, getZettel usecase.GetZettel) {
	var buf bytes.Buffer
	var err error

	switch part {
	case partZettel:
		z, err2 := getZettel.Run(ctx, zid)
		if err2 != nil {
			a.reportUsecaseError(w, err2)
			return
		}
		zContent, encoding := z.Content.Encode()
		err = encodeJSONData(&buf, api.ZettelJSON{
			ID:       api.ZettelID(zid.String()),
			Meta:     z.Meta.Map(),
			Encoding: encoding,
			Content:  zContent,
			Rights:   a.getRights(ctx, z.Meta),
		})

	case partMeta:
		m, err2 := getMeta.Run(ctx, zid)
		if err2 != nil {
			a.reportUsecaseError(w, err2)
			return
		}
		err = encodeJSONData(&buf, api.MetaJSON{
			Meta:   m.Map(),
			Rights: a.getRights(ctx, m),
		})

	case partContent:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err != nil {
		a.log.Fatal().Err(err).Zid(zid).Msg("Unable to store zettel in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = writeBuffer(w, &buf, content.JSON)
	a.log.IfErr(err).Zid(zid).Msg("Write JSON data")
}

// MakeGetJSONZettelHandler creates a new HTTP handler to return a zettel.
func (a *API) MakeGetJSONZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		z, err := a.getZettelFromPath(ctx, w, r, getZettel)
		if err != nil {
			return
		}
		m := z.Meta

		var buf bytes.Buffer
		zContent, encoding := z.Content.Encode()
		err = encodeJSONData(&buf, api.ZettelJSON{
			ID:       api.ZettelID(m.Zid.String()),
			Meta:     m.Map(),
			Encoding: encoding,
			Content:  zContent,
			Rights:   a.getRights(ctx, m),
		})
		if err != nil {
			a.log.Fatal().Err(err).Zid(m.Zid).Msg("Unable to store zettel in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, content.JSON)
		a.log.IfErr(err).Zid(m.Zid).Msg("Write JSON Zettel")
	}
}

func (a *API) getZettelFromPath(ctx context.Context, w http.ResponseWriter, r *http.Request, getZettel usecase.GetZettel) (domain.Zettel, error) {
	zid, err := id.Parse(r.URL.Path[1:])
	if err != nil {
		http.NotFound(w, r)
		return domain.Zettel{}, err
	}

	z, err := getZettel.Run(ctx, zid)
	if err != nil {
		a.reportUsecaseError(w, err)
		return domain.Zettel{}, err
	}
	return z, nil
}

// MakeGetMetaHandler creates a new HTTP handler to return metadata of a zettel.
func (a *API) MakeGetMetaHandler(getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		m, err := getMeta.Run(ctx, zid)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		var buf bytes.Buffer
		err = encodeJSONData(&buf, api.MetaJSON{
			Meta:   m.Map(),
			Rights: a.getRights(ctx, m),
		})
		if err != nil {
			a.log.Fatal().Err(err).Zid(zid).Msg("Unable to store metadata in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, content.JSON)
		a.log.IfErr(err).Zid(zid).Msg("Write JSON Meta")
	}
}

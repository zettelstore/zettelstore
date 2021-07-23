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
	"errors"
	"fmt"
	"net/http"

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetZettelHandler creates a new HTTP handler to return a rendered zettel.
func (api *API) MakeGetZettelHandler(parseZettel usecase.ParseZettel, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()
		enc, _ := adapter.GetEncoding(r, q, encoder.GetDefaultEncoding())
		if enc == zsapi.EncoderRaw {
			ctx = box.NoEnrichContext(ctx)
		}
		zn, err := parseZettel.Run(ctx, zid, q.Get(meta.KeySyntax))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		part := getPart(q, partZettel)
		if part == partUnknown {
			adapter.BadRequest(w, "Unknown _part parameter")
			return
		}
		switch enc {
		case zsapi.EncoderJSON, zsapi.EncoderDJSON:
			w.Header().Set(zsapi.HeaderContentType, encoding2ContentType(enc))
			err = api.getWriteMetaZettelFunc(ctx, enc, part, partZettel, getMeta)(w, zn)
			if err != nil {
				adapter.InternalServerError(w, "Write D/JSON", err)
			}
			return
		}

		env := encoder.Environment{
			LinkAdapter:    adapter.MakeLinkAdapter(ctx, api, 'z', getMeta, part.DefString(partZettel), enc),
			ImageAdapter:   adapter.MakeImageAdapter(ctx, api, getMeta),
			CiteAdapter:    nil,
			Lang:           config.GetLang(zn.InhMeta, api.rtConfig),
			Xhtml:          false,
			MarkerExternal: "",
			NewWindow:      false,
			IgnoreMeta:     map[string]bool{meta.KeyLang: true},
		}
		switch part {
		case partZettel:
			err = writeZettelPartZettel(w, zn, enc, env)
		case partMeta:
			err = writeZettelPartMeta(w, zn, enc)
		case partContent:
			err = api.writeZettelPartContent(w, zn, enc, env)
		}
		if err != nil {
			if errors.Is(err, adapter.ErrNoSuchEncoding) {
				adapter.BadRequest(w, fmt.Sprintf("Zettel %q not available in encoding %q", zid.String(), enc))
				return
			}
			adapter.InternalServerError(w, "Get zettel", err)
		}
	}
}

func writeZettelPartZettel(w http.ResponseWriter, zn *ast.ZettelNode, enc zsapi.EncodingEnum, env encoder.Environment) error {
	encdr := encoder.Create(enc, &env)
	if encdr == nil {
		return adapter.ErrNoSuchEncoding
	}
	inhMeta := false
	if enc != zsapi.EncoderRaw {
		w.Header().Set(zsapi.HeaderContentType, encoding2ContentType(enc))
		inhMeta = true
	}
	_, err := encdr.WriteZettel(w, zn, inhMeta)
	return err
}

func writeZettelPartMeta(w http.ResponseWriter, zn *ast.ZettelNode, enc zsapi.EncodingEnum) error {
	w.Header().Set(zsapi.HeaderContentType, encoding2ContentType(enc))
	if encdr := encoder.Create(enc, nil); encdr != nil {
		if enc == zsapi.EncoderRaw {
			_, err := encdr.WriteMeta(w, zn.Meta)
			return err
		}
		_, err := encdr.WriteMeta(w, zn.InhMeta)
		return err
	}
	return adapter.ErrNoSuchEncoding
}

func (api *API) writeZettelPartContent(w http.ResponseWriter, zn *ast.ZettelNode, enc zsapi.EncodingEnum, env encoder.Environment) error {
	if enc == zsapi.EncoderRaw {
		if ct, ok := syntax2contentType(config.GetSyntax(zn.Meta, api.rtConfig)); ok {
			w.Header().Add(zsapi.HeaderContentType, ct)
		}
	} else {
		w.Header().Set(zsapi.HeaderContentType, encoding2ContentType(enc))
	}
	return writeContent(w, zn, enc, &env)
}

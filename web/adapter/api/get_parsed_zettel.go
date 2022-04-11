//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"bytes"
	"fmt"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetParsedZettelHandler creates a new HTTP handler to return a parsed zettel.
func (a *API) MakeGetParsedZettelHandler(parseZettel usecase.ParseZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		enc, encStr := getEncoding(r, q, encoder.GetDefaultEncoding())
		part := getPart(q, partContent)
		zn, err := parseZettel.Run(r.Context(), zid, q.Get(api.KeySyntax))
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}
		a.writeEncodedZettelPart(w, zn, parser.ParseMetadata, enc, encStr, part)
	}
}

func (a *API) writeEncodedZettelPart(
	w http.ResponseWriter, zn *ast.ZettelNode,
	evalMeta encoder.EvalMetaFunc,
	enc api.EncodingEnum, encStr string, part partType,
) {
	encdr := encoder.Create(enc)
	if encdr == nil {
		adapter.BadRequest(w, fmt.Sprintf("Zettel %q not available in encoding %q", zn.Meta.Zid.String(), encStr))
		return
	}
	var err error
	var buf bytes.Buffer
	switch part {
	case partZettel:
		_, err = encdr.WriteZettel(&buf, zn, evalMeta)
	case partMeta:
		_, err = encdr.WriteMeta(&buf, zn.InhMeta, evalMeta)
	case partContent:
		_, err = encdr.WriteContent(&buf, zn)
	}
	if err != nil {
		a.log.Fatal().Err(err).Zid(zn.Zid).Msg("Unable to store data in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if buf.Len() == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err = writeBuffer(w, &buf, encoding2ContentType(enc))
	a.log.IfErr(err).Zid(zn.Zid).Msg("Write Encoded Zettel")
}

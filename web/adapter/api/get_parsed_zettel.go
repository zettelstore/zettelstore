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
	"fmt"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
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
		enc, encStr := adapter.GetEncoding(r, q, encoder.GetDefaultEncoding())
		part := getPart(q, partContent)
		zn, err := parseZettel.Run(r.Context(), zid, q.Get(api.KeySyntax))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
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
	env := encoder.Environment{
		Lang:           config.GetLang(zn.InhMeta, a.rtConfig),
		Xhtml:          false,
		MarkerExternal: "",
		NewWindow:      false,
		IgnoreMeta:     map[string]bool{api.KeyLang: true},
	}
	encdr := encoder.Create(enc, &env)
	if encdr == nil {
		adapter.BadRequest(w, fmt.Sprintf("Zettel %q not available in encoding %q", zn.Meta.Zid.String(), encStr))
		return
	}
	w.Header().Set(api.HeaderContentType, encoding2ContentType(enc))
	var err error
	switch part {
	case partZettel:
		_, err = encdr.WriteZettel(w, zn, evalMeta)
	case partMeta:
		_, err = encdr.WriteMeta(w, zn.InhMeta, evalMeta)
	case partContent:
		_, err = encdr.WriteContent(w, zn)
	}
	if err != nil {
		adapter.InternalServerError(w, "Write encoded zettel", err)
	}
}

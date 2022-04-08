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
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/usecase"
)

// MakeGetEvalZettelHandler creates a new HTTP handler to return a evaluated zettel.
func (a *API) MakeGetEvalZettelHandler(evaluate usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()
		enc, encStr := getEncoding(r, q, encoder.GetDefaultEncoding())
		part := getPart(q, partContent)
		var env evaluator.Environment
		if q.Has(api.QueryKeyEmbed) {
			env.GetImageMaterial = func(zettel domain.Zettel, syntax string) ast.InlineEmbedNode {
				return &ast.EmbedBLOBNode{
					Blob:   zettel.Content.AsBytes(),
					Syntax: syntax,
				}
			}
		}
		zn, err := evaluate.Run(ctx, zid, q.Get(api.KeySyntax), &env)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}
		evalMeta := func(value string) ast.InlineSlice {
			return evaluate.RunMetadata(ctx, value, &env)
		}
		a.writeEncodedZettelPart(w, zn, evalMeta, enc, encStr, part)
	}
}

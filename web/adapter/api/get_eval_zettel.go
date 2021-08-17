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
	"net/http"

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetEvalZettelHandler creates a new HTTP handler to return a evaluated zettel.
func (api *API) MakeGetEvalZettelHandler(evaluateZettel usecase.EvaluateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()
		enc, encStr := adapter.GetEncoding(r, q, encoder.GetDefaultEncoding())
		part := getPart(q, partZettel)
		getTagRef := func(s string) *ast.Reference {
			return adapter.CreateTagReference(api, 'z', s)
		}
		getHostedRef := func(s string) *ast.Reference {
			return adapter.CreateHostedReference(api, s)
		}
		getFoundRef := func(zid id.Zid, fragment string) *ast.Reference {
			return adapter.CreateFoundReference(api, 'v', part.DefString(partZettel), encStr, zid, fragment)
		}
		getImageRef := func(zid id.Zid, state ast.RefState) *ast.Reference {
			return adapter.CreateImageReference(api, zid, state)
		}
		switch enc {
		case zsapi.EncoderHTML:
			// Get all references only for HTML encoding
		case zsapi.EncoderDJSON:
			// Get all references, except for tags.
			getTagRef = nil
		default:
			// Other encodings do not change the references
			getTagRef = nil
			getHostedRef = nil
			getFoundRef = nil
			getImageRef = nil
		}
		zn, err := evaluateZettel.Run(ctx, zid, &evaluator.Environment{
			Syntax:       q.Get(meta.KeySyntax),
			Config:       api.rtConfig,
			GetTagRef:    getTagRef,
			GetHostedRef: getHostedRef,
			GetFoundRef:  getFoundRef,
			GetImageRef:  getImageRef,
		})
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		api.writeEncodedZettelPart(w, zn, enc, encStr, part)
	}
}

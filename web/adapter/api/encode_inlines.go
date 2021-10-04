//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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
	"encoding/json"
	"net/http"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakePostEncodeInlinesHandler creates a new HTTP handler to encode given
// Zettelmarkup inline material
func (a *API) MakePostEncodeInlinesHandler(evaluate usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		var reqJSON api.EncodeInlineReqJSON
		if err := dec.Decode(&reqJSON); err != nil {
			http.Error(w, "Unable to read request", http.StatusBadRequest)
			return
		}
		envEnc := &encoder.Environment{
			Lang:        reqJSON.Lang,
			Interactive: reqJSON.NoLinks,
		}
		if envEnc.Lang == "" {
			envEnc.Lang = a.rtConfig.GetDefaultLang()
		}
		htmlEnc := encoder.Create(api.EncoderHTML, envEnc)
		ctx := r.Context()
		envEval := evaluator.Environment{}
		var respJSON api.EncodedInlineRespJSON
		if iln := evaluate.RunMetadata(ctx, reqJSON.FirstZmk, &envEval); iln != nil {
			s, err := encodeInlines(htmlEnc, iln)
			if err != nil {
				http.Error(w, "Unable to encode first as HTML", http.StatusBadRequest)
				return
			}
			respJSON.FirstHTML = s

			s, err = encodeInlines(encoder.Create(api.EncoderText, nil), iln)
			if err != nil {
				http.Error(w, "Unable to encode first as Text", http.StatusBadRequest)
				return
			}
			respJSON.FirstText = s
		}

		if reqLen := len(reqJSON.OtherZmk); reqLen > 0 {
			respJSON.OtherHTML = make([]string, reqLen)
			for i, zmk := range reqJSON.OtherZmk {
				iln := evaluate.RunMetadata(ctx, zmk, &envEval)
				if iln == nil {
					continue
				}
				s, err := encodeInlines(htmlEnc, iln)
				if err != nil {
					http.Error(w, "Unable to encode other as HTML", http.StatusBadRequest)
					return
				}
				respJSON.OtherHTML[i] = s
			}
		}

		w.Header().Set(api.HeaderContentType, ctJSON)
		err := encodeJSONData(w, respJSON)
		if err != nil {
			adapter.InternalServerError(w, "Write JSON for encoded Zettelmarkup", err)
		}
	}
}

func encodeInlines(encdr encoder.Encoder, inl *ast.InlineListNode) (string, error) {
	var content strings.Builder
	_, err := encdr.WriteInlines(&content, inl)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}
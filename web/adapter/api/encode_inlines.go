//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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
	"encoding/json"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/usecase"
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
		if iln := evaluate.RunMetadata(ctx, reqJSON.FirstZmk, &envEval); !iln.IsEmpty() {
			s, err := encodeInlines(htmlEnc, &iln)
			if err != nil {
				http.Error(w, "Unable to encode first as HTML", http.StatusBadRequest)
				return
			}
			respJSON.FirstHTML = s

			s, err = encodeInlines(encoder.Create(api.EncoderText, nil), &iln)
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
				if iln.IsEmpty() {
					continue
				}
				s, err := encodeInlines(htmlEnc, &iln)
				if err != nil {
					http.Error(w, "Unable to encode other as HTML", http.StatusBadRequest)
					return
				}
				respJSON.OtherHTML[i] = s
			}
		}

		var buf bytes.Buffer
		err := encodeJSONData(&buf, respJSON)
		if err != nil {
			a.log.Fatal().Err(err).Msg("Unable to store inlines in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, ctJSON)
		a.log.IfErr(err).Msg("Write JSON Inlines")
	}
}

func encodeInlines(encdr encoder.Encoder, inl *ast.InlineListNode) (string, error) {
	var buf bytes.Buffer
	_, err := encdr.WriteInlines(&buf, inl)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

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
	"zettelstore.de/z/collect"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetLinksHandler creates a new API handler to return links to other material.
func MakeGetLinksHandler(evaluate usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ctx := r.Context()
		q := r.URL.Query()
		zn, err := evaluate.Run(ctx, zid, q.Get(meta.KeySyntax), nil)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		summary := collect.References(zn)

		outData := zsapi.ZettelLinksJSON{ID: zid.String()}
		// TODO: calculate incoming links from other zettel (via "backward" metadata?)
		outData.Linked.Incoming = nil
		zetRefs, locRefs, extRefs := collect.DivideReferences(summary.Links)
		outData.Linked.Outgoing = idRefs(zetRefs)
		outData.Linked.Local = stringRefs(locRefs)
		outData.Linked.External = stringRefs(extRefs)
		for _, p := range zn.Meta.PairsRest(false) {
			if meta.Type(p.Key) == meta.TypeURL {
				outData.Linked.Meta = append(outData.Linked.Meta, p.Value)
			}
		}

		zetRefs, locRefs, extRefs = collect.DivideReferences(summary.Embeds)
		outData.Embedded.Outgoing = idRefs(zetRefs)
		outData.Embedded.Local = stringRefs(locRefs)
		outData.Embedded.External = stringRefs(extRefs)

		outData.Cites = stringCites(summary.Cites)

		w.Header().Set(zsapi.HeaderContentType, ctJSON)
		encodeJSONData(w, outData)
	}
}

func idRefs(refs []*ast.Reference) []string {
	result := make([]string, len(refs))
	for i, ref := range refs {
		path := ref.URL.Path
		if fragment := ref.URL.Fragment; len(fragment) > 0 {
			path = path + "#" + fragment
		}
		result[i] = path
	}
	return result
}

func stringRefs(refs []*ast.Reference) []string {
	result := make([]string, 0, len(refs))
	for _, ref := range refs {
		result = append(result, ref.String())
	}
	return result
}

func stringCites(cites []*ast.CiteNode) []string {
	mapKey := make(map[string]bool)
	result := make([]string, 0, len(cites))
	for _, cn := range cites {
		if _, ok := mapKey[cn.Key]; !ok {
			mapKey[cn.Key] = true
			result = append(result, cn.Key)
		}
	}
	return result
}

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
	"io"
	"net/http"
	"net/url"

	"zettelstore.de/c/api"
	"zettelstore.de/c/input"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// getEncoding returns the data encoding selected by the caller.
func getEncoding(r *http.Request, q url.Values, defEncoding api.EncodingEnum) (api.EncodingEnum, string) {
	encoding := q.Get(api.QueryKeyEncoding)
	if len(encoding) > 0 {
		return api.Encoder(encoding), encoding
	}
	if enc, ok := getOneEncoding(r, api.HeaderAccept); ok {
		return api.Encoder(enc), enc
	}
	if enc, ok := getOneEncoding(r, api.HeaderContentType); ok {
		return api.Encoder(enc), enc
	}
	return defEncoding, defEncoding.String()
}

func getOneEncoding(r *http.Request, key string) (string, bool) {
	if values, ok := r.Header[key]; ok {
		for _, value := range values {
			if enc, ok2 := contentType2encoding(value); ok2 {
				return enc, true
			}
		}
	}
	return "", false
}

var mapCT2encoding = map[string]string{
	"application/json": "json",
	"text/html":        api.EncodingHTML,
}

func contentType2encoding(contentType string) (string, bool) {
	// TODO: only check before first ';'
	enc, ok := mapCT2encoding[contentType]
	return enc, ok
}

type partType int

const (
	_ partType = iota
	partMeta
	partContent
	partZettel
)

var partMap = map[string]partType{
	api.PartMeta:    partMeta,
	api.PartContent: partContent,
	api.PartZettel:  partZettel,
}

func getPart(q url.Values, defPart partType) partType {
	if part, ok := partMap[q.Get(api.QueryKeyPart)]; ok {
		return part
	}
	return defPart
}

func (p partType) String() string {
	switch p {
	case partMeta:
		return "meta"
	case partContent:
		return "content"
	case partZettel:
		return "zettel"
	}
	return ""
}

func (p partType) DefString(defPart partType) string {
	if p == defPart {
		return ""
	}
	return p.String()
}

func buildZettelFromPlainData(r *http.Request, zid id.Zid) (domain.Zettel, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return domain.Zettel{}, err
	}
	inp := input.NewInput(b)
	m := meta.NewFromInput(zid, inp)
	return domain.Zettel{
		Meta:    m,
		Content: domain.NewContent(inp.Src[inp.Pos:]),
	}, nil

}

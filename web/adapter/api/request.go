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
	"io"
	"net/http"
	"net/url"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
)

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
	inp := input.NewInput(string(b))
	m := meta.NewFromInput(zid, inp)
	return domain.Zettel{
		Meta:    m,
		Content: domain.NewContent(inp.Src[inp.Pos:]),
	}, nil

}

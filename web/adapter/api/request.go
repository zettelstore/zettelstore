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
	"net/url"

	"zettelstore.de/z/api"
)

type partType int

const (
	partUnknown partType = iota
	partID
	partMeta
	partContent
	partZettel
)

var partMap = map[string]partType{
	api.PartID:      partID,
	api.PartMeta:    partMeta,
	api.PartContent: partContent,
	api.PartZettel:  partZettel,
}

func getPart(q url.Values, defPart partType) partType {
	p := q.Get(api.QueryKeyPart)
	if p == "" {
		return defPart
	}
	if part, ok := partMap[p]; ok {
		return part
	}
	return partUnknown
}

func (p partType) String() string {
	switch p {
	case partID:
		return "id"
	case partMeta:
		return "meta"
	case partContent:
		return "content"
	case partZettel:
		return "zettel"
	case partUnknown:
		return "unknown"
	}
	return ""
}

func (p partType) DefString(defPart partType) string {
	if p == defPart {
		return ""
	}
	return p.String()
}

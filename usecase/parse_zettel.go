//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the zettelstore.
package usecase

import (
	"context"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/parser"
)

// ParseZettel is the data for this use case.
type ParseZettel struct {
	getZettel GetZettel
}

// NewParseZettel creates a new use case.
func NewParseZettel(getZettel GetZettel) ParseZettel {
	return ParseZettel{getZettel: getZettel}
}

// Run executes the use case.
func (uc ParseZettel) Run(
	ctx context.Context, zid id.Zid, syntax string) (*ast.ZettelNode, error) {
	zettel, err := uc.getZettel.Run(ctx, zid)
	if err != nil {
		return nil, err
	}

	return parser.ParseZettel(zettel, syntax), nil
}

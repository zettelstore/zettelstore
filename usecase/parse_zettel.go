//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/parser"
)

// ParseZettel is the data for this use case.
type ParseZettel struct {
	rtConfig  config.Config
	getZettel GetZettel
}

// NewParseZettel creates a new use case.
func NewParseZettel(rtConfig config.Config, getZettel GetZettel) ParseZettel {
	return ParseZettel{rtConfig: rtConfig, getZettel: getZettel}
}

// Run executes the use case.
func (uc ParseZettel) Run(ctx context.Context, zid id.Zid, syntax string) (*ast.ZettelNode, error) {
	zettel, err := uc.getZettel.Run(ctx, zid)
	if err != nil {
		return nil, err
	}

	return parser.ParseZettel(ctx, zettel, syntax, uc.rtConfig), nil
}

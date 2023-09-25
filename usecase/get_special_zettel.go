//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// TagZettel is the usecase of retrieving a "tag zettel", i.e. a zettel that
// describes a given tag. A tag zettel must habe the tag's name in its title
// and must have a role=tag.

// TagZettelPort is the interface used by this use case.
type TagZettelPort interface {
	// GetZettel retrieves a specific zettel.
	GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error)
}

// TagZettel is the data for this use case.
type TagZettel struct {
	port  GetZettelPort
	query *Query
}

// NewTagZettel creates a new use case.
func NewTagZettel(port GetZettelPort, query *Query) TagZettel {
	return TagZettel{port: port, query: query}
}

// Run executes the use case.
func (uc TagZettel) Run(ctx context.Context, tag string) (zettel.Zettel, error) {
	tag = meta.NormalizeTag(tag)
	q := query.Parse(
		api.KeyTitle + api.SearchOperatorEqual + tag + " " +
			api.KeyRole + api.SearchOperatorHas + api.ValueRoleTag)
	ml, err := uc.query.Run(ctx, q)
	if err != nil {
		return zettel.Zettel{}, err
	}
	for _, m := range ml {
		z, errZ := uc.port.GetZettel(ctx, m.Zid)
		if errZ == nil {
			return z, nil
		}
	}
	return zettel.Zettel{}, ErrTagZettelNotFound{Tag: tag}
}

// ErrTagZettelNotFound is returned if a tag zettel was not found.
type ErrTagZettelNotFound struct{ Tag string }

func (etznf ErrTagZettelNotFound) Error() string { return "tag zettel not found: " + etznf.Tag }

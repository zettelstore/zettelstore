//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package index allows to search for metadata and content.
package index

import (
	"context"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

// Updater is used to update metadata by adding new properties.
type Updater interface {
	// Update computes additional properties and updates the given metadata.
	// It is typically called by zettel reading methods.
	Update(ctx context.Context, m *meta.Meta)
}

// Port contains all the used functions to access zettel to be indexed.
type Port interface {
	RegisterObserver(func(place.ChangeInfo))
	FetchZids(ctx context.Context) (map[id.Zid]bool, error)
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
}

// Indexer contains all the functions of an index.
type Indexer interface {
	Updater

	// Start the index. It will read all zettel and store index data for later retrieval.
	Start(Port)

	// Stop the index. No zettel are read any more, but the current index data
	// can stil be retrieved.
	Stop()
}

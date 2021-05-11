//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package store contains general index data for storing a zettel index.
package store

import (
	"context"
	"io"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
	"zettelstore.de/z/search"
)

// Stats records statistics about the store.
type Stats struct {
	// Zettel is the number of zettel managed by the indexer.
	Zettel int

	// Updates count the number of metadata updates.
	Updates uint64

	// Words count the different words stored in the store.
	Words uint64

	// Urls count the different URLs stored in the store.
	Urls uint64
}

// Store all relevant zettel data. There may be multiple implementations, i.e.
// memory-based, file-based, based on SQLite, ...
type Store interface {
	place.Enricher
	search.Selector

	// UpdateReferences for a specific zettel.
	// Returns set of zettel identifier that must also be checked for changes.
	UpdateReferences(context.Context, *ZettelIndex) id.Set

	// DeleteZettel removes index data for given zettel.
	// Returns set of zettel identifier that must also be checked for changes.
	DeleteZettel(context.Context, id.Zid) id.Set

	// ReadStats populates st with store statistics.
	ReadStats(st *Stats)

	// Write the content to a Writer.
	Write(io.Writer)
}

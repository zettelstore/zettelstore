//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package index contains general index data for a zettel index.
package index

import (
	"context"
	"io"
	"time"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/search"
)

// IndexerPort contains all the used functions to access zettel to be indexed.
type IndexerPort interface {
	RegisterObserver(change.Func)
	FetchZids(context.Context) (id.Set, error)
	GetMeta(context.Context, id.Zid) (*meta.Meta, error)
	GetZettel(context.Context, id.Zid) (domain.Zettel, error)
}

// Indexer contains all the functions of an index.
type Indexer interface {
	place.Enricher
	search.Selector

	// Start the index. It will read all zettel and store index data for later retrieval.
	Start(IndexerPort)

	// Stop the index. No zettel are read any more, but the current index data
	// can stil be retrieved.
	Stop()

	// ReadStats populates st with indexer statistics.
	ReadStats(st *IndexerStats)
}

// IndexerStats records statistics about the indexer.
type IndexerStats struct {
	// LastReload stores the timestamp when a full re-index was done.
	LastReload time.Time

	// IndexesSinceReload counts indexing a zettel since the full re-index.
	IndexesSinceReload uint64

	// DurLastIndex is the duration of the last index run. This could be a
	// full re-index or a re-index of a single zettel.
	DurLastIndex time.Duration

	// Store records statistics about the underlying index store.
	Store StoreStats
}

// StoreStats records statistics about the store.
type StoreStats struct {
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
	ReadStats(st *StoreStats)

	// Write the content to a Writer.
	Write(io.Writer)
}

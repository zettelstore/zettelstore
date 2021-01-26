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
	"time"

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
	FetchZids(context.Context) (map[id.Zid]bool, error)
	GetMeta(context.Context, id.Zid) (*meta.Meta, error)
	GetZettel(context.Context, id.Zid) (domain.Zettel, error)
}

// Indexer contains all the functions of an index.
type Indexer interface {
	Updater

	// Start the index. It will read all zettel and store index data for later retrieval.
	Start(Port)

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

// Store all relevant zettel data. There may be multiple implementations, i.e.
// memory-based, file-based, based on SQLite, ...
type Store interface {
	Updater

	// UpdateReferences for a specific zettel.
	UpdateReferences(context.Context, *ZettelIndex)

	// DeleteZettel removes index data for given zettel.
	DeleteZettel(context.Context, id.Zid)

	// ReadStats populates st with store statistics.
	ReadStats(st *StoreStats)
}

// StoreStats records statistics about the store.
type StoreStats struct {
	// Zettel is the number of zettel managed by the indexer.
	Zettel int

	// Updates count the number of metadata updates.
	Updates uint64
}

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

// Index contains all the functions of an index.
type Index interface {
	Updater

	// Start the index. It will read all zettel and store index data for later retrieval.
	Start(Port)

	// Stop the index. No zettel are read any more, but the current index data
	// can stil be retrieved.
	Stop()
}

type index struct {
	done chan struct{}
}

// New creates a new indexer.
func New() Index {
	return &index{}
}

func (idx *index) Start(p Port) {
	if idx.done != nil {
		panic("Index already started")
	}
	idx.done = make(chan struct{})
	go idx.indexer(p)
}

func (idx *index) Stop() {
	if idx.done == nil {
		panic("Index already stopped")
	}
	close(idx.done)
	idx.done = nil
}

// Update reads all properties in the index and updates the metadata.
func (idx *index) Update(ctx context.Context, m *meta.Meta) {
}

// indexer runs in the background and updates the index data structures.
func (idx *index) indexer(p Port) {
	// Something may panic. Ensure a running indexer.
	defer func() {
		if err := recover(); err != nil {
			go idx.indexer(p)
		}
	}()

	for {
		select {
		case _, ok := <-idx.done:
			if !ok {
				return
			}
		}
	}
}

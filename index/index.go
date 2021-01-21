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
	ar      *anterooms
	done    chan struct{}
	observe bool
}

// New creates a new indexer.
func New() Index {
	return &index{
		ar: newAnterooms(10),
	}
}

func (idx *index) observer(ci place.ChangeInfo) {
	switch ci.Reason {
	case place.OnReload:
		idx.ar.Reset()
	case place.OnUpdate:
		idx.ar.Enqueue(ci.Zid, true)
	case place.OnDelete:
		idx.ar.Enqueue(ci.Zid, false)
	}
}

func (idx *index) Start(p Port) {
	if idx.done != nil {
		panic("Index already started")
	}
	idx.done = make(chan struct{})
	if !idx.observe {
		p.RegisterObserver(idx.observer)
		idx.observe = true
	}
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

	ctx := context.Background()
	// TODO: add unique value to context and check that in idx.Update
	for {
		for {
			zid, val := idx.ar.Dequeue()
			if zid.IsValid() {
				if !val {
					idx.deleteZettel(zid)
					continue
				}

				zettel, err := p.GetZettel(ctx, zid)
				if err == nil {
					idx.updateZettel(ctx, zettel)
				}
				continue
			}

			if val == false {
				break
			}
			zids, err := p.FetchZids(ctx)
			if err == nil {
				idx.ar.Reload(nil, zids)
			}
		}

		time.Sleep(time.Second)
		select {
		case _, ok := <-idx.done:
			if !ok {
				return
			}
		default:
		}
	}
}

func (idx *index) updateZettel(ctx context.Context, zettel domain.Zettel) {
	// log.Println("INDX", "Update", zettel.Meta.Zid, zettel.Meta.GetDefault(meta.KeyTitle, "???"))
	// The following produces an import cycle:
	// --> parser --> config.runtime --> config.startup --> index --> parser -->
	// Solution is to put the implementation into a sub-package an leave the interfaces here.
	// zn := parser.ParseZettel(zettel, "")
	// time.Sleep(10 * time.Millisecond)
}

func (idx *index) deleteZettel(zid id.Zid) {
	// log.Println("INDX", "Delete", zid)
	// time.Sleep(10 * time.Millisecond)
}

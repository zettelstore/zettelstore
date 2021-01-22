//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package indexer allows to search for metadata and content.
package indexer

import (
	"context"
	"time"

	"zettelstore.de/z/collect"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
)

type indexer struct {
	ar      *anterooms
	done    chan struct{}
	observe bool
}

// New creates a new indexer.
func New() index.Indexer {
	return &indexer{
		ar: newAnterooms(10),
	}
}

func (idx *indexer) observer(ci place.ChangeInfo) {
	switch ci.Reason {
	case place.OnReload:
		idx.ar.Reset()
	case place.OnUpdate:
		idx.ar.Enqueue(ci.Zid, true)
	case place.OnDelete:
		idx.ar.Enqueue(ci.Zid, false)
	}
}

func (idx *indexer) Start(p index.Port) {
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

func (idx *indexer) Stop() {
	if idx.done == nil {
		panic("Index already stopped")
	}
	close(idx.done)
	idx.done = nil
}

// Update reads all properties in the index and updates the metadata.
func (idx *indexer) Update(ctx context.Context, m *meta.Meta) {
}

// indexer runs in the background and updates the index data structures.
func (idx *indexer) indexer(p index.Port) {
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
				if err != nil {
					// TODO: on some errors put the zid into a "try later" set
					continue
				}
				idx.updateZettel(ctx, zettel)
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

func (idx *indexer) updateZettel(ctx context.Context, zettel domain.Zettel) {
	// log.Println("INDX", "Update", zettel.Meta.Zid, zettel.Meta.GetDefault(meta.KeyTitle, "???"))
	for _, p := range zettel.Meta.PairsRest(false) {
		key := p.Key
		switch meta.KeyType(key) {
		case meta.TypeID:
		case meta.TypeIDSet:
		}
	}
	zn := parser.ParseZettel(zettel, "")
	collect.References(zn)
	// time.Sleep(10 * time.Millisecond)
}

func (idx *indexer) deleteZettel(zid id.Zid) {
	// log.Println("INDX", "Delete", zid)
	// time.Sleep(10 * time.Millisecond)
}

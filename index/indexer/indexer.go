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

	"zettelstore.de/z/ast"
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
	if _, ok := ctx.Value(ctxKey).(*ctxKeyType); ok {
		// Update is called indirectly via indexer
		// -> ignore this call, do not update meta data
		return
	}
}

type indexerPort interface {
	getMetaPort
	FetchZids(ctx context.Context) (map[id.Zid]bool, error)
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
}

type ctxKeyType struct{}

var ctxKey ctxKeyType

// indexer runs in the background and updates the index data structures.
func (idx *indexer) indexer(p indexerPort) {
	// Something may panic. Ensure a running indexer.
	defer func() {
		if err := recover(); err != nil {
			go idx.indexer(p)
		}
	}()

	ctx := context.WithValue(context.Background(), ctxKey, &ctxKey)
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
				idx.updateZettel(ctx, zettel, p)
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

type getMetaPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

func (idx *indexer) updateZettel(ctx context.Context, zettel domain.Zettel, p getMetaPort) {
	// log.Println("INDX", "Update", zettel.Meta.Zid, zettel.Meta.GetDefault(meta.KeyTitle, "???"))
	m := zettel.Meta
	ix := newIndexData(m.Zid)
	for _, pair := range m.PairsRest(false) {
		descr := meta.GetDescription(pair.Key)
		if descr.IsComputed() {
			continue
		}
		switch descr.Type {
		case meta.TypeID:
			updateValue(ctx, descr.Inverse, pair.Value, p, ix)
		case meta.TypeIDSet:
			for _, val := range meta.ListFromValue(pair.Value) {
				updateValue(ctx, descr.Inverse, val, p, ix)
			}
		}
	}
	zn := parser.ParseZettel(zettel, "")
	refs := collect.References(zn)
	updateReferences(ctx, refs.Links, p, ix)
	updateReferences(ctx, refs.Images, p, ix)
	// time.Sleep(10 * time.Millisecond)
}

func updateValue(ctx context.Context, inverse string, value string, p getMetaPort, ix *indexData) {
	zid, err := id.Parse(value)
	if err != nil {
		return
	}
	if _, err := p.GetMeta(ctx, zid); err != nil {
		ix.AddDeadlink(zid)
		return
	}
	if inverse == "" {
		ix.AddBacklink(zid)
		return
	}
	ix.AddLink(inverse, zid)
}

func updateReferences(ctx context.Context, refs []*ast.Reference, p getMetaPort, ix *indexData) {
	zrefs, _, _ := collect.DivideReferences(refs, false)
	for _, ref := range zrefs {
		updateReference(ctx, ref.Value, p, ix)
	}
}

func updateReference(ctx context.Context, value string, p getMetaPort, ix *indexData) {
	zid, err := id.Parse(value)
	if err != nil {
		return
	}
	if _, err := p.GetMeta(ctx, zid); err != nil {
		ix.AddDeadlink(zid)
		return
	}
	ix.AddBacklink(zid)
}

func (idx *indexer) deleteZettel(zid id.Zid) {
	// log.Println("INDX", "Delete", zid)
	// time.Sleep(10 * time.Millisecond)
}

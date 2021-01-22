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

	"zettelstore.de/z/domain/meta"
)

// Remover is used to remove some metadata before they are stored in a place.
type Remover interface {
	// Remove removes computed properties from the given metadata.
	// It is called by the manager place before meta data is updated.
	Remove(ctx context.Context, m *meta.Meta)
}

// MetaFilter is used by places to filter and set computed metadata value.
type MetaFilter interface {
	Updater
	Remover
}

type metaFilter struct {
	index      Indexer
	properties map[string]bool // Set of property key names
}

// NewMetaFilter creates a new meta filter.
func NewMetaFilter(idx Indexer) MetaFilter {
	properties := make(map[string]bool)
	for _, kd := range meta.GetSortedKeyDescriptions() {
		if kd.IsProperty() {
			properties[kd.Name] = true
		}
	}
	return &metaFilter{
		index:      idx,
		properties: properties,
	}
}

func (mf *metaFilter) Update(ctx context.Context, m *meta.Meta) {
	computePublished(m)
	mf.index.Update(ctx, m)
}

func computePublished(m *meta.Meta) {
	if _, ok := m.Get(meta.KeyPublished); ok {
		return
	}
	if modified, ok := m.Get(meta.KeyModified); ok {
		if _, ok = meta.TimeValue(modified); ok {
			m.Set(meta.KeyPublished, modified)
			return
		}
	}
	zid := m.Zid.String()
	if _, ok := meta.TimeValue(zid); ok {
		m.Set(meta.KeyPublished, zid)
		return
	}

	// Neither the zettel was modified nor the zettel identifer contains a valid
	// timestamp. In this case do not set the "published" property.
}

func (mf *metaFilter) Remove(ctx context.Context, m *meta.Meta) {
	for _, p := range m.PairsRest(true) {
		if mf.properties[p.Key] {
			m.Delete(p.Key)
		}
	}
}

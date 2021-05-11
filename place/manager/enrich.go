//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various places and indexes of a Zettelstore.
package manager

import (
	"context"

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager/index"
)

type metaEnricher struct {
	index index.Indexer
}

// NewEnricher creates a new meta filter.
func NewEnricher(idx index.Indexer) place.Enricher {
	return &metaEnricher{index: idx}
}

func (mf *metaEnricher) Enrich(ctx context.Context, m *meta.Meta) {
	computePublished(m)
	mf.index.Enrich(ctx, m)
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

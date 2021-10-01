//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various boxes and indexes of a Zettelstore.
package manager

import (
	"context"
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/meta"
)

// Enrich computes additional properties and updates the given metadata.
func (mgr *Manager) Enrich(ctx context.Context, m *meta.Meta, boxNumber int) {
	if box.DoNotEnrich(ctx) {
		// Enrich is called indirectly via indexer or enrichment is not requested
		// because of other reasons -> ignore this call, do not update meta data
		return
	}
	m.Set(api.KeyBoxNumber, strconv.Itoa(boxNumber))
	computePublished(m)
	mgr.idxStore.Enrich(ctx, m)
}

func computePublished(m *meta.Meta) {
	if _, ok := m.Get(api.KeyPublished); ok {
		return
	}
	if modified, ok := m.Get(api.KeyModified); ok {
		if _, ok = meta.TimeValue(modified); ok {
			m.Set(api.KeyPublished, modified)
			return
		}
	}
	zid := m.Zid.String()
	if _, ok := meta.TimeValue(zid); ok {
		m.Set(api.KeyPublished, zid)
		return
	}

	// Neither the zettel was modified nor the zettel identifer contains a valid
	// timestamp. In this case do not set the "published" property.
}

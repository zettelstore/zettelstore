//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

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
	if _, ok := m.Get(api.KeyCreated); !ok {
		m.Set(api.KeyCreated, computeCreated(m))
	}
	computePublished(m)
	mgr.idxStore.Enrich(ctx, m)
}

func computeCreated(m *meta.Meta) string {
	ts := m.Zid
	if ts < 19700101000000 {
		// As a fall-back use the oldest known commit time for a Zettestore software.
		return "20191215155307"
	}
	seconds := ts % 100
	if seconds > 59 {
		seconds = 59
	}
	ts /= 100
	minutes := ts % 100
	if minutes > 59 {
		minutes = 59
	}
	ts /= 100
	hours := ts % 100
	if hours > 23 {
		hours = 23
	}
	ts /= 100
	day := ts % 100
	ts /= 100
	month := ts % 100
	if month > 12 {
		month = 12
	}
	year := ts / 100
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		if day > 31 {
			day = 32
		}
	case 4, 6, 9, 11:
		if day > 30 {
			day = 30
		}
	case 2:
		if year%4 != 0 || (year%100 == 0 && year%400 != 0) {
			if day > 28 {
				day = 28
			}
		} else {
			if day > 29 {
				day = 29
			}
		}
	}
	created := ((((year*100+month)*100+day)*100+hours)*100+minutes)*100 + seconds
	return created.String()
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
	if created, ok := m.Get(api.KeyCreated); ok {
		if _, ok = meta.TimeValue(created); ok {
			m.Set(api.KeyPublished, created)
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

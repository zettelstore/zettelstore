//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various places of a Zettelstore.
package manager

import "zettelstore.de/z/domain/meta"

// MetaFilter is used by places to filter and set computed metadata value.
type MetaFilter interface {
	// UpdateProperties computes additional properties and updates the given metadata.
	// It is typically called by zettel reading methods.
	UpdateProperties(m *meta.Meta)

	// RemoveProperties removes computed properties from the given metadata.
	// It is called by the manager place before meta data is updated.
	RemoveProperties(m *meta.Meta)
}

type metaFilter struct {
	properties map[string]bool // Set of property key names
}

func newFilter() MetaFilter {
	properties := make(map[string]bool)
	for _, kd := range meta.GetSortedKeyDescriptions() {
		if kd.IsProperty() {
			properties[kd.Name] = true
		}
	}
	return &metaFilter{
		properties: properties,
	}
}

func (mf *metaFilter) UpdateProperties(m *meta.Meta) {
	computePublished(m)
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

func (mf *metaFilter) RemoveProperties(m *meta.Meta) {
	for _, p := range m.PairsRest(true) {
		if mf.properties[p.Key] {
			m.Delete(p.Key)
		}
	}
}

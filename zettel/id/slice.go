//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package id

import (
	"slices"
	"strings"
)

// Slice is a sequence of zettel identifier. A special case is a sorted slice.
type Slice []Zid

// Sort a slice of Zids.
func (zs Slice) Sort() { slices.Sort(zs) }

// Clone a zettel identifier slice
func (zs Slice) Clone() Slice { return slices.Clone(zs) }

// Equal reports whether zs and other are the same length and contain the samle zettel
// identifier. A nil argument is equivalent to an empty slice.
func (zs Slice) Equal(other Slice) bool { return slices.Equal(zs, other) }

func (zs Slice) String() string {
	if len(zs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, zid := range zs {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(zid.String())
	}
	return sb.String()
}

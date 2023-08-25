//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package id

import "slices"

// Edge is a pair of to vertices.
type Edge struct {
	From, To Zid
}

// EdgeSlice is a slice of Edges
type EdgeSlice []Edge

// Equal return true if both slices are the same.
func (es EdgeSlice) Equal(other EdgeSlice) bool {
	return slices.Equal(es, other)
}

// Sort the slice.
func (es EdgeSlice) Sort() EdgeSlice {
	slices.SortFunc(es, func(e1, e2 Edge) int {
		if e1.From < e2.From {
			return -1
		}
		if e1.From > e2.From {
			return 1
		}
		if e1.To < e2.To {
			return -1
		}
		if e1.To > e2.To {
			return 1
		}
		return 0
	})
	return es
}

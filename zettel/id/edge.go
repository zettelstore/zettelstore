//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2023-present Detlef Stern
//-----------------------------------------------------------------------------

package id

import "slices"

// EdgeO is a pair of to vertices.
type EdgeO struct {
	From, To ZidO
}

// EdgeSliceO is a slice of Edges
type EdgeSliceO []EdgeO

// Equal return true if both slices are the same.
func (es EdgeSliceO) Equal(other EdgeSliceO) bool {
	return slices.Equal(es, other)
}

// Sort the slice.
func (es EdgeSliceO) Sort() EdgeSliceO {
	slices.SortFunc(es, func(e1, e2 EdgeO) int {
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

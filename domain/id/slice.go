//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package id provides domain specific types, constants, and functions about
// zettel identifier.
package id

import (
	"sort"
	"strings"
)

// Slice is a sequence of zettel identifier. A special case is a sorted slice.
type Slice []Zid

func (zs Slice) Len() int           { return len(zs) }
func (zs Slice) Less(i, j int) bool { return zs[i] < zs[j] }
func (zs Slice) Swap(i, j int)      { zs[i], zs[j] = zs[j], zs[i] }

// Sort a slice of Zids.
func (zs Slice) Sort() { sort.Sort(zs) }

// Copy a zettel identifier slice
func (zs Slice) Copy() Slice {
	if zs == nil {
		return nil
	}
	result := make(Slice, len(zs))
	copy(result, zs)
	return result
}

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

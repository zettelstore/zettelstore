//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

package id

import (
	"slices"
	"strings"
)

// SliceO is a sequence of zettel identifier. A special case is a sorted slice.
type SliceO []ZidO

// Sort a slice of Zids.
func (zs SliceO) Sort() { slices.Sort(zs) }

// Clone a zettel identifier slice
func (zs SliceO) Clone() SliceO { return slices.Clone(zs) }

// Equal reports whether zs and other are the same length and contain the samle zettel
// identifier. A nil argument is equivalent to an empty slice.
func (zs SliceO) Equal(other SliceO) bool { return slices.Equal(zs, other) }

func (zs SliceO) String() string {
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

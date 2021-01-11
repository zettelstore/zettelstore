//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package domain provides domain specific types, constants, and functions.
package domain

import (
	"zettelstore.de/z/domain/meta"
)

// Zettel is the main data object of a zettelstore.
type Zettel struct {
	Meta    *meta.Meta // Some additional meta-data.
	Content Content    // The content of the zettel itself.
}

// Equal compares two zettel for equality.
func (z Zettel) Equal(o Zettel, allowComputed bool) bool {
	return z.Meta.Equal(o.Meta, allowComputed) && z.Content == o.Content
}

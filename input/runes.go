//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package input provides an abstraction for data to be read.
package input

// IsSpace returns true if rune is a whitespace.
func IsSpace(ch rune) bool {
	switch ch {
	case ' ', '\t':
		return true
	}
	return false
}

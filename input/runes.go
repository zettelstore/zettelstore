//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package input

import "unicode"

// IsSpace returns true if rune is a whitespace.
func IsSpace(ch rune) bool {
	switch ch {
	case ' ', '\t':
		return true
	case '\n', '\r', EOS:
		return false
	}
	return unicode.IsSpace(ch)
}

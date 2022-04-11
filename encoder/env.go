//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package encoder

import "zettelstore.de/z/strfun"

// Environment specifies all data and functions that affects encoding.
type Environment struct {
	// Important for HTML encoder
	Lang       string // default language
	IgnoreMeta strfun.Set
}

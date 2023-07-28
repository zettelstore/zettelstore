//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package query

import "zettelstore.de/client.fossil/api"

// IdentSpec contains all specification values to calculate the ident directive.
type IdentSpec struct{}

func (spec *IdentSpec) Print(pe *PrintEnv) {
	pe.printSpace()
	pe.writeString(api.IdentDirective)
}

// ItemsSpec contains all specification values to calculate items.
type ItemsSpec struct{}

func (spec *ItemsSpec) Print(pe *PrintEnv) {
	pe.printSpace()
	pe.writeString(api.ItemsDirective)
}

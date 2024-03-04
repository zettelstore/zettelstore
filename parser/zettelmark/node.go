//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

package zettelmark

import "zettelstore.de/z/ast"

// Internal nodes for parsing zettelmark. These will be removed in
// post-processing.

// nullItemNode specifies a removable placeholder for an item node.
type nullItemNode struct {
	ast.ItemNode
}

// nullDescriptionNode specifies a removable placeholder.
type nullDescriptionNode struct {
	ast.DescriptionNode
}

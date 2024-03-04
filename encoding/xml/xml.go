//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2022-present Detlef Stern
//-----------------------------------------------------------------------------

// Package xml provides helper for a XML-based encoding.
package xml

import (
	"bytes"

	"zettelstore.de/z/strfun"
)

// Header contains the string that should start all XML documents.
const Header = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

// WriteTag writes a simple XML tag with a given prefix and a specific value.
func WriteTag(buf *bytes.Buffer, prefix, tag, value string) {
	buf.WriteString(prefix)
	buf.WriteByte('<')
	buf.WriteString(tag)
	buf.WriteByte('>')
	strfun.XMLEscape(buf, value)
	buf.WriteString("</")
	buf.WriteString(tag)
	buf.WriteString(">\n")
}

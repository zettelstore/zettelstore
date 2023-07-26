//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package compbox

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func genParserM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Supported Parser")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreVTime).(string))
	m.Set(api.KeyVisibility, api.ValueVisibilityLogin)
	return m
}

func genParserC(*meta.Meta) []byte {
	var buf bytes.Buffer
	buf.WriteString("|=Syntax<|=Alt. Value(s):|=Text Parser?:|=Text Format?:|=Image Format?:\n")
	syntaxes := parser.GetSyntaxes()
	sort.Strings(syntaxes)
	for _, syntax := range syntaxes {
		info := parser.Get(syntax)
		if info.Name != syntax {
			continue
		}
		altNames := info.AltNames
		sort.Strings(altNames)
		fmt.Fprintf(
			&buf, "|%v|%v|%v|%v|%v\n",
			syntax, strings.Join(altNames, ", "), info.IsASTParser, info.IsTextFormat, info.IsImageFormat)
	}
	return buf.Bytes()
}

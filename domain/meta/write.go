//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta provides the domain specific type 'meta'.
package meta

import (
	"bytes"
	"io"
)

// Write writes a zettel meta to a writer.
func (m *Meta) Write(w io.Writer, allowComputed bool) (int, error) {
	var buf bytes.Buffer
	for _, p := range m.Pairs(allowComputed) {
		buf.WriteString(p.Key)
		buf.WriteString(": ")
		buf.WriteString(p.Value)
		buf.WriteByte('\n')
	}
	return w.Write(buf.Bytes())
}

var (
	newline = []byte{'\n'}
	yamlSep = []byte{'-', '-', '-', '\n'}
)

// WriteAsHeader writes the zettel meta to the writer, plus the separators
func (m *Meta) WriteAsHeader(w io.Writer, allowComputed bool) (int, error) {
	var lb, lc, la int
	var err error

	if m.YamlSep {
		lb, err = w.Write(yamlSep)
		if err != nil {
			return lb, err
		}
	}
	lc, err = m.Write(w, allowComputed)
	if err != nil {
		return lb + lc, err
	}
	if m.YamlSep {
		la, err = w.Write(yamlSep)
	} else {
		la, err = w.Write(newline)
	}
	return lb + lc + la, err
}

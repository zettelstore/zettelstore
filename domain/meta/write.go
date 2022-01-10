//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
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

// Write writes metadata to a writer. If "allowComputed" is true, then
// computed values are also written, but not property values.
func (m *Meta) Write(w io.Writer, allowComputed bool) (int, error) {
	var buf bytes.Buffer
	for _, p := range m.Pairs(true) {
		key := p.Key
		kd := GetDescription(key)
		if allowComputed {
			if kd.IsProperty() {
				continue
			}
		} else {
			if kd.IsComputed() {
				continue
			}
		}
		buf.WriteString(key)
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

// WriteAsHeader writes metadata to the writer, plus the separators.
// If "allowComputed" is true, then // computed values are also written, but not
// property values.
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

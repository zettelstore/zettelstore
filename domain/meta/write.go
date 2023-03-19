//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package meta

import "io"

// Write writes metadata to a writer, excluding computed and propery values.
func (m *Meta) Write(w io.Writer) (int, error) {
	return m.doWrite(w, IsComputed)
}

// WriteComputed writes metadata to a writer, including computed values,
// but excluding property values.
func (m *Meta) WriteComputed(w io.Writer) (int, error) {
	return m.doWrite(w, IsProperty)
}

func (m *Meta) doWrite(w io.Writer, ignoreKeyPred func(string) bool) (length int, err error) {
	for _, p := range m.ComputedPairs() {
		key := p.Key
		if ignoreKeyPred(key) {
			continue
		}
		if err != nil {
			break
		}
		var l int
		l, err = io.WriteString(w, key)
		length += l
		if err == nil {
			l, err = w.Write(colonSpace)
			length += l
		}
		if err == nil {
			l, err = io.WriteString(w, p.Value)
			length += l
		}
		if err == nil {
			l, err = w.Write(newline)
			length += l
		}
	}
	return length, err
}

var (
	colonSpace = []byte{':', ' '}
	newline    = []byte{'\n'}
)

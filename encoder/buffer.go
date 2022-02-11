//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package encoder

import (
	"encoding/base64"
	"io"
)

// BufWriter is a specialized buffered writer for encoding zettel.
type BufWriter struct {
	w      io.Writer // The io.Writer to write to
	err    error     // Collect error
	length int       // Collected length
}

// NewBufWriter creates a new BufWriter
func NewBufWriter(w io.Writer) BufWriter {
	return BufWriter{w: w}
}

// Write writes the contents of p into the buffer.
func (w *BufWriter) Write(p []byte) (l int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	l, w.err = w.w.Write(p)
	w.length += l
	return l, w.err
}

// WriteString writes the contents of s into the buffer.
func (w *BufWriter) WriteString(s string) {
	if w.err != nil {
		return
	}
	var l int
	l, w.err = io.WriteString(w.w, s)
	w.length += l
}

// WriteStrings writes the contents of sl into the buffer.
func (w *BufWriter) WriteStrings(sl ...string) {
	for _, s := range sl {
		w.WriteString(s)
	}
}

// WriteByte writes the content of b into the buffer.
func (w *BufWriter) WriteByte(b byte) error {
	var l int
	l, err := w.Write([]byte{b})
	w.length += l
	return err
}

// WriteBytes writes the content of bs into the buffer.
func (w *BufWriter) WriteBytes(bs ...byte) {
	w.Write(bs)
}

// WriteBase64 writes the content of p into the buffer, encoded with base64.
func (w *BufWriter) WriteBase64(p []byte) {
	if w.err == nil {
		encoder := base64.NewEncoder(base64.StdEncoding, w.w)
		var l int
		l, w.err = encoder.Write(p)
		w.length += l
		err1 := encoder.Close()
		if w.err == nil {
			w.err = err1
		}
	}
}

// Flush returns the collected length and error.
func (w *BufWriter) Flush() (int, error) { return w.length, w.err }

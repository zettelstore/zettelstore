//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package logger

import (
	"io"
	"sync"
	"time"
)

// LogWriter writes log messages to their specified destinations.
type LogWriter interface {
	WriteMessage(level Level, ts time.Time, prefix string, msg string, details []byte) error
}

// LogWriterAdapter adapts an io.Writer to a LogWriter
type LogWriterAdapter struct {
	w   io.Writer
	mx  sync.Mutex // protects buf and serializes w.Write
	buf []byte
}

// NewLogWriterAdapter creates a new LogWriter from an io.Writer.
func NewLogWriterAdapter(w io.Writer) *LogWriterAdapter {
	return &LogWriterAdapter{
		w:   w,
		buf: make([]byte, 0, 500),
	}
}

var eol = []byte{'\n'}

func (lwa *LogWriterAdapter) WriteMessage(level Level, ts time.Time, prefix string, msg string, details []byte) error {
	year, month, day := ts.Date()
	hour, minute, second := ts.Clock()

	lwa.mx.Lock()
	lwa.buf = lwa.buf[:0]
	buf := lwa.buf
	itoa(&buf, year, 4)
	buf = append(buf, '-')
	itoa(&buf, int(month), 2)
	buf = append(buf, '-')
	itoa(&buf, day, 2)
	buf = append(buf, ' ')
	itoa(&buf, hour, 2)
	buf = append(buf, ':')
	itoa(&buf, minute, 2)
	buf = append(buf, ':')
	itoa(&buf, second, 2)
	buf = append(buf, ' ')
	buf = append(buf, logLevel[level]...)
	buf = append(buf, ' ')
	if prefix != "" {
		buf = append(buf, prefix...)
		buf = append(buf, ' ')
	}
	buf = append(buf, msg...)
	write := lwa.w.Write
	_, err := write(buf)
	if len(details) > 0 && err == nil {
		_, err = write(details)
	}
	if err == nil {
		write(eol)
	}
	lwa.mx.Unlock()
	return err

}

func itoa(buf *[]byte, i, wid int) {
	var b [20]byte
	for bp := wid - 1; bp >= 0; bp-- {
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		i = q
	}
	*buf = append(*buf, b[:wid]...)
}

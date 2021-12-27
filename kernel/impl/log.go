//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package impl

import (
	"os"
	"sync"
	"time"

	"zettelstore.de/z/logger"
)

// kernelLogWriter adapts an io.Writer to a LogWriter
type kernelLogWriter struct {
	mx  sync.Mutex // protects buf and serializes w.Write
	buf []byte
}

// newKernelLogWriter creates a new LogWriter for kernel logging.
func newKernelLogWriter() *kernelLogWriter {
	return &kernelLogWriter{
		buf: make([]byte, 0, 500),
	}
}

func (lwa *kernelLogWriter) WriteMessage(level logger.Level, ts time.Time, prefix, msg string, details []byte) error {
	lwa.mx.Lock()
	lwa.buf = lwa.buf[:0]
	buf := lwa.buf
	addTimestamp(&buf, ts)
	buf = append(buf, ' ')
	buf = append(buf, level.Format()...)
	buf = append(buf, ' ')
	if prefix != "" {
		buf = append(buf, prefix...)
		buf = append(buf, ' ')
	}
	buf = append(buf, msg...)
	buf = append(buf, details...)
	buf = append(buf, '\n')
	_, err := os.Stdout.Write(buf)
	lwa.mx.Unlock()
	if level == logger.PanicLevel {
		panic(err)
	}
	return err
}

func addTimestamp(buf *[]byte, ts time.Time) {
	year, month, day := ts.Date()
	itoa(buf, year, 4)
	*buf = append(*buf, '-')
	itoa(buf, int(month), 2)
	*buf = append(*buf, '-')
	itoa(buf, day, 2)
	*buf = append(*buf, ' ')
	hour, minute, second := ts.Clock()
	itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	itoa(buf, minute, 2)
	*buf = append(*buf, ':')
	itoa(buf, second, 2)

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

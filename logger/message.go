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
	"sync"

	"zettelstore.de/z/domain/id"
)

// Message presents a message to log.
type Message struct {
	logger *Logger
	level  Level
	buf    []byte
}

func newMessage(logger *Logger, level Level) *Message {
	if logger != nil {
		realLogger := logger
		for realLogger.parent != nil {
			realLogger = realLogger.parent
		}
		if realLogger.Level() <= level {
			m := messagePool.Get().(*Message)
			m.logger = realLogger
			m.level = level
			m.buf = append(m.buf[:0], logger.context...)
			return m
		}
	}
	return nil
}

func recycleMessage(m *Message) {
	messagePool.Put(m)
}

var messagePool = &sync.Pool{
	New: func() interface{} {
		return &Message{
			buf: make([]byte, 0, 500),
		}
	},
}

// Enabled returns whether the message will log or not.
func (m *Message) Enabled() bool {
	return m != nil && m.level != NeverLevel
}

// Str adds a string value to the full message
func (m *Message) Str(text, val string) *Message {
	if m.Enabled() {
		buf := append(m.buf, ',', ' ')
		buf = append(buf, text...)
		buf = append(buf, '=')
		m.buf = append(buf, val...)
	}
	return m
}

// Bytes adds a byte slice value to the full message
func (m *Message) Bytes(text string, val []byte) *Message {
	if m.Enabled() {
		buf := append(m.buf, ',', ' ')
		buf = append(buf, text...)
		buf = append(buf, '=')
		m.buf = append(buf, val...)
	}
	return m
}

// Err adds an error value to the full message
func (m *Message) Err(err error) *Message {
	if err != nil {
		return m.Str("error", err.Error())
	}
	return m
}

// Zid adds a zettel identifier to the full message
func (m *Message) Zid(zid id.Zid) *Message {
	return m.Bytes("zid", zid.Bytes())
}

// Msg add the given text to the message and writes it to the log.
func (m *Message) Msg(text string) {
	if m.Enabled() {
		m.logger.writeMessage(m.level, text, m.buf)
		recycleMessage(m)
	}
}

// Child creates a child logger with context of this message.
func (m *Message) Child() *Logger {
	return newFromMessage(m)
}

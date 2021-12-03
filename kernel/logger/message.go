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

// Message presents a message to log.
type Message struct {
	logger *Logger
	level  Level
	buf    []byte
}

func newMessage(logger *Logger, level Level) *Message {
	if logger != nil && logger.Level() <= level {
		return &Message{
			logger: logger,
			level:  level,
			buf:    make([]byte, 0, 500),
		}
	}
	return nil
}

// Enabled returns whether the message will log or not.
func (m *Message) Enabled() bool {
	return m != nil && m.level != DisabledLevel
}

// Str adds a string value to the full message
func (m *Message) Str(text, val string) *Message {
	if m != nil {
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

// Msg add the given text to the message and writes it to the log.
func (m *Message) Msg(text string) {
	if m == nil {
		return
	}
	m.write(text)
}

func (m *Message) write(text string) {
	m.buf = append(m.buf, '\n')
	prefix := m.logger.prefix
	level := logLevel[m.level][:]

	// Ensure m.buf is big enough.
	bufLen := len(m.buf)
	neededCap := len(level) + bufLen
	if l := len(prefix); l > 0 {
		neededCap += l
		neededCap++
	}
	if l := len(text); l > 0 {
		neededCap += l
		neededCap++
	}
	room := neededCap - bufLen

	if neededCap > cap(m.buf) {
		buf := growBuf(make([]byte, 0, neededCap), room)
		m.buf = append(buf, m.buf...)
	} else {
		m.buf = growBuf(m.buf, room)
		if bufLen > 0 {
			copy(m.buf[room:], m.buf[:bufLen])
		}
	}

	c := copy(m.buf[0:], level)
	if len(prefix) > 0 {
		m.buf[c] = ' '
		d := copy(m.buf[c+1:], prefix)
		c = c + 1 + d
	}
	if text != "" {
		m.buf[c] = ' '
		copy(m.buf[c+1:], text)
	}
	m.logger.Write(m.buf)
}

const spaces = "                                                               "

func growBuf(buf []byte, n int) []byte {
	toAppend := n
	for toAppend >= len(spaces) {
		buf = append(buf, spaces...)
		toAppend -= len(spaces)
	}
	return append(buf, spaces[:toAppend]...)
}

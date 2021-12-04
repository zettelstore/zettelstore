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

import "time"

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
	const datetimeLen = 20
	neededCap := datetimeLen + len(level) + bufLen
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

	now := time.Now()
	year, month, day := now.Date()
	itoa(m.buf, year, 4)
	m.buf[4] = '-'
	itoa(m.buf[5:], int(month), 2)
	m.buf[7] = '-'
	itoa(m.buf[8:], day, 2)
	m.buf[10] = ' '
	hour, minute, second := now.Clock()
	itoa(m.buf[11:], hour, 2)
	m.buf[13] = ':'
	itoa(m.buf[14:], minute, 2)
	m.buf[16] = ':'
	itoa(m.buf[17:], second, 2)
	m.buf[19] = ' '
	c := datetimeLen + copy(m.buf[datetimeLen:], level)
	if len(prefix) > 0 {
		m.buf[c] = ' '
		c++
		c += copy(m.buf[c:], prefix)
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

func itoa(buf []byte, i, wid int) {
	for bp := wid - 1; bp >= 0; bp-- {
		q := i / 10
		buf[bp] = byte('0' + i - q*10)
		i = q
	}
}

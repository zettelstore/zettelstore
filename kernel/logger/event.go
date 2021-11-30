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
	text   string
	pairs  []strPair
}

type strPair struct{ text, val string }

func newMessage(logger *Logger, level Level) *Message {
	if logger != nil && logger.GetLevel() <= level {
		return &Message{
			logger: logger,
			level:  level,
		}
	}
	return nil
}

// Msg add the given text to the message and writes it to the log.
func (m *Message) Msg(text string) {
	if m == nil {
		return
	}
	m.text = text
	m.logger.write(m)
}

// Str adds a string value to the full message
func (m *Message) Str(text, val string) *Message {
	if m != nil {
		m.pairs = append(m.pairs, strPair{text, val})
	}
	return m
}

// Err adds an error value to the full message
func (m *Message) Err(text string, err error) *Message {
	if err != nil {
		return m.Str(text, err.Error())
	}
	if text != "" {
		return m.Str(text, "nil")
	}
	return m
}

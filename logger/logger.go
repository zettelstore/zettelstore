//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package logger implements a logging package for use in the Zettelstore.
package logger

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Level defines the possible log levels
type Level uint8

// Constants for Level
const (
	noLevel        Level = iota // the absent log level
	TraceLevel                  // Log most internal activities
	DebugLevel                  // Log most data updates
	InfoLevel                   // Log normal activities
	WarnLevel                   // Log event that can be easily recovered
	ErrorLevel                  // Log (persistent) errors
	FatalLevel                  // Log event that cannot be recovered within an internal acitivty
	PanicLevel                  // Log event that must stop the software
	MandatoryLevel              // Log only mandatory events
	NeverLevel                  // Logging is disabled
)

var logLevel = [...]string{
	"     ",
	"TRACE",
	"DEBUG",
	"INFO ",
	"WARN ",
	"ERROR",
	"FATAL",
	"PANIC",
	">>>>>",
	"NEVER",
}

var strLevel = [...]string{
	"",
	"trace",
	"debug",
	"info",
	"warn",
	"error",
	"fatal",
	"panic",
	"mandatory",
	"disabled",
}

// IsValid returns true, if the level is a valid level
func (l Level) IsValid() bool { return TraceLevel <= l && l <= NeverLevel }

func (l Level) String() string {
	if l.IsValid() {
		return strLevel[l]
	}
	return strconv.Itoa(int(l))
}

// Format returns a string representation suitable for logging.
func (l Level) Format() string {
	if l.IsValid() {
		return logLevel[l]
	}
	return strconv.Itoa(int(l))
}

// ParseLevel returns the recognized level.
func ParseLevel(text string) Level {
	for lv := TraceLevel; lv <= NeverLevel; lv++ {
		if len(text) > 2 && strings.HasPrefix(strLevel[lv], text) {
			return lv
		}
	}
	return noLevel
}

// Logger represents an objects that emits logging messages.
type Logger struct {
	lw       LogWriter
	levelVal uint32
	prefix   string
	parent   *Logger
	context  []byte
}

// LogWriter writes log messages to their specified destinations.
type LogWriter interface {
	WriteMessage(level Level, ts time.Time, prefix, msg string, details []byte) error
}

// New creates a new logger for the given service.
//
// This function must only be called from a kernel implementation, not from
// code that tries to log something.
func New(lw LogWriter, prefix string) *Logger {
	if prefix != "" && len(prefix) < 6 {
		prefix = (prefix + "     ")[:6]
	}
	return &Logger{
		lw:       lw,
		levelVal: uint32(InfoLevel),
		prefix:   prefix,
		parent:   nil,
		context:  nil,
	}
}

func newFromMessage(msg *Message) *Logger {
	if msg == nil {
		return nil
	}
	parent := msg.logger
	context := make([]byte, 0, len(msg.buf))
	context = append(context, msg.buf...)
	return &Logger{
		lw:       nil,
		levelVal: 0,
		prefix:   "",
		parent:   parent,
		context:  context,
	}
}

// SetLevel sets the level of the logger.
func (l *Logger) SetLevel(newLevel Level) *Logger {
	if l != nil {
		if l.parent != nil {
			panic("try to set level for child logger")
		}
		atomic.StoreUint32(&l.levelVal, uint32(newLevel))
	}
	return l
}

// Level returns the current level of the given logger
func (l *Logger) Level() Level {
	if l != nil {
		return Level(atomic.LoadUint32(&l.levelVal))
	}
	return NeverLevel
}

// Trace creates a tracing message.
func (l *Logger) Trace() *Message { return newMessage(l, TraceLevel) }

// Debug creates a debug message.
func (l *Logger) Debug() *Message { return newMessage(l, DebugLevel) }

// Info creates a message suitable for information data.
func (l *Logger) Info() *Message { return newMessage(l, InfoLevel) }

// Warn creates a message suitable for warning the user.
func (l *Logger) Warn() *Message { return newMessage(l, WarnLevel) }

// Error creates a message suitable for errors.
func (l *Logger) Error() *Message { return newMessage(l, ErrorLevel) }

// IfErr creates an error message and sets the go error, if there is an error.
func (l *Logger) IfErr(err error) *Message {
	if err != nil {
		return newMessage(l, ErrorLevel).Err(err)
	}
	return nil
}

// Fatal creates a message suitable for fatal errors.
func (l *Logger) Fatal() *Message { return newMessage(l, FatalLevel) }

// Panic creates a message suitable for panicing.
func (l *Logger) Panic() *Message { return newMessage(l, PanicLevel) }

// Mandatory creates a message that will always logged, except when logging
// is disabled.
func (l *Logger) Mandatory() *Message { return newMessage(l, MandatoryLevel) }

// Clone creates a message to clone the logger.
func (l *Logger) Clone() *Message {
	msg := newMessage(l, NeverLevel)
	if msg != nil {
		msg.level = noLevel
	}
	return msg
}

func (l *Logger) writeMessage(level Level, msg string, details []byte) error {
	return l.lw.WriteMessage(level, time.Now(), l.prefix, msg, details)
}

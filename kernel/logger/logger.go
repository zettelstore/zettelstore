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
	"io"
	"strconv"
	"sync"
)

// Level defines the possible log levels
type Level uint8

// Constants for Level
const (
	NoLevel        Level = iota // the absent log level
	DisabledLevel               // Logging is disabled
	TraceLevel                  // Log most internal activities
	DebugLevel                  // Log most data updates
	InfoLevel                   // Log normal activities
	WarnLevel                   // Log event that can be easily recovered
	ErrorLevel                  // Log (persistent) errors
	FatalLevel                  // Log event that cannot be recovered within an internal acitivty
	PanicLevel                  // Log event that must stop the software
	MandatoryLevel              // Log only mandatory events
	maxLevel
)

var logLevel = [...]string{
	"     ",
	"DISAB",
	"TRACE",
	"DEBUG",
	"INFO ",
	"WARN ",
	"ERROR",
	"FSTAL",
	"PANIC",
	">>>>>",
}

var strLevel = [...]string{
	"",
	"disabled",
	"trace",
	"debug",
	"info",
	"warn",
	"error",
	"fatal",
	"panic",
	"mandatory",
}

// IsValid returns true, if the level is a valid level
func (l Level) IsValid() bool { return NoLevel < l && l < maxLevel }

func (l Level) String() string {
	if l.IsValid() {
		return strLevel[l]
	}
	return strconv.Itoa(int(l))
}

// Logger represents an objects that emits logging messages.
type Logger struct {
	w      io.Writer
	mx     sync.RWMutex
	level  Level
	prefix string
}

// New creates a new logger for the given service.
//
// This function must only be called from a kernel implementation, not from
// code that tries to log something.
func New(w io.Writer) *Logger {
	return (&Logger{w: w}).SetLevel(InfoLevel)
}

// SetLevel sets the level of the logger.
func (l *Logger) SetLevel(newLevel Level) *Logger {
	if l != nil {
		l.mx.Lock()
		l.level = newLevel
		l.mx.Unlock()
	}
	return l
}

// Level returns the current level of the given logger
func (l *Logger) Level() Level {
	if l != nil {
		l.mx.RLock()
		result := l.level
		l.mx.RUnlock()
		return result
	}
	return DisabledLevel
}

// Prefix sets the prefix, but only once.
func (l *Logger) Prefix(newPrefix string) *Logger {
	if l != nil && l.prefix == "" {
		l.prefix = (newPrefix + "      ")[:6]
	}
	return l
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

// Fatal creates a message suitable for fatal errors.
func (l *Logger) Fatal() *Message { return newMessage(l, FatalLevel) }

// Panic creates a message suitable for panicing.
func (l *Logger) Panic() *Message { return newMessage(l, PanicLevel) }

// Mandatory creates a message that will always logged, except when logging
// is disabled.
func (l *Logger) Mandatory() *Message { return newMessage(l, MandatoryLevel) }

func (l *Logger) Write(p []byte) (int, error) {
	l.mx.Lock()
	siz, err := l.w.Write(p)
	l.mx.Unlock()
	return siz, err
}

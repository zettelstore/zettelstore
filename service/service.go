//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package service provides the main internal service.
package service

import "net/http"

// Service is the main internal service.
type Service interface {
	// WaitForShutdown blocks the call until Shutdown is called.
	WaitForShutdown()

	// SetDebug to enable/disable debug mode
	SetDebug(enable bool) bool

	// Shutdown the service. Waits for all concurrent activities to stop.
	Shutdown()

	// ShutdownNotifier returns a channel where the caller gets notified to stop.
	ShutdownNotifier() ShutdownChan

	// IgnoreShutdown marks the given channel as to be ignored on shutdown.
	IgnoreShutdown(ob ShutdownChan)

	// Log some activity.
	Log(args ...interface{})

	// LogRecover outputs some information about the previous panic.
	LogRecover(name string, recoverInfo interface{})

	// --- Web server related methods ----------------------------------------

	// WebSetConfig store the configuration data for the next start of the web server.
	WebSetConfig(addrListen string, handler http.Handler)

	WebStart() error

	WebStop() error
}

// Main references the main service.
var Main Service

// Unit is a type with just one value.
type Unit struct{}

// ShutdownChan is a channel used to signal a system shutdown.
type ShutdownChan <-chan Unit

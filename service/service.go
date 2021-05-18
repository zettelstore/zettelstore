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

import (
	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
	"zettelstore.de/z/web/server"
)

// Service is the main internal service.
type Service interface {
	// Start the service.
	Start(headline bool)

	// WaitForShutdown blocks the call until Shutdown is called.
	WaitForShutdown()

	// SetDebug to enable/disable debug mode
	SetDebug(enable bool) bool

	// Shutdown the service. Waits for all concurrent activities to stop.
	Shutdown(silent bool)

	// ShutdownNotifier returns a channel where the caller gets notified to stop.
	ShutdownNotifier() ShutdownChan

	// IgnoreShutdown marks the given channel as to be ignored on shutdown.
	IgnoreShutdown(ob ShutdownChan)

	// Log some activity.
	Log(args ...interface{})

	// LogRecover outputs some information about the previous panic.
	LogRecover(name string, recoverInfo interface{}) bool

	// SetConfig stores a configuration value.
	SetConfig(subsrv Subservice, key, value string) bool

	// GetConfig returns a configuration value.
	GetConfig(subsrv Subservice, key string) interface{}

	// GetConfigList returns a sorted list of configuration data.
	GetConfigList(subsrv Subservice) []KeyDescrValue

	// StartSub start the given sub-service.
	StartSub(subsrv Subservice) error

	// StopSub stop the given sub-service.
	StopSub(subsrv Subservice) error

	// GetSubStatistics returns a key/value list with statistical data.
	GetSubStatistics(subsrv Subservice) []KeyValue

	// SetCreators store the configuration data for the next start of the web server.
	SetCreators(CreateAuthManagerFunc, CreatePlaceManagerFunc, SetupWebServerFunc)
}

// Main references the main service.
var Main Service

// Unit is a type with just one value.
type Unit struct{}

// ShutdownChan is a channel used to signal a system shutdown.
type ShutdownChan <-chan Unit

// Subservice specifies an sub-service of the main service, e.g. web, ...
type Subservice uint8

// Constants for type Subservice.
const (
	_ Subservice = iota
	SubCore
	SubAuth
	SubPlace
	SubWeb
)

// Constants for core subservice system keys.
const (
	CoreGoArch    = "go-arch"
	CoreGoOS      = "go-os"
	CoreGoVersion = "go-version"
	CoreHostname  = "hostname"
	CorePort      = "port"
	CoreProgname  = "progname"
	CoreVerbose   = "verbose"
	CoreVersion   = "version"
)

// Constants for authentication subservice keys.
const (
	AuthOwner    = "owner"
	AuthReadonly = "readonly"
)

// Constants for place subservice keys.
const (
	PlaceDefaultDirType = "defdirtype"
)

// Allowed values for PlaceDefaultDirType
const (
	PlaceDirTypeNotify = "notify"
	PlaceDirTypeSimple = "simple"
)

// Constants for web subservice keys.
const (
	WebListenAddress = "listen"
	WebURLPrefix     = "prefix"
)

// KeyDescrValue is a triple of config data.
type KeyDescrValue struct{ Key, Descr, Value string }

// KeyValue is a pair of key and value.
type KeyValue struct{ Key, Value string }

// CreateAuthManagerFunc is called to create a new auth manager.
type CreateAuthManagerFunc func(readonly bool, owner id.Zid) (auth.Manager, error)

// CreatePlaceManagerFunc is called to create a new place manager.
type CreatePlaceManagerFunc func(authManager auth.Manager) (place.Manager, error)

// SetupWebServerFunc is called to create a new web service handler.
type SetupWebServerFunc func(
	webServer server.Server,
	placeManager place.Manager,
	authManager auth.Manager) error

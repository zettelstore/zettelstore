//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package kernel provides the main kernel service.
package kernel

import (
	"zettelstore.de/z/auth"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
	"zettelstore.de/z/web/server"
)

// Kernel is the main internal service.
type Kernel interface {
	// Start the service.
	Start(headline bool)

	// WaitForShutdown blocks the call until Shutdown is called.
	WaitForShutdown()

	// SetDebug to enable/disable debug mode
	SetDebug(enable bool) bool

	// Shutdown the service. Waits for all concurrent activities to stop.
	Shutdown(silent bool)

	// Log some activity.
	Log(args ...interface{})

	// LogRecover outputs some information about the previous panic.
	LogRecover(name string, recoverInfo interface{}) bool

	// SetConfig stores a configuration value.
	SetConfig(srv Service, key, value string) bool

	// GetConfig returns a configuration value.
	GetConfig(srv Service, key string) interface{}

	// GetConfigList returns a sorted list of configuration data.
	GetConfigList(Service) []KeyDescrValue

	// StartService start the given service.
	StartService(Service) error

	// RestartService stops and restarts the given service, while maintaining service dependencies.
	RestartService(Service) error

	// StopService stop the given service.
	StopService(Service) error

	// GetServiceStatistics returns a key/value list with statistical data.
	GetServiceStatistics(Service) []KeyValue

	// SetCreators store functions to be called when a service has to be created.
	SetCreators(CreateAuthManagerFunc, CreatePlaceManagerFunc, SetupWebServerFunc)
}

// Main references the main kernel.
var Main Kernel

// Unit is a type with just one value.
type Unit struct{}

// ShutdownChan is a channel used to signal a system shutdown.
type ShutdownChan <-chan Unit

// Service specifies a service, e.g. web, ...
type Service uint8

// Constants for type Service.
const (
	_ Service = iota
	CoreService
	ConfigService
	AuthService
	PlaceService
	WebService
)

// Constants for core service system keys.
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

// Constants for authentication service keys.
const (
	AuthOwner    = "owner"
	AuthReadonly = "readonly"
)

// Constants for place service keys.
const (
	PlaceDefaultDirType = "defdirtype"
)

// Allowed values for PlaceDefaultDirType
const (
	PlaceDirTypeNotify = "notify"
	PlaceDirTypeSimple = "simple"
)

// Constants for web service keys.
const (
	WebListenAddress     = "listen"
	WebPersistentCookie  = "persistent"
	WebSecureCookie      = "secure"
	WebTokenLifetimeAPI  = "api-lifetime"
	WebTokenLifetimeHTML = "html-lifetime"
	WebURLPrefix         = "prefix"
)

// KeyDescrValue is a triple of config data.
type KeyDescrValue struct{ Key, Descr, Value string }

// KeyValue is a pair of key and value.
type KeyValue struct{ Key, Value string }

// CreateAuthManagerFunc is called to create a new auth manager.
type CreateAuthManagerFunc func(readonly bool, owner id.Zid) (auth.Manager, error)

// CreatePlaceManagerFunc is called to create a new place manager.
type CreatePlaceManagerFunc func(authManager auth.Manager, rtConfig config.Config) (place.Manager, error)

// SetupWebServerFunc is called to create a new web service handler.
type SetupWebServerFunc func(
	webServer server.Server,
	placeManager place.Manager,
	authManager auth.Manager,
	rtConfig config.Config,
) error

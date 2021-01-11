//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the zettelstore.
package usecase

import (
	"context"
)

// ReloadPort is the interface used by this use case.
type ReloadPort interface {
	// Reload clears all caches, reloads all internal data to reflect changes
	// that were possibly undetected.
	Reload(ctx context.Context) error
}

// Reload is the data for this use case.
type Reload struct {
	port ReloadPort
}

// NewReload creates a new use case.
func NewReload(port ReloadPort) Reload {
	return Reload{port: port}
}

// Run executes the use case.
func (uc Reload) Run(ctx context.Context) error {
	return uc.port.Reload(ctx)
}

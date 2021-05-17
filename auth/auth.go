//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package auth provides services for authentification / authorization.
package auth

// Manager is the main interface for providing the service.
type Manager interface {
	// IsReadonly returns true, if the systems is configured to run in read-only-mode.
	IsReadonly() bool
}

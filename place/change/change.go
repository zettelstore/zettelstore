//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package change provides definition for place changes.
package change

import "zettelstore.de/z/domain/id"

// Reason gives an indication, why the ObserverFunc was called.
type Reason uint8

// Values for Reason
const (
	_        Reason = iota
	OnReload        // Place was reloaded
	OnUpdate        // A zettel was created or changed
	OnDelete        // A zettel was removed
)

// Info contains all the data about a changed zettel.
type Info struct {
	Reason Reason
	Zid    id.Zid
}

// Func is a function to be called when a change is detected.
type Func func(Info)

// Subject is a place that notifies observers about changes.
type Subject interface {
	// RegisterObserver registers an observer that will be notified
	// if one or all zettel are found to be changed.
	RegisterObserver(Func)
}

//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package place provides a generic interface to zettel places.
package place

import (
	"context"
	"errors"
	"fmt"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// Place is implemented by all Zettel places.
type Place interface {
	// Location returns some information where the place is located.
	// Format is dependent of the place.
	Location() string

	// CanCreateZettel returns true, if place could possibly create a new zettel.
	CanCreateZettel(ctx context.Context) bool

	// CreateZettel creates a new zettel.
	// Returns the new zettel id (and an error indication).
	CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error)

	// GetZettel retrieves a specific zettel.
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)

	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)

	// FetchZids returns the set of all zettel identifer managed by the place.
	FetchZids(ctx context.Context) (id.Set, error)

	// SelectMeta returns all zettel meta data that match the selection criteria.
	// TODO: more docs
	SelectMeta(ctx context.Context, f *Filter, s *Sorter) ([]*meta.Meta, error)

	// CanUpdateZettel returns true, if place could possibly update the given zettel.
	CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool

	// UpdateZettel updates an existing zettel.
	UpdateZettel(ctx context.Context, zettel domain.Zettel) error

	// AllowRenameZettel returns true, if place will not disallow renaming the zettel.
	AllowRenameZettel(ctx context.Context, zid id.Zid) bool

	// RenameZettel changes the current Zid to a new Zid.
	RenameZettel(ctx context.Context, curZid, newZid id.Zid) error

	// CanDeleteZettel returns true, if place could possibly delete the given zettel.
	CanDeleteZettel(ctx context.Context, zid id.Zid) bool

	// DeleteZettel removes the zettel from the place.
	DeleteZettel(ctx context.Context, zid id.Zid) error

	// Reload clears all caches, reloads all internal data to reflect changes
	// that were possibly undetected.
	Reload(ctx context.Context) error

	// ReadStats populates st with place statistics
	ReadStats(st *Stats)
}

// Stats records statistics about the place.
type Stats struct {
	// ReadOnly indicates that the places cannot be changed
	ReadOnly bool

	// Zettel is the number of zettel managed by the place.
	Zettel int
}

// StartStopper performs simple lifecycle management.
type StartStopper interface {
	// Start the place. Now all other functions of the place are allowed.
	// Starting an already started place is not allowed.
	Start(ctx context.Context) error

	// Stop the started place. Now only the Start() function is allowed.
	Stop(ctx context.Context) error
}

// ChangeReason gives an indication, why the ObserverFunc was called.
type ChangeReason int

// Values for ChangeReason
const (
	_        ChangeReason = iota
	OnReload              // Place was reloaded
	OnUpdate              // A zettel was created or changed
	OnDelete              // A zettel was removed
)

// ChangeInfo contains all the data about a changed zettel.
type ChangeInfo struct {
	Reason ChangeReason
	Zid    id.Zid
}

// Manager is a place-managing place.
type Manager interface {
	Place
	StartStopper

	// RegisterObserver registers an observer that will be notified
	// if one or all zettel are found to be changed.
	RegisterObserver(func(ChangeInfo))

	// NumPlaces returns the number of managed places.
	NumPlaces() int
}

// ErrNotAllowed is returned if the caller is not allowed to perform the operation.
type ErrNotAllowed struct {
	Op   string
	User *meta.Meta
	Zid  id.Zid
}

// NewErrNotAllowed creates an new authorization error.
func NewErrNotAllowed(op string, user *meta.Meta, zid id.Zid) error {
	return &ErrNotAllowed{
		Op:   op,
		User: user,
		Zid:  zid,
	}
}

func (err *ErrNotAllowed) Error() string {
	if err.User == nil {
		if err.Zid.IsValid() {
			return fmt.Sprintf(
				"Operation %q on zettel %v not allowed for not authorized user",
				err.Op,
				err.Zid.String())
		}
		return fmt.Sprintf("Operation %q not allowed for not authorized user", err.Op)
	}
	if err.Zid.IsValid() {
		return fmt.Sprintf(
			"Operation %q on zettel %v not allowed for user %v/%v",
			err.Op,
			err.Zid.String(),
			err.User.GetDefault(meta.KeyUserID, "?"),
			err.User.Zid.String())
	}
	return fmt.Sprintf(
		"Operation %q not allowed for user %v/%v",
		err.Op,
		err.User.GetDefault(meta.KeyUserID, "?"),
		err.User.Zid.String())
}

// IsErrNotAllowed return true, if the error is of type ErrNotAllowed.
func IsErrNotAllowed(err error) bool {
	_, ok := err.(*ErrNotAllowed)
	return ok
}

// ErrStarted is returned when trying to start an already started place.
var ErrStarted = errors.New("Place is already started")

// ErrStopped is returned if calling methods on a place that was not started.
var ErrStopped = errors.New("Place is stopped")

// ErrReadOnly is returned if there is an attepmt to write to a read-only place.
var ErrReadOnly = errors.New("Read-only place")

// ErrNotFound is returned if a zettel was not found in the place.
var ErrNotFound = errors.New("Zettel not found")

// ErrInvalidID is returned if the zettel id is not appropriate for the place operation.
type ErrInvalidID struct{ Zid id.Zid }

func (err *ErrInvalidID) Error() string { return "Invalid Zettel id: " + err.Zid.String() }

// Filter specifies a mechanism for selecting zettel.
type Filter struct {
	Expr   FilterExpr
	Negate bool
	Select func(*meta.Meta) bool
}

// FilterExpr is the encoding of a search filter.
type FilterExpr map[string][]string // map of keys to or-ed values

// Sorter specifies ordering and limiting a sequnce of meta data.
type Sorter struct {
	Order      string // Name of meta key. None given: use "id"
	Descending bool   // Sort by order, but descending
	Offset     int    // <= 0: no offset
	Limit      int    // <= 0: no limit
}

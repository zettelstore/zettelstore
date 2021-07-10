//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta provides the domain specific type 'meta'.
package meta

// Visibility enumerates the variations of the 'visibility' meta key.
type Visibility int

// Supported values for visibility.
const (
	_ Visibility = iota
	VisibilityUnknown
	VisibilityPublic
	VisibilityCreator
	VisibilityLogin
	VisibilityOwner
	VisibilityExpert
)

var visMap = map[string]Visibility{
	ValueVisibilityPublic:  VisibilityPublic,
	ValueVisibilityCreator: VisibilityCreator,
	ValueVisibilityLogin:   VisibilityLogin,
	ValueVisibilityOwner:   VisibilityOwner,
	ValueVisibilityExpert:  VisibilityExpert,
}

// GetVisibility returns the visibility value of the given string
func GetVisibility(val string) Visibility {
	if vis, ok := visMap[val]; ok {
		return vis
	}
	return VisibilityUnknown
}

// UserRole enumerates the supported values of meta key 'user-role'.
type UserRole int

// Supported values for user roles.
const (
	_ UserRole = iota
	UserRoleUnknown
	UserRoleCreator
	UserRoleReader
	UserRoleWriter
	UserRoleOwner
)

var urMap = map[string]UserRole{
	ValueUserRoleCreator: UserRoleCreator,
	ValueUserRoleReader:  UserRoleReader,
	ValueUserRoleWriter:  UserRoleWriter,
	ValueUserRoleOwner:   UserRoleOwner,
}

// GetUserRole role returns the user role of the given string.
func GetUserRole(val string) UserRole {
	if ur, ok := urMap[val]; ok {
		return ur
	}
	return UserRoleUnknown
}

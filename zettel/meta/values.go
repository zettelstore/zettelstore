//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package meta

import (
	"fmt"

	"zettelstore.de/c/api"
)

// Supported syntax values.
const (
	SyntaxCSS      = api.ValueSyntaxCSS
	SyntaxDraw     = api.ValueSyntaxDraw
	SyntaxGif      = api.ValueSyntaxGif
	SyntaxHTML     = api.ValueSyntaxHTML
	SyntaxJPEG     = "jpeg"
	SyntaxJPG      = "jpg"
	SyntaxMarkdown = api.ValueSyntaxMarkdown
	SyntaxMD       = api.ValueSyntaxMD
	SyntaxMustache = api.ValueSyntaxMustache
	SyntaxNone     = api.ValueSyntaxNone
	SyntaxPlain    = "plain"
	SyntaxPNG      = "png"
	SyntaxSVG      = api.ValueSyntaxSVG
	SyntaxSxn      = api.ValueSyntaxSxn
	SyntaxText     = api.ValueSyntaxText
	SyntaxTxt      = "txt"
	SyntaxWebp     = "webp"
	SyntaxZmk      = api.ValueSyntaxZmk
)

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
	api.ValueVisibilityPublic:  VisibilityPublic,
	api.ValueVisibilityCreator: VisibilityCreator,
	api.ValueVisibilityLogin:   VisibilityLogin,
	api.ValueVisibilityOwner:   VisibilityOwner,
	api.ValueVisibilityExpert:  VisibilityExpert,
}
var revVisMap = map[Visibility]string{}

func init() {
	for k, v := range visMap {
		revVisMap[v] = k
	}
}

// GetVisibility returns the visibility value of the given string
func GetVisibility(val string) Visibility {
	if vis, ok := visMap[val]; ok {
		return vis
	}
	return VisibilityUnknown
}

func (v Visibility) String() string {
	if s, ok := revVisMap[v]; ok {
		return s
	}
	return fmt.Sprintf("Unknown (%d)", v)
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
	api.ValueUserRoleCreator: UserRoleCreator,
	api.ValueUserRoleReader:  UserRoleReader,
	api.ValueUserRoleWriter:  UserRoleWriter,
	api.ValueUserRoleOwner:   UserRoleOwner,
}

// GetUserRole role returns the user role of the given string.
func GetUserRole(val string) UserRole {
	if ur, ok := urMap[val]; ok {
		return ur
	}
	return UserRoleUnknown
}

//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package ast

import (
	"net/url"
	"strings"

	"zettelstore.de/z/domain/id"
)

// SearchPrefix is the prefix that denotes a search expression.
const SearchPrefix = "search:"

// ParseReference parses a string and returns a reference.
func ParseReference(s string) *Reference {
	if s == "" || s == "00000000000000" {
		return &Reference{URL: nil, Value: s, State: RefStateInvalid}
	}
	if strings.HasPrefix(s, SearchPrefix) {
		return &Reference{URL: nil, Value: s[len(SearchPrefix):], State: RefStateSearch}
	}
	if state, ok := localState(s); ok {
		if state == RefStateBased {
			s = s[1:]
		}
		u, err := url.Parse(s)
		if err == nil {
			return &Reference{URL: u, Value: s, State: state}
		}
	}
	u, err := url.Parse(s)
	if err != nil {
		return &Reference{URL: nil, Value: s, State: RefStateInvalid}
	}
	if len(u.Scheme)+len(u.Opaque)+len(u.Host) == 0 && u.User == nil {
		if _, err = id.Parse(u.Path); err == nil {
			return &Reference{URL: u, Value: s, State: RefStateZettel}
		}
		if u.Path == "" && u.Fragment != "" {
			return &Reference{URL: u, Value: s, State: RefStateSelf}
		}
	}
	return &Reference{URL: u, Value: s, State: RefStateExternal}
}

func localState(path string) (RefState, bool) {
	if len(path) > 0 && path[0] == '/' {
		if len(path) > 1 && path[1] == '/' {
			return RefStateBased, true
		}
		return RefStateHosted, true
	}
	if len(path) > 1 && path[0] == '.' {
		if len(path) > 2 && path[1] == '.' && path[2] == '/' {
			return RefStateHosted, true
		}
		return RefStateHosted, path[1] == '/'
	}
	return RefStateInvalid, false
}

// String returns the string representation of a reference.
func (r Reference) String() string {
	if r.URL != nil {
		return r.URL.String()
	}
	if r.State == RefStateSearch {
		return SearchPrefix + r.Value
	}
	return r.Value
}

// IsValid returns true if reference is valid
func (r *Reference) IsValid() bool { return r.State != RefStateInvalid }

// IsZettel returns true if it is a referencen to a local zettel.
func (r *Reference) IsZettel() bool {
	switch r.State {
	case RefStateZettel, RefStateSelf, RefStateFound, RefStateBroken:
		return true
	}
	return false
}

// IsLocal returns true if reference is local
func (r *Reference) IsLocal() bool {
	return r.State == RefStateHosted || r.State == RefStateBased
}

// IsExternal returns true if it is a referencen to external material.
func (r *Reference) IsExternal() bool { return r.State == RefStateExternal }

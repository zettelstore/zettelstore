//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package adapter provides handlers for web requests.
package adapter

import (
	"net/url"
	"strings"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
)

type urlQuery struct{ key, val string }

// URLBuilder should be used to create zettelstore URLs.
type URLBuilder struct {
	key      byte
	path     []string
	query    []urlQuery
	fragment string
}

// NewURLBuilder creates a new URLBuilder.
func NewURLBuilder(key byte) *URLBuilder {
	return &URLBuilder{key: key}
}

// Clone an URLBuilder
func (ub *URLBuilder) Clone() *URLBuilder {
	cpy := new(URLBuilder)
	cpy.key = ub.key
	if len(ub.path) > 0 {
		cpy.path = make([]string, 0, len(ub.path))
		cpy.path = append(cpy.path, ub.path...)
	}
	if len(ub.query) > 0 {
		cpy.query = make([]urlQuery, 0, len(ub.query))
		cpy.query = append(cpy.query, ub.query...)
	}
	cpy.fragment = ub.fragment
	return cpy
}

// SetZid sets the zettel identifier.
func (ub *URLBuilder) SetZid(zid id.Zid) *URLBuilder {
	if len(ub.path) > 0 {
		panic("Cannot add Zid")
	}
	ub.path = append(ub.path, zid.String())
	return ub
}

// AppendPath adds a new path element
func (ub *URLBuilder) AppendPath(p string) *URLBuilder {
	ub.path = append(ub.path, p)
	return ub
}

// AppendQuery adds a new query parameter
func (ub *URLBuilder) AppendQuery(key string, value string) *URLBuilder {
	ub.query = append(ub.query, urlQuery{key, value})
	return ub
}

// ClearQuery removes all query parameters.
func (ub *URLBuilder) ClearQuery() *URLBuilder {
	ub.query = nil
	ub.fragment = ""
	return ub
}

// SetFragment stores the fragment
func (ub *URLBuilder) SetFragment(s string) *URLBuilder {
	ub.fragment = s
	return ub
}

// String produces a string value.
func (ub *URLBuilder) String() string {
	var sb strings.Builder

	sb.WriteString(startup.URLPrefix())
	if ub.key != '/' {
		sb.WriteByte(ub.key)
	}
	for _, p := range ub.path {
		sb.WriteByte('/')
		sb.WriteString(url.PathEscape(p))
	}
	if len(ub.fragment) > 0 {
		sb.WriteByte('#')
		sb.WriteString(ub.fragment)
	}
	for i, q := range ub.query {
		if i == 0 {
			sb.WriteByte('?')
		} else {
			sb.WriteByte('&')
		}
		sb.WriteString(q.key)
		sb.WriteByte('=')
		sb.WriteString(url.QueryEscape(q.val))
	}
	return sb.String()
}

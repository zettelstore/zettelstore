//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta_test provides tests for the domain specific type 'meta'.
package meta_test

import (
	"strings"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

const testID = id.Zid(98765432101234)

func newMeta(title string, tags []string, syntax string) *meta.Meta {
	m := meta.New(testID)
	if title != "" {
		m.Set(api.KeyTitle, title)
	}
	if tags != nil {
		m.Set(api.KeyTags, strings.Join(tags, " "))
	}
	if syntax != "" {
		m.Set(api.KeySyntax, syntax)
	}
	return m
}
func assertWriteMeta(t *testing.T, m *meta.Meta, expected string) {
	t.Helper()
	sb := strings.Builder{}
	m.Write(&sb, true)
	if got := sb.String(); got != expected {
		t.Errorf("\nExp: %q\ngot: %q", expected, got)
	}
}

func TestWriteMeta(t *testing.T) {
	t.Parallel()
	assertWriteMeta(t, newMeta("", nil, ""), "")

	m := newMeta("TITLE", []string{"#t1", "#t2"}, "syntax")
	assertWriteMeta(t, m, "title: TITLE\ntags: #t1 #t2\nsyntax: syntax\n")

	m = newMeta("TITLE", nil, "")
	m.Set("user", "zettel")
	m.Set("auth", "basic")
	assertWriteMeta(t, m, "title: TITLE\nauth: basic\nuser: zettel\n")
}

//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package index allows to search for metadata and content.
package index_test

import (
	"sort"
	"testing"

	"zettelstore.de/z/index"
)

func equalWordList(exp, got []string) bool {
	if len(exp) != len(got) {
		return false
	}
	if len(got) == 0 {
		return len(exp) == 0
	}
	sort.Strings(got)
	for i, w := range exp {
		if w != got[i] {
			return false
		}
	}
	return true
}

func TestWordsWords(t *testing.T) {
	testcases := []struct {
		words index.WordSet
		exp   []string
	}{
		{nil, nil},
		{index.WordSet{}, nil},
		{index.WordSet{"a": 1, "b": 2}, []string{"a", "b"}},
	}
	for i, tc := range testcases {
		got := tc.words.Words()
		if !equalWordList(tc.exp, got) {
			t.Errorf("%d: %v.Words() == %v, but got %v", i, tc.words, tc.exp, got)
		}
	}
}

func TestWordsDiff(t *testing.T) {
	testcases := []struct {
		cur        index.WordSet
		old        []string
		expN, expR []string
	}{
		{nil, nil, nil, nil},
		{index.WordSet{}, []string{}, nil, nil},
		{index.WordSet{"a": 1}, []string{}, []string{"a"}, nil},
		{index.WordSet{"a": 1}, []string{"b"}, []string{"a"}, []string{"b"}},
		{index.WordSet{}, []string{"b"}, nil, []string{"b"}},
		{index.WordSet{"a": 1}, []string{"a"}, nil, nil},
	}
	for i, tc := range testcases {
		gotN, gotR := tc.cur.Diff(tc.old)
		if !equalWordList(tc.expN, gotN) {
			t.Errorf("%d: %v.Diff(%v)->new %v, but got %v", i, tc.cur, tc.old, tc.expN, gotN)
		}
		if !equalWordList(tc.expR, gotR) {
			t.Errorf("%d: %v.Diff(%v)->rem %v, but got %v", i, tc.cur, tc.old, tc.expR, gotR)
		}
	}
}

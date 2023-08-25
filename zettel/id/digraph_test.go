//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package id_test

import (
	"testing"

	"zettelstore.de/z/zettel/id"
)

type zps = id.EdgeSlice

func createDigraph(pairs zps) (dg id.Digraph) {
	return dg.AddEgdes(pairs)
}

func TestDigraphOriginators(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name string
		dg   id.EdgeSlice
		orig id.Set
		term id.Set
	}{
		{"empty", nil, nil, nil},
		{"single", zps{{0, 1}}, id.NewSet(0), id.NewSet(1)},
		{"chain", zps{{0, 1}, {1, 2}, {2, 3}}, id.NewSet(0), id.NewSet(3)},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			dg := createDigraph(tc.dg)
			if got := dg.Originators(); !tc.orig.Equal(got) {
				t.Errorf("Originators: expected:\n%v, but got:\n%v", tc.orig, got)
			}
			if got := dg.Terminators(); !tc.term.Equal(got) {
				t.Errorf("Termintors: expected:\n%v, but got:\n%v", tc.orig, got)
			}
		})
	}
}

func TestDigraphReachableVertices(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name  string
		pairs id.EdgeSlice
		start id.Zid
		exp   id.Set
	}{
		{"nil", nil, 0, nil},
		{"0-2", zps{{1, 2}, {2, 3}}, 1, id.NewSet(2, 3)},
		{"1,2", zps{{1, 2}, {2, 3}}, 2, id.NewSet(3)},
		{"0-2,1-2", zps{{1, 2}, {2, 3}, {1, 3}}, 1, id.NewSet(2, 3)},
		{"0-2,1-2/1", zps{{1, 2}, {2, 3}, {1, 3}}, 2, id.NewSet(3)},
		{"0-2,1-2/2", zps{{1, 2}, {2, 3}, {1, 3}}, 3, nil},
		{"0-2,1-2,3*", zps{{1, 2}, {2, 3}, {1, 3}, {4, 4}}, 1, id.NewSet(2, 3)},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			dg := createDigraph(tc.pairs)
			if got := dg.ReachableVertices(tc.start); !got.Equal(tc.exp) {
				t.Errorf("\n%v, but got:\n%v", tc.exp, got)
			}

		})
	}
}

func TestDigraphTransitiveClosure(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name  string
		pairs id.EdgeSlice
		start id.Zid
		exp   id.EdgeSlice
	}{
		{"nil", nil, 0, nil},
		{"1-3", zps{{1, 2}, {2, 3}}, 1, zps{{1, 2}, {2, 3}}},
		{"1,2", zps{{1, 1}, {2, 3}}, 2, zps{{2, 3}}},
		{"0-2,1-2", zps{{1, 2}, {2, 3}, {1, 3}}, 1, zps{{1, 2}, {1, 3}, {2, 3}}},
		{"0-2,1-2/1", zps{{1, 2}, {2, 3}, {1, 3}}, 1, zps{{1, 2}, {1, 3}, {2, 3}}},
		{"0-2,1-2/2", zps{{1, 2}, {2, 3}, {1, 3}}, 2, zps{{2, 3}}},
		{"0-2,1-2,3*", zps{{1, 2}, {2, 3}, {1, 3}, {4, 4}}, 1, zps{{1, 2}, {1, 3}, {2, 3}}},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			dg := createDigraph(tc.pairs)
			if got := dg.TransitiveClosure(tc.start).Edges().Sort(); !got.Equal(tc.exp) {
				t.Errorf("\n%v, but got:\n%v", tc.exp, got)
			}

		})
	}
}

func TestIsDAG(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name string
		dg   id.EdgeSlice
		exp  bool
	}{
		{"empty", nil, true},
		{"single-edge", zps{{1, 2}}, true},
		{"single-loop", zps{{1, 1}}, false},
		{"long-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 2}}, false},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if zid, got := createDigraph(tc.dg).IsDAG(); got != tc.exp {
				t.Errorf("expected %v, but got %v (%v)", tc.exp, got, zid)
			}
		})
	}
}

func TestDigraphSortReverse(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name string
		dg   id.EdgeSlice
		exp  id.Slice
	}{
		{"empty", nil, nil},
		{"single-edge", zps{{1, 2}}, id.Slice{2, 1}},
		{"single-loop", zps{{1, 1}}, nil},
		{"end-loop", zps{{1, 2}, {2, 2}}, id.Slice{2, 1}},
		{"long-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 2}}, id.Slice{5, 4, 3, 2, 1}},
		{"sect-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {4, 2}}, id.Slice{5, 4, 3, 2, 1}},
		{"two-islands", zps{{1, 2}, {2, 3}, {4, 5}}, id.Slice{5, 4, 3, 2, 1}},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := createDigraph(tc.dg).SortReverse(); !got.Equal(tc.exp) {
				t.Errorf("expected:\n%v, but got:\n%v", tc.exp, got)
			}
		})
	}
}

//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2023-present Detlef Stern
//-----------------------------------------------------------------------------

package id_test

import (
	"testing"

	"zettelstore.de/z/zettel/id"
)

type zps = id.EdgeSliceO

func createDigraphO(pairs zps) (dg id.DigraphO) {
	return dg.AddEgdes(pairs)
}

func TestDigraphOriginatorsO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name string
		dg   id.EdgeSliceO
		orig id.SetO
		term id.SetO
	}{
		{"empty", nil, nil, nil},
		{"single", zps{{0, 1}}, id.NewSetO(0), id.NewSetO(1)},
		{"chain", zps{{0, 1}, {1, 2}, {2, 3}}, id.NewSetO(0), id.NewSetO(3)},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			dg := createDigraphO(tc.dg)
			if got := dg.Originators(); !tc.orig.Equal(got) {
				t.Errorf("Originators: expected:\n%v, but got:\n%v", tc.orig, got)
			}
			if got := dg.Terminators(); !tc.term.Equal(got) {
				t.Errorf("Termintors: expected:\n%v, but got:\n%v", tc.orig, got)
			}
		})
	}
}

func TestDigraphReachableVerticesO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name  string
		pairs id.EdgeSliceO
		start id.ZidO
		exp   id.SetO
	}{
		{"nil", nil, 0, nil},
		{"0-2", zps{{1, 2}, {2, 3}}, 1, id.NewSetO(2, 3)},
		{"1,2", zps{{1, 2}, {2, 3}}, 2, id.NewSetO(3)},
		{"0-2,1-2", zps{{1, 2}, {2, 3}, {1, 3}}, 1, id.NewSetO(2, 3)},
		{"0-2,1-2/1", zps{{1, 2}, {2, 3}, {1, 3}}, 2, id.NewSetO(3)},
		{"0-2,1-2/2", zps{{1, 2}, {2, 3}, {1, 3}}, 3, nil},
		{"0-2,1-2,3*", zps{{1, 2}, {2, 3}, {1, 3}, {4, 4}}, 1, id.NewSetO(2, 3)},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			dg := createDigraphO(tc.pairs)
			if got := dg.ReachableVertices(tc.start); !got.Equal(tc.exp) {
				t.Errorf("\n%v, but got:\n%v", tc.exp, got)
			}

		})
	}
}

func TestDigraphTransitiveClosureO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name  string
		pairs id.EdgeSliceO
		start id.ZidO
		exp   id.EdgeSliceO
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
			dg := createDigraphO(tc.pairs)
			if got := dg.TransitiveClosure(tc.start).Edges().Sort(); !got.Equal(tc.exp) {
				t.Errorf("\n%v, but got:\n%v", tc.exp, got)
			}
		})
	}
}

func TestIsDAGO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name string
		dg   id.EdgeSliceO
		exp  bool
	}{
		{"empty", nil, true},
		{"single-edge", zps{{1, 2}}, true},
		{"single-loop", zps{{1, 1}}, false},
		{"long-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 2}}, false},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if zid, got := createDigraphO(tc.dg).IsDAG(); got != tc.exp {
				t.Errorf("expected %v, but got %v (%v)", tc.exp, got, zid)
			}
		})
	}
}

func TestDigraphReverseO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name string
		dg   id.EdgeSliceO
		exp  id.EdgeSliceO
	}{
		{"empty", nil, nil},
		{"single-edge", zps{{1, 2}}, zps{{2, 1}}},
		{"single-loop", zps{{1, 1}}, zps{{1, 1}}},
		{"end-loop", zps{{1, 2}, {2, 2}}, zps{{2, 1}, {2, 2}}},
		{"long-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 2}}, zps{{2, 1}, {2, 5}, {3, 2}, {4, 3}, {5, 4}}},
		{"sect-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {4, 2}}, zps{{2, 1}, {2, 4}, {3, 2}, {4, 3}, {5, 4}}},
		{"two-islands", zps{{1, 2}, {2, 3}, {4, 5}}, zps{{2, 1}, {3, 2}, {5, 4}}},
		{"direct-indirect", zps{{1, 2}, {1, 3}, {3, 2}}, zps{{2, 1}, {2, 3}, {3, 1}}},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			dg := createDigraphO(tc.dg)
			if got := dg.Reverse().Edges().Sort(); !got.Equal(tc.exp) {
				t.Errorf("\n%v, but got:\n%v", tc.exp, got)
			}
		})
	}
}

func TestDigraphSortReverseO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name string
		dg   id.EdgeSliceO
		exp  id.SliceO
	}{
		{"empty", nil, nil},
		{"single-edge", zps{{1, 2}}, id.SliceO{2, 1}},
		{"single-loop", zps{{1, 1}}, nil},
		{"end-loop", zps{{1, 2}, {2, 2}}, id.SliceO{}},
		{"long-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 2}}, id.SliceO{}},
		{"sect-loop", zps{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {4, 2}}, id.SliceO{5}},
		{"two-islands", zps{{1, 2}, {2, 3}, {4, 5}}, id.SliceO{5, 3, 4, 2, 1}},
		{"direct-indirect", zps{{1, 2}, {1, 3}, {3, 2}}, id.SliceO{2, 3, 1}},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := createDigraphO(tc.dg).SortReverse(); !got.Equal(tc.exp) {
				t.Errorf("expected:\n%v, but got:\n%v", tc.exp, got)
			}
		})
	}
}

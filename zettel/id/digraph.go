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

package id

import (
	"maps"
	"slices"
)

// DigraphO relates zettel identifier in a directional way.
type DigraphO map[ZidO]SetO

// AddVertex adds an edge / vertex to the digraph.
func (dg DigraphO) AddVertex(zid ZidO) DigraphO {
	if dg == nil {
		return DigraphO{zid: nil}
	}
	if _, found := dg[zid]; !found {
		dg[zid] = nil
	}
	return dg
}

// RemoveVertex removes a vertex and all its edges from the digraph.
func (dg DigraphO) RemoveVertex(zid ZidO) {
	if len(dg) > 0 {
		delete(dg, zid)
		for vertex, closure := range dg {
			dg[vertex] = closure.Remove(zid)
		}
	}
}

// AddEdge adds a connection from `zid1` to `zid2`.
// Both vertices must be added before. Otherwise the function may panic.
func (dg DigraphO) AddEdge(fromZid, toZid ZidO) DigraphO {
	if dg == nil {
		return DigraphO{fromZid: SetO(nil).Add(toZid), toZid: nil}
	}
	dg[fromZid] = dg[fromZid].Add(toZid)
	return dg
}

// AddEgdes adds all given `Edge`s to the digraph.
//
// In contrast to `AddEdge` the vertices must not exist before.
func (dg DigraphO) AddEgdes(edges EdgeSliceO) DigraphO {
	if dg == nil {
		if len(edges) == 0 {
			return nil
		}
		dg = make(DigraphO, len(edges))
	}
	for _, edge := range edges {
		dg = dg.AddVertex(edge.From)
		dg = dg.AddVertex(edge.To)
		dg = dg.AddEdge(edge.From, edge.To)
	}
	return dg
}

// Equal returns true if both digraphs have the same vertices and edges.
func (dg DigraphO) Equal(other DigraphO) bool {
	return maps.EqualFunc(dg, other, func(cg, co SetO) bool { return cg.Equal(co) })
}

// Clone a digraph.
func (dg DigraphO) Clone() DigraphO {
	if len(dg) == 0 {
		return nil
	}
	copyDG := make(DigraphO, len(dg))
	for vertex, closure := range dg {
		copyDG[vertex] = closure.Clone()
	}
	return copyDG
}

// HasVertex returns true, if `zid` is a vertex of the digraph.
func (dg DigraphO) HasVertex(zid ZidO) bool {
	if len(dg) == 0 {
		return false
	}
	_, found := dg[zid]
	return found
}

// Vertices returns the set of all vertices.
func (dg DigraphO) Vertices() SetO {
	if len(dg) == 0 {
		return nil
	}
	verts := NewSetCapO(len(dg))
	for vert := range dg {
		verts.Add(vert)
	}
	return verts
}

// Edges returns an unsorted slice of the edges of the digraph.
func (dg DigraphO) Edges() (es EdgeSliceO) {
	for vert, closure := range dg {
		for next := range closure {
			es = append(es, EdgeO{From: vert, To: next})
		}
	}
	return es
}

// Originators will return the set of all vertices that are not referenced
// a the to-part of an edge.
func (dg DigraphO) Originators() SetO {
	if len(dg) == 0 {
		return nil
	}
	origs := dg.Vertices()
	for _, closure := range dg {
		origs.Substract(closure)
	}
	return origs
}

// Terminators returns the set of all vertices that does not reference
// other vertices.
func (dg DigraphO) Terminators() (terms SetO) {
	for vert, closure := range dg {
		if len(closure) == 0 {
			terms = terms.Add(vert)
		}
	}
	return terms
}

// TransitiveClosure calculates the sub-graph that is reachable from `zid`.
func (dg DigraphO) TransitiveClosure(zid ZidO) (tc DigraphO) {
	if len(dg) == 0 {
		return nil
	}
	var marked SetO
	stack := SliceO{zid}
	for pos := len(stack) - 1; pos >= 0; pos = len(stack) - 1 {
		curr := stack[pos]
		stack = stack[:pos]
		if marked.Contains(curr) {
			continue
		}
		tc = tc.AddVertex(curr)
		for next := range dg[curr] {
			tc = tc.AddVertex(next)
			tc = tc.AddEdge(curr, next)
			stack = append(stack, next)
		}
		marked = marked.Add(curr)
	}
	return tc
}

// ReachableVertices calculates the set of all vertices that are reachable
// from the given `zid`.
func (dg DigraphO) ReachableVertices(zid ZidO) (tc SetO) {
	if len(dg) == 0 {
		return nil
	}
	stack := dg[zid].Sorted()
	for last := len(stack) - 1; last >= 0; last = len(stack) - 1 {
		curr := stack[last]
		stack = stack[:last]
		if tc.Contains(curr) {
			continue
		}
		closure, found := dg[curr]
		if !found {
			continue
		}
		tc = tc.Add(curr)
		for next := range closure {
			stack = append(stack, next)
		}
	}
	return tc
}

// IsDAG returns a vertex and false, if the graph has a cycle containing the vertex.
func (dg DigraphO) IsDAG() (ZidO, bool) {
	for vertex := range dg {
		if dg.ReachableVertices(vertex).Contains(vertex) {
			return vertex, false
		}
	}
	return Invalid, true
}

// Reverse returns a graph with reversed edges.
func (dg DigraphO) Reverse() (revDg DigraphO) {
	for vertex, closure := range dg {
		revDg = revDg.AddVertex(vertex)
		for next := range closure {
			revDg = revDg.AddVertex(next)
			revDg = revDg.AddEdge(next, vertex)
		}
	}
	return revDg
}

// SortReverse returns a deterministic, topological, reverse sort of the
// digraph.
//
// Works only if digraph is a DAG. Otherwise the algorithm will not terminate
// or returns an arbitrary value.
func (dg DigraphO) SortReverse() (sl SliceO) {
	if len(dg) == 0 {
		return nil
	}
	tempDg := dg.Clone()
	for len(tempDg) > 0 {
		terms := tempDg.Terminators()
		if len(terms) == 0 {
			break
		}
		termSlice := terms.Sorted()
		slices.Reverse(termSlice)
		sl = append(sl, termSlice...)
		for t := range terms {
			tempDg.RemoveVertex(t)
		}
	}
	return sl
}

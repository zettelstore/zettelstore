//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package query

import (
	"context"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// ContextSpec contains all specification values for calculating a context.
type ContextSpec struct {
	Direction ContextDirection
	MaxCost   int
	MaxCount  int
}

// ContextDirection specifies the direction a context should be calculated.
type ContextDirection uint8

const (
	ContextDirBoth ContextDirection = iota
	ContextDirForward
	ContextDirBackward
)

// ContextPort is the collection of box methods needed by this directive.
type ContextPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	SelectMeta(ctx context.Context, metaSeq []*meta.Meta, q *Query) ([]*meta.Meta, error)
}

func (spec *ContextSpec) Print(pe *PrintEnv) {
	pe.printSpace()
	pe.writeString(api.ContextDirective)
	switch spec.Direction {
	case ContextDirBackward:
		pe.printSpace()
		pe.writeString(api.BackwardDirective)
	case ContextDirForward:
		pe.printSpace()
		pe.writeString(api.ForwardDirective)
	}
	pe.printPosInt(api.CostDirective, spec.MaxCost)
	pe.printPosInt(api.MaxDirective, spec.MaxCount)
}

func (spec *ContextSpec) Execute(ctx context.Context, startSeq []*meta.Meta, port ContextPort) []*meta.Meta {
	maxCost := spec.MaxCost
	if maxCost <= 0 {
		maxCost = 17
	}
	maxCount := spec.MaxCount
	if maxCount <= 0 {
		maxCount = 200
	}
	tasks := newQueue(startSeq, maxCost, maxCount, port)
	isBackward := spec.Direction == ContextDirBoth || spec.Direction == ContextDirBackward
	isForward := spec.Direction == ContextDirBoth || spec.Direction == ContextDirForward
	result := []*meta.Meta{}
	for {
		m, cost := tasks.next()
		if m == nil {
			break
		}
		result = append(result, m)

		for _, p := range m.ComputedPairsRest() {
			tasks.addPair(ctx, p.Key, p.Value, cost, isBackward, isForward)
		}
		if tags, found := m.GetList(api.KeyTags); found {
			for _, tag := range tags {
				tasks.addSameTag(ctx, tag, cost)
			}
		}
	}
	return result
}

type ztlCtxTask struct {
	next *ztlCtxTask
	prev *ztlCtxTask
	meta *meta.Meta
	cost int
}

type contextQueue struct {
	port    ContextPort
	seen    id.Set
	first   *ztlCtxTask
	last    *ztlCtxTask
	maxCost int
	limit   int
	tagCost map[string][]*meta.Meta
}

func newQueue(startSeq []*meta.Meta, maxCost, limit int, port ContextPort) *contextQueue {
	result := &contextQueue{
		port:    port,
		seen:    id.NewSet(),
		first:   nil,
		last:    nil,
		maxCost: maxCost,
		limit:   limit,
		tagCost: make(map[string][]*meta.Meta, 1024),
	}

	var prev *ztlCtxTask
	for _, m := range startSeq {
		task := &ztlCtxTask{next: nil, prev: prev, meta: m, cost: 1}
		if prev == nil {
			result.first = task
		} else {
			prev.next = task
		}
		result.last = task
		prev = task
	}
	return result
}

func (zc *contextQueue) addPair(ctx context.Context, key, value string, curCost int, isBackward, isForward bool) {
	if key == api.KeyBack {
		return
	}
	newCost := curCost + contextCost(key)
	if key == api.KeyBackward {
		if isBackward {
			zc.addIDSet(ctx, newCost, value)
		}
		return
	}
	if key == api.KeyForward {
		if isForward {
			zc.addIDSet(ctx, newCost, value)
		}
		return
	}
	hasInverse := meta.Inverse(key) != ""
	if (!hasInverse || !isBackward) && (hasInverse || !isForward) {
		return
	}
	if t := meta.Type(key); t == meta.TypeID {
		zc.addID(ctx, newCost, value)
	} else if t == meta.TypeIDSet {
		zc.addIDSet(ctx, newCost, value)
	}
}

func contextCost(key string) int {
	switch key {
	case api.KeyFolge, api.KeyPrecursor:
		return 1
	case api.KeySuccessors, api.KeyPredecessor,
		api.KeySubordinates, api.KeySuperior:
		return 2
	}
	return 3
}

func (zc *contextQueue) addID(ctx context.Context, newCost int, value string) {
	if zc.costMaxed(newCost) {
		return
	}
	if zid, errParse := id.Parse(value); errParse == nil {
		if m, errGetMeta := zc.port.GetMeta(ctx, zid); errGetMeta == nil {
			zc.addMeta(m, newCost)
		}
	}
}
func (zc *contextQueue) addMeta(m *meta.Meta, newCost int) {
	task := &ztlCtxTask{next: nil, prev: nil, meta: m, cost: newCost}
	if zc.first == nil {
		zc.first = task
		zc.last = task
		return
	}

	// Search backward for a task t with at most the same cost
	for t := zc.last; t != nil; t = t.prev {
		if t.cost <= task.cost {
			// Found!
			if t.next != nil {
				t.next.prev = task
			}
			task.next = t.next
			t.next = task
			task.prev = t
			if task.next == nil {
				zc.last = task
			}
			return
		}
	}

	// We have not found a task, therefore the new task is the first one
	task.next = zc.first
	zc.first.prev = task
	zc.first = task
}

func (zc *contextQueue) costMaxed(newCost int) bool {
	// If len(zc.seen) <= 1, the initial zettel is processed. In this case allow all
	// other zettel that are directly reachable, without taking the cost into account.
	// Of course, the limit ist still relevant.
	return (len(zc.seen) > 1 && zc.maxCost > 0 && newCost > zc.maxCost) || zc.hasLimit()
}

func (zc *contextQueue) addIDSet(ctx context.Context, newCost int, value string) {
	elems := meta.ListFromValue(value)
	refCost := referenceCost(newCost, len(elems))
	for _, val := range elems {
		zc.addID(ctx, refCost, val)
	}
}

func referenceCost(baseCost int, numReferences int) int {
	if numReferences < 5 {
		return baseCost + 1
	}
	if numReferences < 9 {
		return baseCost * 2
	}
	if numReferences < 17 {
		return baseCost * 3
	}
	if numReferences < 33 {
		return baseCost * 4
	}
	if numReferences < 65 {
		return baseCost * 5
	}
	return baseCost * numReferences / 8
}

func (zc *contextQueue) addSameTag(ctx context.Context, tag string, baseCost int) {
	tagMetas, found := zc.tagCost[tag]
	if !found {
		q := Parse(api.KeyTags + api.SearchOperatorHas + tag + " ORDER REVERSE " + api.KeyID)
		ml, err := zc.port.SelectMeta(ctx, nil, q)
		if err != nil {
			return
		}
		tagMetas = ml
		zc.tagCost[tag] = ml
	}
	cost := tagCost(baseCost, len(tagMetas))
	if zc.costMaxed(cost) {
		return
	}
	for _, m := range tagMetas {
		zc.addMeta(m, cost)
	}
}

func tagCost(baseCost, numTags int) int {
	if numTags < 8 {
		return baseCost + numTags/2
	}
	return (baseCost + 2) * (numTags / 4)
}

func (zc *contextQueue) next() (*meta.Meta, int) {
	if zc.hasLimit() {
		return nil, -1
	}
	for zc.first != nil {
		task := zc.first
		zc.first = task.next
		if zc.first == nil {
			zc.last = nil
		} else {
			zc.first.prev = nil
		}
		m := task.meta
		zid := m.Zid
		_, found := zc.seen[zid]
		if found {
			continue
		}
		zc.seen.Zid(zid)
		return m, task.cost
	}
	return nil, -1
}

func (zc *contextQueue) hasLimit() bool {
	limit := zc.limit
	return limit > 0 && len(zc.seen) >= limit
}

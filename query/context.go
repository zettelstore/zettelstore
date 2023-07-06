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

	"zettelstore.de/c/api"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

type contextSpec struct {
	dir      contextDirection
	maxCost  int
	maxCount int
}

type contextDirection uint8

const (
	_ contextDirection = iota
	dirForward
	dirBackward
	dirBoth
)

func (q *Query) getContext(ctx context.Context, preMatch MetaMatchFunc, getMeta GetMetaFunc, selectMeta SelectMetaFunc) ([]*meta.Meta, error) {
	spec := q.context
	if spec == nil {
		return nil, nil
	}

	startSeq := make([]*meta.Meta, 0, len(q.zids))
	for _, zid := range q.zids {
		m, err := getMeta(ctx, zid)
		if err != nil {
			return nil, err
		}
		if preMatch(m) {
			startSeq = append(startSeq, m)
		}
	}
	if len(startSeq) == 0 {
		return []*meta.Meta{}, nil
	}

	maxCost := spec.maxCost
	if maxCost <= 0 {
		maxCost = 17
	}
	maxCount := spec.maxCount
	if maxCount <= 0 {
		maxCount = 200
	}
	tasks := newQueue(startSeq, maxCost, maxCount, preMatch, getMeta, selectMeta)
	isBackward := spec.dir == dirBoth || spec.dir == dirBackward
	isForward := spec.dir == dirBoth || spec.dir == dirForward
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
	return result, nil
}

type ztlCtxTask struct {
	next *ztlCtxTask
	prev *ztlCtxTask
	meta *meta.Meta
	cost int
}

type contextQueue struct {
	preMatch   MetaMatchFunc
	getMeta    GetMetaFunc
	selectMeta SelectMetaFunc
	seen       id.Set
	first      *ztlCtxTask
	last       *ztlCtxTask
	maxCost    int
	limit      int
	tagCost    map[string][]*meta.Meta
}

func newQueue(startSeq []*meta.Meta, maxCost, limit int, preMatch MetaMatchFunc, getMeta GetMetaFunc, selectMeta SelectMetaFunc) *contextQueue {
	result := &contextQueue{
		preMatch:   preMatch,
		getMeta:    getMeta,
		selectMeta: selectMeta,
		seen:       id.NewSet(),
		first:      nil,
		last:       nil,
		maxCost:    maxCost,
		limit:      limit,
		tagCost:    make(map[string][]*meta.Meta, 1024),
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

	zid, err := id.Parse(value)
	if err != nil {
		return
	}
	m, err := zc.getMeta(ctx, zid)
	if err != nil {
		return
	}
	if zc.preMatch(m) {
		zc.addMeta(m, newCost)
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
		ml, err := zc.selectMeta(ctx, q)
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
		if zc.preMatch(m) { // selectMeta will not check preMatch
			zc.addMeta(m, cost)
		}
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

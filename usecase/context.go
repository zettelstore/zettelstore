//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// ZettelContextPort is the interface used by this use case.
type ZettelContextPort interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// ZettelContextConfig is the interface to allow the usecase to read some config data.
type ZettelContextConfig interface {
	// Get returns a config value that might be user-specific.
	Get(ctx context.Context, m *meta.Meta, key string) string
}

// ZettelContext is the data for this use case.
type ZettelContext struct {
	port   ZettelContextPort
	config ZettelContextConfig
}

// NewZettelContext creates a new use case.
func NewZettelContext(port ZettelContextPort, config ZettelContextConfig) ZettelContext {
	return ZettelContext{port: port, config: config}
}

// ZettelContextDirection determines the way, the context is calculated.
type ZettelContextDirection int

// Constant values for ZettelContextDirection
const (
	_                     ZettelContextDirection = iota
	ZettelContextForward                         // Traverse all forwarding links
	ZettelContextBackward                        // Traverse all backwaring links
	ZettelContextBoth                            // Traverse both directions
)

// Run executes the use case.
func (uc ZettelContext) Run(ctx context.Context, zid id.Zid, dir ZettelContextDirection, maxCost, limit int) (result []*meta.Meta, err error) {
	start, err := uc.port.GetMeta(ctx, zid)
	if err != nil {
		return nil, err
	}
	tasks := newQueue(start, maxCost, limit)
	isBackward := dir == ZettelContextBoth || dir == ZettelContextBackward
	isForward := dir == ZettelContextBoth || dir == ZettelContextForward
	for {
		m, cost := tasks.next()
		if m == nil {
			break
		}
		result = append(result, m)

		for _, p := range m.ComputedPairsRest() {
			tasks.addPair(ctx, uc.port, p.Key, p.Value, cost, isBackward, isForward)
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
	seen    id.Set
	first   *ztlCtxTask
	last    *ztlCtxTask
	maxCost int
	limit   int
}

func newQueue(m *meta.Meta, maxCost, limit int) *contextQueue {
	task := &ztlCtxTask{
		next: nil,
		prev: nil,
		meta: m,
		cost: 0,
	}
	result := &contextQueue{
		seen:    id.NewSet(),
		first:   task,
		last:    task,
		maxCost: maxCost,
		limit:   limit,
	}
	return result
}

func (zc *contextQueue) addPair(
	ctx context.Context, port ZettelContextPort,
	key, value string,
	curCost int, isBackward, isForward bool,
) {
	if key == api.KeyBack {
		return
	}
	newCost := curCost + contextCost(key)
	if key == api.KeyBackward {
		if isBackward {
			zc.addIDSet(ctx, port, newCost, value)
		}
		return
	}
	if key == api.KeyForward {
		if isForward {
			zc.addIDSet(ctx, port, newCost, value)
		}
		return
	}
	hasInverse := meta.Inverse(key) != ""
	if (!hasInverse || !isBackward) && (hasInverse || !isForward) {
		return
	}
	if t := meta.Type(key); t == meta.TypeID {
		zc.addID(ctx, port, newCost, value)
	} else if t == meta.TypeIDSet {
		zc.addIDSet(ctx, port, newCost, value)
	}
}

func contextCost(key string) int {
	switch key {
	case api.KeyFolge, api.KeyPrecursor:
		return 1
	case api.KeySuccessors, api.KeyPredecessor:
		return 2
	}
	return 5
}

func (zc *contextQueue) addID(ctx context.Context, port ZettelContextPort, newCost int, value string) {
	if (zc.maxCost > 0 && newCost > zc.maxCost) || zc.hasLimit() {
		return
	}

	zid, err := id.Parse(value)
	if err != nil {
		return
	}

	m, err := port.GetMeta(ctx, zid)
	if err != nil {
		return
	}

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

func (zc *contextQueue) addIDSet(ctx context.Context, port ZettelContextPort, newCost int, value string) {
	for _, val := range meta.ListFromValue(value) {
		zc.addID(ctx, port, newCost, val)
	}
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
	return limit > 0 && len(zc.seen) > limit
}

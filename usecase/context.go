//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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
	// GetHomeZettel returns the value of the "home-zettel" key.
	GetHomeZettel() id.Zid
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
func (uc ZettelContext) Run(ctx context.Context, zid id.Zid, dir ZettelContextDirection, depth, limit int) (result []*meta.Meta, err error) {
	start, err := uc.port.GetMeta(ctx, zid)
	if err != nil {
		return nil, err
	}
	tasks := newQueue(start, depth, limit, uc.config.GetHomeZettel())
	isBackward := dir == ZettelContextBoth || dir == ZettelContextBackward
	isForward := dir == ZettelContextBoth || dir == ZettelContextForward
	for {
		m, curDepth, found := tasks.next()
		if !found {
			break
		}
		result = append(result, m)

		for _, p := range m.ComputedPairsRest() {
			tasks.addPair(ctx, uc.port, p.Key, p.Value, curDepth+1, isBackward, isForward)
		}
	}
	return result, nil
}

type ztlCtxTask struct {
	next  *ztlCtxTask
	meta  *meta.Meta
	depth int
}

type contextQueue struct {
	home     id.Zid
	seen     id.Set
	first    *ztlCtxTask
	last     *ztlCtxTask
	maxDepth int
	limit    int
}

func newQueue(m *meta.Meta, maxDepth, limit int, home id.Zid) *contextQueue {
	task := &ztlCtxTask{
		next:  nil,
		meta:  m,
		depth: 0,
	}
	result := &contextQueue{
		home:     home,
		seen:     id.NewSet(),
		first:    task,
		last:     task,
		maxDepth: maxDepth,
		limit:    limit,
	}
	return result
}

func (zc *contextQueue) addPair(
	ctx context.Context, port ZettelContextPort,
	key, value string,
	curDepth int, isBackward, isForward bool,
) {
	if key == api.KeyBackward {
		if isBackward {
			zc.addIDSet(ctx, port, curDepth, value)
		}
		return
	}
	if key == api.KeyForward {
		if isForward {
			zc.addIDSet(ctx, port, curDepth, value)
		}
		return
	}
	if key == api.KeyBack {
		return
	}
	hasInverse := meta.Inverse(key) != ""
	if (!hasInverse || !isBackward) && (hasInverse || !isForward) {
		return
	}
	if t := meta.Type(key); t == meta.TypeID {
		zc.addID(ctx, port, curDepth, value)
	} else if t == meta.TypeIDSet {
		zc.addIDSet(ctx, port, curDepth, value)
	}
}

func (zc *contextQueue) addID(ctx context.Context, port ZettelContextPort, depth int, value string) {
	if (zc.maxDepth > 0 && depth > zc.maxDepth) || zc.hasLimit() {
		return
	}

	zid, err := id.Parse(value)
	if err != nil || zid == zc.home {
		return
	}

	m, err := port.GetMeta(ctx, zid)
	if err != nil {
		return
	}

	task := &ztlCtxTask{next: nil, meta: m, depth: depth}
	if zc.first == nil {
		zc.first = task
		zc.last = task
	} else {
		zc.last.next = task
		zc.last = task
	}
}

func (zc *contextQueue) addIDSet(ctx context.Context, port ZettelContextPort, curDepth int, value string) {
	for _, val := range meta.ListFromValue(value) {
		zc.addID(ctx, port, curDepth, val)
	}
}

func (zc *contextQueue) next() (*meta.Meta, int, bool) {
	if zc.hasLimit() {
		return nil, -1, false
	}
	for zc.first != nil {
		task := zc.first
		zc.first = task.next
		if zc.first == nil {
			zc.last = nil
		}
		m := task.meta
		zid := m.Zid
		_, found := zc.seen[zid]
		if found {
			continue
		}
		zc.seen.Zid(zid)
		return m, task.depth, true
	}
	return nil, -1, false
}

func (zc *contextQueue) hasLimit() bool {
	limit := zc.limit
	return limit > 0 && len(zc.seen) > limit
}

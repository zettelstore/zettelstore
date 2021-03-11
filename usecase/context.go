//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the Zettelstore.
package usecase

import (
	"context"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// ZettelContextPort is the interface used by this use case.
type ZettelContextPort interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// ZettelContext is the data for this use case.
type ZettelContext struct {
	port ZettelContextPort
}

// NewZettelContext creates a new use case.
func NewZettelContext(port ZettelContextPort) ZettelContext {
	return ZettelContext{port: port}
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

// ParseZCDirection returns a direction value for a given string.
func ParseZCDirection(s string) ZettelContextDirection {
	switch s {
	case "backward":
		return ZettelContextBackward
	case "forward":
		return ZettelContextForward
	}
	return ZettelContextBoth
}

// Run executes the use case.
func (uc ZettelContext) Run(ctx context.Context, zid id.Zid, dir ZettelContextDirection, depth, limit int) (result []*meta.Meta, err error) {
	start, err := uc.port.GetMeta(ctx, zid)
	if err != nil {
		return nil, err
	}
	tasks := ztlCtx{depth: depth}
	uc.addInitialTasks(ctx, &tasks, start)
	visited := id.NewSet()
	isBackward := dir == ZettelContextBoth || dir == ZettelContextBackward
	isForward := dir == ZettelContextBoth || dir == ZettelContextForward
	for !tasks.empty() {
		m, curDepth := tasks.pop()
		if _, ok := visited[m.Zid]; ok {
			continue
		}
		visited[m.Zid] = true
		result = append(result, m)
		if limit > 0 && len(result) > limit { // start is the first element of result
			break
		}
		curDepth++
		for _, p := range m.PairsRest(true) {
			if p.Key == meta.KeyBackward {
				if isBackward {
					uc.addIDSet(ctx, &tasks, curDepth, p.Value)
				}
				continue
			}
			if p.Key == meta.KeyForward {
				if isForward {
					uc.addIDSet(ctx, &tasks, curDepth, p.Value)
				}
				continue
			}
			if p.Key != meta.KeyBack {
				hasInverse := meta.Inverse(p.Key) != ""
				if (!hasInverse || !isBackward) && (hasInverse || !isForward) {
					continue
				}
				if t := meta.Type(p.Key); t == meta.TypeID {
					uc.addID(ctx, &tasks, curDepth, p.Value)
				} else if t == meta.TypeIDSet {
					uc.addIDSet(ctx, &tasks, curDepth, p.Value)
				}
			}
		}
	}
	return result, nil
}

func (uc ZettelContext) addInitialTasks(ctx context.Context, tasks *ztlCtx, start *meta.Meta) {
	tasks.add(start, 0)
}

func (uc ZettelContext) addID(ctx context.Context, tasks *ztlCtx, depth int, value string) {
	if zid, err := id.Parse(value); err == nil {
		if m, err := uc.port.GetMeta(ctx, zid); err == nil {
			tasks.add(m, depth)
		}
	}
}

func (uc ZettelContext) addIDSet(ctx context.Context, tasks *ztlCtx, depth int, value string) {
	for _, val := range meta.ListFromValue(value) {
		uc.addID(ctx, tasks, depth, val)
	}
}

type ztlCtxTask struct {
	next  *ztlCtxTask
	meta  *meta.Meta
	depth int
}

type ztlCtx struct {
	first *ztlCtxTask
	last  *ztlCtxTask
	depth int
}

func (zc *ztlCtx) add(m *meta.Meta, depth int) {
	if zc.depth > 0 && depth > zc.depth {
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

func (zc *ztlCtx) empty() bool {
	return zc.first == nil
}

func (zc *ztlCtx) pop() (*meta.Meta, int) {
	task := zc.first
	if task == nil {
		return nil, -1
	}
	zc.first = task.next
	if zc.first == nil {
		zc.last = nil
	}
	return task.meta, task.depth
}

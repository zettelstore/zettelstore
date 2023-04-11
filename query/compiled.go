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
	"math/rand"
	"sort"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// Compiled is a compiled query, to be used in a Box
type Compiled struct {
	hasQuery bool
	seed     int
	pick     int
	order    []sortOrder
	offset   int // <= 0: no offset
	limit    int // <= 0: no limit

	contextMeta []*meta.Meta
	PreMatch    MetaMatchFunc // Precondition for Match and Retrieve
	Terms       []CompiledTerm
}

// MetaMatchFunc is a function determine whethe some metadata should be selected or not.
type MetaMatchFunc func(*meta.Meta) bool

func matchAlways(*meta.Meta) bool { return true }
func matchNever(*meta.Meta) bool  { return false }

// CompiledTerm is the preprocessed sequence of conjugated search terms.
type CompiledTerm struct {
	Match    MetaMatchFunc     // Match on metadata
	Retrieve RetrievePredicate // Retrieve from full-text search
}

// RetrievePredicate returns true, if the given Zid is contained in the (full-text) search.
type RetrievePredicate func(id.Zid) bool

func (c *Compiled) isDeterministic() bool { return c.seed > 0 }

// AfterSearch applies all terms to the metadata list that was searched.
//
// This includes sorting, offset, limit, and picking.
func (c *Compiled) AfterSearch(metaList []*meta.Meta) []*meta.Meta {
	if len(metaList) == 0 {
		return metaList
	}

	if !c.hasQuery {
		return sortMetaByZid(metaList)
	}

	metaList = c.doPick(metaList)

	if len(c.order) == 0 {
		if c.pick <= 0 {
			metaList = c.sortMetaByDefault(metaList)
		}
	} else if c.order[0].isRandom() {
		metaList = c.sortRandomly(metaList)
	} else {
		sort.Slice(metaList, createSortFunc(c.order, metaList))
	}

	if c.offset > 0 {
		if c.offset > len(metaList) {
			return nil
		}
		metaList = metaList[c.offset:]
	}
	return c.doLimit(metaList)
}

func (c *Compiled) sortMetaByDefault(metaList []*meta.Meta) []*meta.Meta {
	if len(c.contextMeta) == 0 {
		return sortMetaByZid(metaList)
	}
	contextOrder := make(map[id.Zid]int, len(c.contextMeta))
	for pos, m := range c.contextMeta {
		contextOrder[m.Zid] = pos + 1
	}
	sort.Slice(metaList, func(i, j int) bool { return contextOrder[metaList[i].Zid] < contextOrder[metaList[j].Zid] })
	return metaList
}

func (c *Compiled) doPick(metaList []*meta.Meta) []*meta.Meta {
	pick := c.pick
	if pick <= 0 {
		return metaList
	}
	if limit := c.limit; limit > 0 && limit < pick {
		pick = limit
	}
	if pick >= len(metaList) {
		return c.doRandom(metaList)
	}
	return c.doPickN(metaList, pick)
}
func (c *Compiled) doPickN(metaList []*meta.Meta, pick int) []*meta.Meta {
	if c.isDeterministic() {
		metaList = sortMetaByZid(metaList)
	}
	rnd := c.newRandom()
	result := make([]*meta.Meta, pick)
	for i := 0; i < pick; i++ {
		last := len(metaList) - i
		n := rnd.Intn(last)
		result[i] = metaList[n]
		metaList[n] = metaList[last-1]
		metaList[last-1] = nil
	}
	return result
}

func (c *Compiled) sortRandomly(metaList []*meta.Meta) []*meta.Meta {
	// Optimization: RANDOM LIMIT n, where n < len(metaList) is essentially a PICK n.
	if limit := c.limit; limit > 0 && limit < len(metaList) {
		return c.doPickN(metaList, limit)
	}
	return c.doRandom(metaList)
}

func (c *Compiled) doRandom(metaList []*meta.Meta) []*meta.Meta {
	if c.isDeterministic() {
		metaList = sortMetaByZid(metaList)
	}
	rnd := c.newRandom()
	rnd.Shuffle(
		len(metaList),
		func(i, j int) { metaList[i], metaList[j] = metaList[j], metaList[i] },
	)
	return metaList
}

func (c *Compiled) newRandom() *rand.Rand {
	seed := c.seed
	if seed <= 0 {
		seed = rand.Intn(10000) + 10001
	}
	return rand.New(rand.NewSource(int64(seed)))
}

// doLimit returns only s.GetLimit() elements of the given list.
func (c *Compiled) doLimit(metaList []*meta.Meta) []*meta.Meta {
	if c == nil {
		return metaList
	}
	if limit := c.limit; limit > 0 && limit < len(metaList) {
		return metaList[:limit]
	}
	return metaList
}

func sortMetaByZid(metaList []*meta.Meta) []*meta.Meta {
	sort.Slice(metaList, func(i, j int) bool { return metaList[i].Zid > metaList[j].Zid })
	return metaList
}

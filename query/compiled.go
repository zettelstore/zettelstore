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

	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
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

// Result returns a result of the compiled search, that is achievable without iterating through a box.
func (c *Compiled) Result() []*meta.Meta {
	if len(c.contextMeta) == 0 {
		// nil -> no context
		// empty slice -> nothing found
		return c.contextMeta
	}
	result := make([]*meta.Meta, 0, len(c.contextMeta))
	for _, m := range c.contextMeta {
		for _, term := range c.Terms {
			if term.Match(m) {
				result = append(result, m)
				break
			}
		}
	}
	result = c.pickElements(result)
	result = c.sortElements(result)
	result = c.offsetElements(result)
	return limitElements(result, c.limit)
}

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

	if c.isDeterministic() {
		// We need to sort to make it deterministic
		if len(c.order) == 0 || c.order[0].isRandom() {
			metaList = sortMetaByZid(metaList)
		} else {
			sort.Slice(metaList, createSortFunc(c.order, metaList))
		}
	}
	metaList = c.pickElements(metaList)
	if c.isDeterministic() {
		if len(c.order) > 0 && c.order[0].isRandom() {
			metaList = c.sortRandomly(metaList)
		}
	} else {
		metaList = c.sortElements(metaList)
	}
	metaList = c.offsetElements(metaList)
	return limitElements(metaList, c.limit)
}

func (c *Compiled) sortElements(metaList []*meta.Meta) []*meta.Meta {
	if len(c.order) > 0 {
		if c.order[0].isRandom() {
			metaList = c.sortRandomly(metaList)
		} else {
			sort.Slice(metaList, createSortFunc(c.order, metaList))
		}
	}
	return metaList
}

func (c *Compiled) offsetElements(metaList []*meta.Meta) []*meta.Meta {
	if c.offset == 0 {
		return metaList
	}
	if c.offset > len(metaList) {
		return nil
	}
	return metaList[c.offset:]
}

func (c *Compiled) pickElements(metaList []*meta.Meta) []*meta.Meta {
	count := c.pick
	if count <= 0 || count >= len(metaList) {
		return metaList
	}
	if limit := c.limit; limit > 0 && limit < count {
		count = limit
		c.limit = 0
	}

	order := make([]int, len(metaList))
	for i := 0; i < len(metaList); i++ {
		order[i] = i
	}
	rnd := c.newRandom()
	picked := make([]int, count)
	for i := 0; i < count; i++ {
		last := len(order) - i
		n := rnd.Intn(last)
		picked[i] = order[n]
		order[n] = order[last-1]
	}
	order = nil
	sort.Ints(picked)
	result := make([]*meta.Meta, count)
	for i, p := range picked {
		result[i] = metaList[p]
	}
	return result
}

func (c *Compiled) sortRandomly(metaList []*meta.Meta) []*meta.Meta {
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

func limitElements(metaList []*meta.Meta, limit int) []*meta.Meta {
	if limit > 0 && limit < len(metaList) {
		return metaList[:limit]
	}
	return metaList
}

func sortMetaByZid(metaList []*meta.Meta) []*meta.Meta {
	sort.Slice(metaList, func(i, j int) bool { return metaList[i].Zid > metaList[j].Zid })
	return metaList
}

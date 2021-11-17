//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"
	"strings"
	"unicode"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/search"
)

// UnlinkedReferencesPort is the interface used by this use case.
type UnlinkedReferencesPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
	SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error)
}

// UnlinkedReferences is the data for this use case.
type UnlinkedReferences struct {
	port    UnlinkedReferencesPort
	encText encoder.Encoder
}

// NewUnlinkedReferences creates a new use case.
func NewUnlinkedReferences(port UnlinkedReferencesPort) UnlinkedReferences {
	return UnlinkedReferences{
		port:    port,
		encText: encoder.Create(api.EncoderText, nil),
	}
}

// Run executes the usecase with already evaluated title value.
func (uc *UnlinkedReferences) Run(ctx context.Context, zid id.Zid, title string) ([]*meta.Meta, error) {
	s := uc.searchTextWords(title)
	if s == nil {
		return nil, nil
	}
	candidates, err := uc.port.SelectMeta(ctx, s)
	if err != nil {
		return nil, err
	}
	candidates = uc.filterCandidates(zid, candidates)
	return candidates, nil
}

func (uc *UnlinkedReferences) searchTextWords(text string) *search.Search {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.In(r, unicode.C, unicode.P, unicode.Z)
	})
	var s *search.Search
	for _, word := range words {
		s = s.AddExpr("", "="+word)
	}
	return s
}

func (uc *UnlinkedReferences) filterCandidates(zid id.Zid, candidates []*meta.Meta) []*meta.Meta {
	result := make([]*meta.Meta, 0, len(candidates))
	for _, cand := range candidates {
		if zid == cand.Zid || linksTo(zid, cand) {
			continue
		}
		result = append(result, cand)
	}
	return result
}

// linksTo returns true, if any metadata from source mentions zid
func linksTo(zid id.Zid, source *meta.Meta) bool {
	zidVal := zid.String()
	for _, pair := range source.PairsRest(true) {
		key := pair.Key
		switch meta.Type(key) {
		case meta.TypeID:
			if zidVal == pair.Value {
				return true
			}
		case meta.TypeIDSet:
			for _, val := range meta.ListFromValue(pair.Value) {
				if zidVal == val {
					return true
				}
			}
		}
	}
	return false
}

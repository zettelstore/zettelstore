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
	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/parser"
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
	port     UnlinkedReferencesPort
	rtConfig config.Config
	encText  encoder.Encoder
}

// NewUnlinkedReferences creates a new use case.
func NewUnlinkedReferences(port UnlinkedReferencesPort, rtConfig config.Config) UnlinkedReferences {
	return UnlinkedReferences{
		port:     port,
		rtConfig: rtConfig,
		encText:  encoder.Create(api.EncoderText, nil),
	}
}

// Run executes the usecase with already evaluated title value.
func (uc *UnlinkedReferences) Run(ctx context.Context, zid id.Zid, title string) ([]*meta.Meta, error) {
	words := makeWords(title)
	var s *search.Search
	for _, word := range words {
		s = s.AddExpr("", "="+word)
	}
	if s == nil {
		return nil, nil
	}
	candidates, err := uc.port.SelectMeta(ctx, s)
	if err != nil {
		return nil, err
	}
	candidates = uc.filterCandidates(ctx, zid, candidates, words)
	return candidates, nil
}

func makeWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return unicode.In(r, unicode.C, unicode.P, unicode.Z)
	})
}

func (uc *UnlinkedReferences) filterCandidates(ctx context.Context, zid id.Zid, candidates []*meta.Meta, words []string) []*meta.Meta {
	result := make([]*meta.Meta, 0, len(candidates))
	for _, cand := range candidates {
		if zid == cand.Zid || linksTo(zid, cand) {
			continue
		}
		zettel, err := uc.port.GetZettel(ctx, cand.Zid)
		if err != nil {
			continue
		}
		zn, err := parser.ParseZettel(zettel, "", uc.rtConfig), nil
		if err != nil {
			continue
		}
		evaluator.EvaluateZettel(ctx, uc.port, nil, uc.rtConfig, zn)
		if !containsWords(zn, words) {
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

func containsWords(zn *ast.ZettelNode, words []string) bool {
	v := unlinkedVisitor{
		words: words,
		found: false,
	}
	v.text = v.joinWords(words)
	ast.Walk(&v, zn.Ast)
	return v.found
}

func (*unlinkedVisitor) joinWords(words []string) string {
	return " " + strings.ToLower(strings.Join(words, " ")) + " "
}

type unlinkedVisitor struct {
	words []string
	text  string
	found bool
}

func (v *unlinkedVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.InlineListNode:
		v.checkWords(n)
		return nil
	case *ast.HeadingNode:
		return nil
	case *ast.LinkNode, *ast.EmbedNode, *ast.CiteNode:
		return nil
	}
	return v
}

func (v *unlinkedVisitor) checkWords(iln *ast.InlineListNode) {
	if len(iln.List) < 2*len(v.words)-1 {
		return
	}
	for _, text := range v.splitInlineTextList(iln) {
		if strings.Contains(text, v.text) {
			v.found = true
		}
	}
}

func (v *unlinkedVisitor) splitInlineTextList(iln *ast.InlineListNode) []string {
	var result []string
	var curList []string
	for _, in := range iln.List {
		switch n := in.(type) {
		case *ast.TextNode:
			curList = append(curList, makeWords(n.Text)...)
		case *ast.SpaceNode:
		default:
			if curList != nil {
				result = append(result, v.joinWords(curList))
				curList = nil
			}
		}
	}
	if curList != nil {
		result = append(result, v.joinWords(curList))
	}
	return result
}

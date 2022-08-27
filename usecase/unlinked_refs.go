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
	"strings"
	"unicode"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder/textenc"
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
	encText  *textenc.Encoder
}

// NewUnlinkedReferences creates a new use case.
func NewUnlinkedReferences(port UnlinkedReferencesPort, rtConfig config.Config) UnlinkedReferences {
	return UnlinkedReferences{
		port:     port,
		rtConfig: rtConfig,
		encText:  textenc.Create(),
	}
}

// Run executes the usecase with already evaluated title value.
func (uc *UnlinkedReferences) Run(ctx context.Context, phrase string, s *search.Search) ([]*meta.Meta, error) {
	words := makeWords(phrase)
	if len(words) == 0 {
		return nil, nil
	}
	var sb strings.Builder
	for _, word := range words {
		sb.WriteString(" :")
		sb.WriteString(word)
	}
	s = s.Parse(sb.String())

	// Limit applies to the filtering process, not to SelectMeta
	limit := s.GetLimit()
	s = s.SetLimit(0)

	candidates, err := uc.port.SelectMeta(ctx, s)
	if err != nil {
		return nil, err
	}
	s = s.SetLimit(limit) // Restore limit
	return s.Limit(uc.filterCandidates(ctx, candidates, words)), nil
}

func makeWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return unicode.In(r, unicode.C, unicode.P, unicode.Z)
	})
}

func (uc *UnlinkedReferences) filterCandidates(ctx context.Context, candidates []*meta.Meta, words []string) []*meta.Meta {
	result := make([]*meta.Meta, 0, len(candidates))
candLoop:
	for _, cand := range candidates {
		zettel, err := uc.port.GetZettel(ctx, cand.Zid)
		if err != nil {
			continue
		}
		v := unlinkedVisitor{
			words: words,
			found: false,
		}
		v.text = v.joinWords(words)

		for _, pair := range zettel.Meta.Pairs() {
			if meta.Type(pair.Key) != meta.TypeZettelmarkup {
				continue
			}
			is := parser.ParseMetadata(pair.Value)
			evaluator.EvaluateInline(ctx, uc.port, uc.rtConfig, &is)
			ast.Walk(&v, &is)
			if v.found {
				result = append(result, cand)
				continue candLoop
			}
		}

		syntax := zettel.Meta.GetDefault(api.KeySyntax, "")
		if !parser.IsTextParser(syntax) {
			continue
		}
		zn, err := parser.ParseZettel(ctx, zettel, syntax, nil), nil
		if err != nil {
			continue
		}
		evaluator.EvaluateZettel(ctx, uc.port, uc.rtConfig, zn)
		ast.Walk(&v, &zn.Ast)
		if v.found {
			result = append(result, cand)
		}
	}
	return result
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
	case *ast.InlineSlice:
		v.checkWords(n)
		return nil
	case *ast.HeadingNode:
		return nil
	case *ast.LinkNode, *ast.EmbedRefNode, *ast.EmbedBLOBNode, *ast.CiteNode:
		return nil
	}
	return v
}

func (v *unlinkedVisitor) checkWords(is *ast.InlineSlice) {
	if len(*is) < 2*len(v.words)-1 {
		return
	}
	for _, text := range v.splitInlineTextList(is) {
		if strings.Contains(text, v.text) {
			v.found = true
		}
	}
}

func (v *unlinkedVisitor) splitInlineTextList(is *ast.InlineSlice) []string {
	var result []string
	var curList []string
	for _, in := range *is {
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

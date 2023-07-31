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
	"strings"
	"unicode"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// UnlinkedReferencesPort is the interface used by this use case.
type UnlinkedReferencesPort interface {
	SelectMeta(ctx context.Context, metaSeq []*meta.Meta, q *query.Query) ([]*meta.Meta, error)
}

// UnlinkedReferences is the data for this use case.
type UnlinkedReferences struct {
	port        UnlinkedReferencesPort
	ucGetZettel *GetZettel
	ucQuery     *Query
	rtConfig    config.Config
	encText     *textenc.Encoder
}

// NewUnlinkedReferences creates a new use case.
func NewUnlinkedReferences(port UnlinkedReferencesPort, ucGetZettel *GetZettel, ucQuery *Query, rtConfig config.Config) UnlinkedReferences {
	return UnlinkedReferences{
		port:        port,
		ucGetZettel: ucGetZettel,
		ucQuery:     ucQuery,
		rtConfig:    rtConfig,
		encText:     textenc.Create(),
	}
}

// Run executes the usecase with already evaluated title value.
func (uc *UnlinkedReferences) Run(ctx context.Context, phrase string, q *query.Query) ([]*meta.Meta, error) {
	words := makeWords(phrase)
	if len(words) == 0 {
		return nil, nil
	}
	var sb strings.Builder
	for _, word := range words {
		sb.WriteString(" :")
		sb.WriteString(word)
	}
	q = q.Parse(sb.String())

	// Limit applies to the filtering process, not to SelectMeta
	q, prevLimit := q.SetLimit(0)

	candidates, err := uc.port.SelectMeta(ctx, nil, q)
	if err != nil {
		return nil, err
	}
	q, _ = q.SetLimit(prevLimit) // Restore limit
	return q.Limit(uc.filterCandidates(ctx, candidates, words)), nil
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
		zettel, err := uc.ucGetZettel.Run(ctx, cand.Zid)
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
			evaluator.EvaluateInline(ctx, uc, uc.rtConfig, &is)
			ast.Walk(&v, &is)
			if v.found {
				result = append(result, cand)
				continue candLoop
			}
		}

		syntax := zettel.Meta.GetDefault(api.KeySyntax, "")
		if !parser.IsASTParser(syntax) {
			continue
		}
		zn, err := parser.ParseZettel(ctx, zettel, syntax, uc.rtConfig), nil
		if err != nil {
			continue
		}
		evaluator.EvaluateZettel(ctx, uc, uc.rtConfig, zn)
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

// GetZettel retrieves the full zettel of a given zettel identifier.
func (uc *UnlinkedReferences) GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error) {
	return uc.ucGetZettel.Run(ctx, zid)
}

// QueryMeta returns a list of metadata that comply to the given selection criteria.
func (uc *UnlinkedReferences) QueryMeta(ctx context.Context, q *query.Query) ([]*meta.Meta, error) {
	return uc.ucQuery.Run(ctx, q)
}

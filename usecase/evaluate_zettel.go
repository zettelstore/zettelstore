//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the zettelstore.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
)

// EvaluateZettel is the data for this use case.
type EvaluateZettel struct {
	getMeta     GetMeta
	parseZettel ParseZettel
}

// NewEvaluateZettel creates a new use case.
func NewEvaluateZettel(getMeta GetMeta, parseZettel ParseZettel) EvaluateZettel {
	return EvaluateZettel{
		getMeta:     getMeta,
		parseZettel: parseZettel,
	}
}

// Run executes the use case.
func (uc *EvaluateZettel) Run(ctx context.Context, zid id.Zid, env *EvaluateEnvironment) (*ast.ZettelNode, error) {
	zn, err := uc.parseZettel.Run(ctx, zid, env.Syntax)
	if err != nil {
		return nil, err
	}

	e := evaluator{
		ctx:     ctx,
		getMeta: uc.getMeta,
		env:     env,
	}
	ast.WalkBlockSlice(&e, zn.Ast)
	return zn, nil
}

// EvaluateEnvironment contains values to control the evaluation.
type EvaluateEnvironment struct {
	Syntax        string
	Encoding      api.EncodingEnum
	Key           byte
	Part          string
	GetURLPrefix  func() string
	NewURLBuilder func(key byte) *api.URLBuilder
}

type evaluator struct {
	ctx     context.Context
	getMeta GetMeta
	env     *EvaluateEnvironment
}

func (e *evaluator) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ParaNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	case *ast.RegionNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	case *ast.HeadingNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	case *ast.DescriptionListNode:
		for _, desc := range n.Descriptions {
			desc.Term = e.walkInlineSlice(desc.Term)
			for _, dns := range desc.Descriptions {
				ast.WalkDescriptionSlice(e, dns)
			}
		}
	case *ast.TableNode:
		for _, cell := range n.Header {
			e.walkInlineSlice(cell.Inlines)
		}
		for _, row := range n.Rows {
			for _, cell := range row {
				e.walkInlineSlice(cell.Inlines)
			}
		}
	case *ast.LinkNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	case *ast.EmbedNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	case *ast.CiteNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	case *ast.FootnoteNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	case *ast.FormatNode:
		n.Inlines = e.walkInlineSlice(n.Inlines)
	default:
		return e
	}
	return nil
}

func (e *evaluator) walkInlineSlice(ins ast.InlineSlice) ast.InlineSlice {
	for i, in := range ins {
		ast.Walk(e, in)
		switch n := in.(type) {
		case *ast.LinkNode:
			ins[i] = e.evalLinkNode(n)
		case *ast.EmbedNode:
			ins[i] = e.evalEmbedNode(n)
		}
	}
	return ins
}

func (e *evaluator) evalLinkNode(ln *ast.LinkNode) ast.InlineNode {
	origRef := ln.Ref
	if origRef == nil {
		return ln
	}
	if origRef.State == ast.RefStateBased {
		newLink := *ln
		urlPrefix := e.env.GetURLPrefix()
		newRef := ast.ParseReference(urlPrefix + origRef.Value[1:])
		newRef.State = ast.RefStateHosted
		newLink.Ref = newRef
		return &newLink
	}
	if origRef.State != ast.RefStateZettel {
		return ln
	}
	zid, err := id.Parse(origRef.URL.Path)
	if err != nil {
		panic(err)
	}
	_, err = e.getMeta.Run(box.NoEnrichContext(e.ctx), zid)
	if errors.Is(err, &box.ErrNotAllowed{}) {
		return &ast.FormatNode{
			Kind:    ast.FormatSpan,
			Attrs:   ln.Attrs,
			Inlines: ln.Inlines,
		}
	}
	var newRef *ast.Reference
	if err == nil {
		ub := e.env.NewURLBuilder(e.env.Key).SetZid(zid)
		if part := e.env.Part; part != "" {
			ub.AppendQuery(api.QueryKeyPart, part)
		}
		if enc := e.env.Encoding; enc != api.EncoderUnknown {
			ub.AppendQuery(api.QueryKeyEncoding, enc.String())
		}
		if fragment := origRef.URL.EscapedFragment(); fragment != "" {
			ub.SetFragment(fragment)
		}

		newRef = ast.ParseReference(ub.String())
		newRef.State = ast.RefStateFound
	} else {
		newRef = ast.ParseReference(origRef.Value)
		newRef.State = ast.RefStateBroken
	}
	newLink := *ln
	newLink.Ref = newRef
	return &newLink
}

func (e *evaluator) evalEmbedNode(en *ast.EmbedNode) ast.InlineNode {
	switch en.Material.(type) {
	case *ast.ReferenceMaterialNode:
	case *ast.BLOBMaterialNode:
		return en
	default:
		panic(fmt.Sprintf("Unknown material type %t for %v", en.Material, en.Material))
	}

	origRef := en.Material.(*ast.ReferenceMaterialNode)
	switch origRef.Ref.State {
	case ast.RefStateInvalid:
		return e.createZettelEmbed(en, id.EmojiZid, ast.RefStateInvalid)
	case ast.RefStateZettel:
		zid, err := id.Parse(origRef.Ref.Value)
		if err != nil {
			panic(err)
		}
		_, err = e.getMeta.Run(box.NoEnrichContext(e.ctx), zid)
		if err != nil {
			return e.createZettelEmbed(en, id.EmojiZid, ast.RefStateBroken)
		}
		return e.createZettelEmbed(en, zid, ast.RefStateFound)
	}
	return en
}

func (e *evaluator) createZettelEmbed(en *ast.EmbedNode, zid id.Zid, state ast.RefState) *ast.EmbedNode {
	switch e.env.Encoding {
	case api.EncoderDJSON, api.EncoderHTML:
		newRef := ast.ParseReference(e.env.NewURLBuilder('z').SetZid(zid).AppendQuery(api.QueryKeyRaw, "").String())
		newRef.State = state
		newEmbed := *en
		newEmbed.Material = &ast.ReferenceMaterialNode{Ref: newRef}
		return &newEmbed
	}
	return en
}

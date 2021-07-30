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

	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/parser"
)

// EvaluateZettel is the data for this use case.
type EvaluateZettel struct {
	rtConfig  config.Config
	getZettel GetZettel
	getMeta   GetMeta
}

// NewEvaluateZettel creates a new use case.
func NewEvaluateZettel(rtConfig config.Config, getZettel GetZettel, getMeta GetMeta) EvaluateZettel {
	return EvaluateZettel{
		rtConfig:  rtConfig,
		getZettel: getZettel,
		getMeta:   getMeta,
	}
}

// Run executes the use case.
func (uc *EvaluateZettel) Run(ctx context.Context, zid id.Zid, env *EvaluateEnvironment) (*ast.ZettelNode, error) {
	zettel, err := uc.getZettel.Run(ctx, zid)
	if err != nil {
		return nil, err
	}
	zn, err := parser.ParseZettel(zettel, env.Syntax, uc.rtConfig), nil
	if err != nil {
		return nil, err
	}

	e := evaluator{
		ctx:     ctx,
		getMeta: uc.getMeta,
		env:     env,
	}
	ast.Walk(&e, zn.Ast)
	return zn, nil
}

// EvaluateEnvironment contains values to control the evaluation.
type EvaluateEnvironment struct {
	Syntax       string
	GetHostedRef func(string) *ast.Reference
	GetFoundRef  func(zid id.Zid, fragment string) *ast.Reference
	GetImageRef  func(zid id.Zid, state ast.RefState) *ast.Reference
}

type evaluator struct {
	ctx     context.Context
	getMeta GetMeta
	env     *EvaluateEnvironment
}

func (e *evaluator) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.InlineListNode:
		e.visitInlineList(n)
	default:
		return e
	}
	return nil
}

func (e *evaluator) visitInlineList(iln *ast.InlineListNode) {
	for i, in := range iln.List {
		ast.Walk(e, in)
		switch n := in.(type) {
		case *ast.LinkNode:
			iln.List[i] = e.evalLinkNode(n)
		case *ast.EmbedNode:
			iln.List[i] = e.evalEmbedNode(n)
		}
	}
}

func (e *evaluator) evalLinkNode(ln *ast.LinkNode) ast.InlineNode {
	ref := ln.Ref
	if ref == nil {
		return ln
	}
	if ref.State == ast.RefStateBased {
		if ghr := e.env.GetHostedRef; ghr != nil {
			ln.Ref = ghr(ref.Value[1:])
		}
		return ln
	}
	if ref.State != ast.RefStateZettel {
		return ln
	}
	zid, err := id.Parse(ref.URL.Path)
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
	} else if err != nil {
		ln.Ref.State = ast.RefStateBroken
		return ln
	}

	if gfr := e.env.GetFoundRef; gfr != nil {
		ln.Ref = gfr(zid, ref.URL.EscapedFragment())
	}
	return ln
}

func (e *evaluator) evalEmbedNode(en *ast.EmbedNode) ast.InlineNode {
	switch en.Material.(type) {
	case *ast.ReferenceMaterialNode:
	case *ast.BLOBMaterialNode:
		return en
	default:
		panic(fmt.Sprintf("Unknown material type %t for %v", en.Material, en.Material))
	}

	ref := en.Material.(*ast.ReferenceMaterialNode)
	switch ref.Ref.State {
	case ast.RefStateInvalid:
		return e.createZettelEmbed(en, ref, id.EmojiZid, ast.RefStateInvalid)
	case ast.RefStateZettel:
		zid, err := id.Parse(ref.Ref.Value)
		if err != nil {
			panic(err)
		}
		_, err = e.getMeta.Run(box.NoEnrichContext(e.ctx), zid)
		if err != nil {
			return e.createZettelEmbed(en, ref, id.EmojiZid, ast.RefStateBroken)
		}
		return e.createZettelEmbed(en, ref, zid, ast.RefStateFound)
	}
	return en
}

func (e *evaluator) createZettelEmbed(
	en *ast.EmbedNode, ref *ast.ReferenceMaterialNode, zid id.Zid, state ast.RefState) *ast.EmbedNode {

	if gir := e.env.GetImageRef; gir != nil {
		en.Material = &ast.ReferenceMaterialNode{Ref: gir(zid, state)}
		return en
	}
	ref.Ref.State = state
	en.Material = ref
	return en
}

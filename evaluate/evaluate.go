//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package evaluate interprets and evaluates the AST.
package evaluate

import (
	"context"
	"errors"
	"fmt"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
)

// Environment contains values to control the evaluation.
type Environment struct {
	Config       config.Config
	Syntax       string
	GetHostedRef func(string) *ast.Reference
	GetFoundRef  func(zid id.Zid, fragment string) *ast.Reference
	GetImageRef  func(zid id.Zid, state ast.RefState) *ast.Reference
}

// Port contains all methods to retrieve zettel (or part of it) to evaluate a zettel.
type Port interface {
	GetMeta(context.Context, id.Zid) (*meta.Meta, error)
}

// Evaluate the given AST in the given context, with the given ports, and the
// given environment.
func Evaluate(ctx context.Context, port Port, env *Environment, zn *ast.ZettelNode) {
	e := evaluator{
		ctx:  ctx,
		port: port,
		env:  env,
	}
	ast.Walk(&e, zn.Ast)
}

type evaluator struct {
	ctx  context.Context
	port Port
	env  *Environment
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
	_, err = e.port.GetMeta(box.NoEnrichContext(e.ctx), zid)
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
		return e.createImage(en, ref, id.EmojiZid, ast.RefStateInvalid)
	case ast.RefStateZettel, ast.RefStateFound:
		zid, err := id.Parse(ref.Ref.Value)
		if err != nil {
			panic(err)
		}
		m, err := e.port.GetMeta(box.NoEnrichContext(e.ctx), zid)
		if err != nil {
			return e.createImage(en, ref, id.EmojiZid, ast.RefStateBroken)
		}
		return e.createEmbed(en, ref, m)
	case ast.RefStateSelf:
		panic("TODO: Zettel references itself")
	case ast.RefStateBroken:
		return e.createImage(en, ref, id.EmojiZid, ast.RefStateBroken)
	case ast.RefStateHosted, ast.RefStateBased, ast.RefStateExternal:
		return en
	default:
		panic(fmt.Sprintf("Unknown state %v for reference %v", ref.Ref.State, ref.Ref))
	}
}

func (e *evaluator) getSyntax(m *meta.Meta) string {
	if cfg := e.env.Config; cfg != nil {
		return config.GetSyntax(m, cfg)
	}
	return m.GetDefault(meta.KeySyntax, "")
}
func (e *evaluator) createEmbed(en *ast.EmbedNode, ref *ast.ReferenceMaterialNode, m *meta.Meta) ast.InlineNode {
	syntax := e.getSyntax(m)
	if parser.IsImageFormat(syntax) {
		return e.createImage(en, ref, m.Zid, ast.RefStateFound)
	}
	if parser.IsTextParser(syntax) {
		// Search for text to be embedded.
		return en
	}

	// Not embeddable.
	return createErrorText(en, "Cannot", "embed (syntax="+syntax+"):")
}

func (e *evaluator) createImage(
	en *ast.EmbedNode, ref *ast.ReferenceMaterialNode, zid id.Zid, state ast.RefState) *ast.EmbedNode {

	if gir := e.env.GetImageRef; gir != nil {
		en.Material = &ast.ReferenceMaterialNode{Ref: gir(zid, state)}
		return en
	}
	ref.Ref.State = state
	en.Material = ref
	return en
}

func createErrorText(en *ast.EmbedNode, msgWords ...string) ast.InlineNode {
	ref := en.Material.(*ast.ReferenceMaterialNode)
	ln := &ast.LinkNode{
		Ref:     ref.Ref,
		Inlines: ast.CreateInlineListNodeFromWords(ref.Ref.String()),
		OnlyRef: true,
	}
	text := ast.CreateInlineListNodeFromWords(msgWords...)
	text.Append(&ast.SpaceNode{Lexeme: " "}, ln)
	fn := &ast.FormatNode{
		Kind:    ast.FormatMonospace,
		Inlines: text,
	}
	fn = &ast.FormatNode{
		Kind:    ast.FormatBold,
		Inlines: ast.CreateInlineListNode(fn),
	}
	fn.Attrs = fn.Attrs.AddClass("error")
	return fn
}

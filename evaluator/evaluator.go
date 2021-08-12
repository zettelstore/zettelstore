//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package evaluator interprets and evaluates the AST.
package evaluator

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/parser/cleaner"
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
	GetZettel(context.Context, id.Zid) (domain.Zettel, error)
}

// Evaluate the given AST in the given context, with the given ports, and the
// given environment.
func Evaluate(ctx context.Context, port Port, env *Environment, rtConfig config.Config, zn *ast.ZettelNode) {
	e := evaluator{
		ctx:        ctx,
		port:       port,
		env:        env,
		rtConfig:   rtConfig,
		astMap:     map[id.Zid]*ast.ZettelNode{},
		embedMap:   map[string]*ast.InlineListNode{},
		embedCount: 0,
		marker:     &ast.ZettelNode{},
	}
	ast.Walk(&e, zn.Ast)
	cleaner.CleanBlockList(zn.Ast)
}

type evaluator struct {
	ctx        context.Context
	port       Port
	env        *Environment
	rtConfig   config.Config
	astMap     map[id.Zid]*ast.ZettelNode
	marker     *ast.ZettelNode
	embedMap   map[string]*ast.InlineListNode
	embedCount int
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
	for i := 0; i < len(iln.List); i++ {
		in := iln.List[i]
		ast.Walk(e, in)
		switch n := in.(type) {
		case *ast.LinkNode:
			iln.List[i] = e.evalLinkNode(n)
		case *ast.EmbedNode:
			in := e.evalEmbedNode(n)
			if ln, ok := in.(*ast.InlineListNode); ok {
				iln.List = replaceWithInlineNodes(iln.List, i, ln.List)
				i += len(ln.List) - 1
			} else {
				iln.List[i] = in
			}
		}
	}
}

func replaceWithInlineNodes(ins []ast.InlineNode, i int, replaceIns []ast.InlineNode) []ast.InlineNode {
	if len(replaceIns) == 1 {
		ins[i] = replaceIns[0]
		return ins
	}
	newIns := make([]ast.InlineNode, 0, len(ins)+len(replaceIns)-1)
	if i > 0 {
		newIns = append(newIns, ins[:i]...)
	}
	if len(replaceIns) > 0 {
		newIns = append(newIns, replaceIns...)
	}
	if i+1 < len(ins) {
		newIns = append(newIns, ins[i+1:]...)
	}
	return newIns
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
	case ast.RefStateSelf:
		return createErrorText(en, "Self", "embed", "reference:")
	case ast.RefStateBroken:
		return e.createImage(en, ref, id.EmojiZid, ast.RefStateBroken)
	case ast.RefStateHosted, ast.RefStateBased, ast.RefStateExternal:
		return en
	default:
		panic(fmt.Sprintf("Unknown state %v for reference %v", ref.Ref.State, ref.Ref))
	}

	zid, err := id.Parse(ref.Ref.URL.Path)
	if err != nil {
		panic(err)
	}
	m, err := e.port.GetMeta(box.NoEnrichContext(e.ctx), zid)
	if err != nil {
		return e.createImage(en, ref, id.EmojiZid, ast.RefStateBroken)
	}

	syntax := e.getSyntax(m)
	if parser.IsImageFormat(syntax) {
		return e.createImage(en, ref, m.Zid, ast.RefStateFound)
	}
	if !parser.IsTextParser(syntax) {
		// Not embeddable.
		return createErrorText(en, "Not", "embeddable (syntax="+syntax+"):")
	}

	zn, ok := e.astMap[zid]
	if zn == e.marker {
		return createErrorText(en, "Recursive", "transclusion:")
	}
	if !ok {
		e.astMap[zid] = e.marker

		zn, err = e.evaluateEmbeddedZettel(zid, syntax)
		if err != nil {
			delete(e.astMap, zid)
			return createErrorText(en, "Cannot", "parse", "zettel (error="+err.Error()+"):")
		}
		e.astMap[zid] = zn
	}

	result, ok := e.embedMap[ref.Ref.Value]
	if !ok {
		// Search for text to be embedded.
		result = findInlineList(zn.Ast, ref.Ref.URL.Fragment)
		e.embedMap[ref.Ref.Value] = result
		if result.IsEmpty() {
			return createErrorText(en, "Nothing", "to", "transclude:")
		}
	}

	e.embedCount++
	if maxTrans := e.rtConfig.GetMaxTransclusions(); e.embedCount > maxTrans {
		return createErrorText(en, "Too", "many", "transclusions ("+strconv.Itoa(maxTrans)+"):")
	}
	return result
}

func (e *evaluator) getSyntax(m *meta.Meta) string {
	if cfg := e.env.Config; cfg != nil {
		return config.GetSyntax(m, cfg)
	}
	return m.GetDefault(meta.KeySyntax, "")
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

func (e *evaluator) evaluateEmbeddedZettel(zid id.Zid, syntax string) (*ast.ZettelNode, error) {
	zettel, err := e.port.GetZettel(e.ctx, zid)
	if err == nil {
		zn := parser.ParseZettel(zettel, syntax, e.rtConfig)
		ast.Walk(e, zn.Ast)
		return zn, nil
	}
	return nil, err
}

func findInlineList(bnl *ast.BlockListNode, fragment string) *ast.InlineListNode {
	if fragment == "" {
		return firstFirstTopLevelParagraph(bnl.List)
	}
	fs := fragmentSearcher{
		fragment: fragment,
		result:   nil,
	}
	ast.Walk(&fs, bnl)
	return fs.result
}

func firstFirstTopLevelParagraph(bns []ast.BlockNode) *ast.InlineListNode {
	for _, bn := range bns {
		pn, ok := bn.(*ast.ParaNode)
		if !ok {
			continue
		}
		inl := pn.Inlines
		if inl != nil && len(inl.List) > 0 {
			return inl
		}
	}
	return nil
}

type fragmentSearcher struct {
	fragment string
	result   *ast.InlineListNode
}

func (fs *fragmentSearcher) Visit(node ast.Node) ast.Visitor {
	if fs.result != nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BlockListNode:
		for i, bn := range n.List {
			if hn, ok := bn.(*ast.HeadingNode); ok && hn.Fragment == fs.fragment {
				fs.result = firstFirstTopLevelParagraph(n.List[i+1:])
				return nil
			}
			ast.Walk(fs, bn)
		}
	case *ast.InlineListNode:
		for i, in := range n.List {
			if mn, ok := in.(*ast.MarkNode); ok && mn.Fragment == fs.fragment {
				fs.result = ast.CreateInlineListNode(skipSpaceNodes(n.List[i+1:])...)
				return nil
			}
			ast.Walk(fs, in)
		}
	default:
		return fs
	}
	return nil
}

func skipSpaceNodes(ins []ast.InlineNode) []ast.InlineNode {
	for i, in := range ins {
		switch in.(type) {
		case *ast.SpaceNode:
		case *ast.BreakNode:
		default:
			return ins[i:]
		}
	}
	return nil
}

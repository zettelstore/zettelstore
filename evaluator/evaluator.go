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

	"zettelstore.de/c/api"
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
	EmbedImage   bool
	GetTagRef    func(string) *ast.Reference
	GetHostedRef func(string) *ast.Reference
	GetFoundRef  func(zid id.Zid, fragment string) *ast.Reference
}

// Port contains all methods to retrieve zettel (or part of it) to evaluate a zettel.
type Port interface {
	GetMeta(context.Context, id.Zid) (*meta.Meta, error)
	GetZettel(context.Context, id.Zid) (domain.Zettel, error)
}

var emptyEnv Environment

// EvaluateZettel evaluates the given zettel in the given context, with the
// given ports, and the given environment.
func EvaluateZettel(ctx context.Context, port Port, env *Environment, rtConfig config.Config, zn *ast.ZettelNode) {
	evaluateNode(ctx, port, env, rtConfig, zn.Ast)
	cleaner.CleanBlockList(zn.Ast)
}

// EvaluateInline evaluates the given inline list in the given context, with
// the given ports, and the given environment.
func EvaluateInline(ctx context.Context, port Port, env *Environment, rtConfig config.Config, iln *ast.InlineListNode) {
	evaluateNode(ctx, port, env, rtConfig, iln)
	cleaner.CleanInlineList(iln)
}

func evaluateNode(ctx context.Context, port Port, env *Environment, rtConfig config.Config, n ast.Node) {
	if env == nil {
		env = &emptyEnv
	}
	e := evaluator{
		ctx:        ctx,
		port:       port,
		env:        env,
		rtConfig:   rtConfig,
		costMap:    map[id.Zid]embedCost{},
		embedMap:   map[string]*ast.InlineListNode{},
		embedCount: 0,
		marker:     &ast.ZettelNode{},
	}
	ast.Walk(&e, n)
}

type evaluator struct {
	ctx        context.Context
	port       Port
	env        *Environment
	rtConfig   config.Config
	costMap    map[id.Zid]embedCost
	marker     *ast.ZettelNode
	embedMap   map[string]*ast.InlineListNode
	embedCount int
}

type embedCost struct {
	zn *ast.ZettelNode
	ec int
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
		case *ast.TagNode:
			iln.List[i] = e.visitTag(n)
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

func (e *evaluator) visitTag(tn *ast.TagNode) ast.InlineNode {
	if gtr := e.env.GetTagRef; gtr != nil {
		fullTag := "#" + tn.Tag
		return &ast.LinkNode{
			Ref:     e.env.GetTagRef(fullTag),
			Inlines: ast.CreateInlineListNodeFromWords(fullTag),
		}
	}
	return tn
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
	if maxTrans := e.rtConfig.GetMaxTransclusions(); e.embedCount > maxTrans {
		e.embedCount = maxTrans + 1 // To prevent e.embedCount from counting
		return createErrorText(en,
			"Too", "many", "transclusions", "(must", "be", "at", "most", strconv.Itoa(maxTrans)+",",
			"see", "runtime", "configuration", "key", "max-transclusions):")
	}
	switch en.Material.(type) {
	case *ast.ReferenceMaterialNode:
	case *ast.BLOBMaterialNode:
		return en
	default:
		panic(fmt.Sprintf("Unknown material type %t for %v", en.Material, en.Material))
	}

	ref := en.Material.(*ast.ReferenceMaterialNode)
	switch ref.Ref.State {
	case ast.RefStateInvalid, ast.RefStateBroken:
		e.embedCount++
		return e.createErrorImage(en)
	case ast.RefStateZettel, ast.RefStateFound:
	case ast.RefStateSelf:
		e.embedCount++
		return createErrorText(en, "Self", "embed", "reference:")
	case ast.RefStateHosted, ast.RefStateBased, ast.RefStateExternal:
		return en
	default:
		panic(fmt.Sprintf("Unknown state %v for reference %v", ref.Ref.State, ref.Ref))
	}

	zid, err := id.Parse(ref.Ref.URL.Path)
	if err != nil {
		panic(err)
	}
	zettel, err := e.port.GetZettel(box.NoEnrichContext(e.ctx), zid)
	if err != nil {
		e.embedCount++
		return e.createErrorImage(en)
	}

	syntax := e.getSyntax(zettel.Meta)
	if parser.IsImageFormat(syntax) {
		return e.embedImage(en, zettel, syntax)
	}
	if !parser.IsTextParser(syntax) {
		// Not embeddable.
		e.embedCount++
		return createErrorText(en, "Not", "embeddable (syntax="+syntax+"):")
	}

	cost, ok := e.costMap[zid]
	zn := cost.zn
	if zn == e.marker {
		e.embedCount++
		return createErrorText(en, "Recursive", "transclusion:")
	}
	if !ok {
		ec := e.embedCount
		e.costMap[zid] = embedCost{zn: e.marker, ec: ec}
		zn = e.evaluateEmbeddedZettel(zettel, syntax)
		e.costMap[zid] = embedCost{zn: zn, ec: e.embedCount - ec}
		e.embedCount = 0 // No stack needed, because embedding is done left-recursive, depth-first.
	}
	e.embedCount++

	result, ok := e.embedMap[ref.Ref.Value]
	if !ok {
		// Search for text to be embedded.
		result = findInlineList(zn.Ast, ref.Ref.URL.Fragment)
		e.embedMap[ref.Ref.Value] = result
	}
	if result.IsEmpty() {
		return &ast.LiteralNode{
			Kind: ast.LiteralComment,
			Text: "Nothing to transclude: " + en.Material.(*ast.ReferenceMaterialNode).Ref.String(),
		}
	}

	if ec := cost.ec; ec > 0 {
		e.embedCount += cost.ec
	}
	return result
}

func (e *evaluator) getSyntax(m *meta.Meta) string {
	if cfg := e.rtConfig; cfg != nil {
		return config.GetSyntax(m, cfg)
	}
	return m.GetDefault(api.KeySyntax, "")
}

func (e *evaluator) createErrorImage(en *ast.EmbedNode) *ast.EmbedNode {
	zid := id.EmojiZid
	if !e.env.EmbedImage {
		en.Material = &ast.ReferenceMaterialNode{Ref: ast.ParseReference(zid.String())}
		return en
	}
	zettel, err := e.port.GetZettel(box.NoEnrichContext(e.ctx), zid)
	if err == nil {
		return doEmbedImage(en, zettel, e.getSyntax(zettel.Meta))
	}
	panic(err)
}

func (e *evaluator) embedImage(en *ast.EmbedNode, zettel domain.Zettel, syntax string) *ast.EmbedNode {
	if e.env.EmbedImage {
		return doEmbedImage(en, zettel, syntax)
	}
	return en
}

func doEmbedImage(en *ast.EmbedNode, zettel domain.Zettel, syntax string) *ast.EmbedNode {
	en.Material = &ast.BLOBMaterialNode{
		Blob:   zettel.Content.AsBytes(),
		Syntax: syntax,
	}
	return en
}

func createErrorText(en *ast.EmbedNode, msgWords ...string) ast.InlineNode {
	ln := linkNodeToEmbeddedReference(en)
	text := ast.CreateInlineListNodeFromWords(msgWords...)
	text.Append(&ast.SpaceNode{Lexeme: " "}, ln, &ast.TextNode{Text: "."}, &ast.SpaceNode{Lexeme: " "})
	fn := &ast.FormatNode{
		Kind:    ast.FormatMonospace,
		Inlines: text,
	}
	fn = &ast.FormatNode{
		Kind:    ast.FormatStrong,
		Inlines: ast.CreateInlineListNode(fn),
	}
	fn.Attrs = fn.Attrs.AddClass("error")
	return fn
}

func linkNodeToEmbeddedReference(en *ast.EmbedNode) *ast.LinkNode {
	ref := en.Material.(*ast.ReferenceMaterialNode)
	ln := &ast.LinkNode{
		Ref:     ref.Ref,
		Inlines: ast.CreateInlineListNodeFromWords(ref.Ref.String()),
		OnlyRef: true,
	}
	return ln
}

func (e *evaluator) evaluateEmbeddedZettel(zettel domain.Zettel, syntax string) *ast.ZettelNode {
	zn := parser.ParseZettel(zettel, syntax, e.rtConfig)
	ast.Walk(e, zn.Ast)
	return zn
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

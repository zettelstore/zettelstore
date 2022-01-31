//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
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
	GetTagRef        func(string) *ast.Reference
	GetHostedRef     func(string) *ast.Reference
	GetFoundRef      func(zid id.Zid, fragment string) *ast.Reference
	GetImageMaterial func(zettel domain.Zettel, syntax string) ast.InlineEmbedNode
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
	if zn.Syntax == api.ValueSyntaxNone {
		// AST is empty, evaluate to a description list of metadata.
		zn.Ast = evaluateMetadata(zn.Meta)
		return
	}
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
		ctx:             ctx,
		port:            port,
		env:             env,
		rtConfig:        rtConfig,
		transcludeMax:   rtConfig.GetMaxTransclusions(),
		transcludeCount: 0,
		costMap:         map[id.Zid]transcludeCost{},
		embedMap:        map[string]*ast.InlineListNode{},
		marker:          &ast.ZettelNode{},
	}
	ast.Walk(&e, n)
}

type evaluator struct {
	ctx             context.Context
	port            Port
	env             *Environment
	rtConfig        config.Config
	transcludeMax   int
	transcludeCount int
	costMap         map[id.Zid]transcludeCost
	marker          *ast.ZettelNode
	embedMap        map[string]*ast.InlineListNode
}

type transcludeCost struct {
	zn *ast.ZettelNode
	ec int
}

func (e *evaluator) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		e.visitBlockList(n)
	case *ast.InlineListNode:
		e.visitInlineList(n)
	default:
		return e
	}
	return nil
}

func (e *evaluator) visitBlockList(bln *ast.BlockListNode) {
	for i := 0; i < len(bln.List); i++ {
		bn := bln.List[i]
		ast.Walk(e, bn)
		switch n := bn.(type) {
		case *ast.VerbatimNode:
			transcludeNode(bln, i, e.evalVerbatimNode(n))
		case *ast.TranscludeNode:
			transcludeNode(bln, i, e.evalTransclusionNode(n))
		}
	}
}

func transcludeNode(bln *ast.BlockListNode, i int, bn ast.BlockNode) {
	if ln, ok := bn.(*ast.BlockListNode); ok {
		bln.List = replaceWithBlockNodes(bln.List, i, ln.List)
		i += len(ln.List) - 1
	} else {
		bln.List[i] = bn
	}
}

func replaceWithBlockNodes(bns []ast.BlockNode, i int, replaceBns []ast.BlockNode) []ast.BlockNode {
	if len(replaceBns) == 1 {
		bns[i] = replaceBns[0]
		return bns
	}
	newIns := make([]ast.BlockNode, 0, len(bns)+len(replaceBns)-1)
	if i > 0 {
		newIns = append(newIns, bns[:i]...)
	}
	if len(replaceBns) > 0 {
		newIns = append(newIns, replaceBns...)
	}
	if i+1 < len(bns) {
		newIns = append(newIns, bns[i+1:]...)
	}
	return newIns
}

func (e *evaluator) evalVerbatimNode(vn *ast.VerbatimNode) ast.BlockNode {
	if vn.Kind != ast.VerbatimZettel {
		return vn
	}
	m := meta.New(id.Invalid)
	m.Set(api.KeySyntax, getSyntax(vn.Attrs, api.ValueSyntaxDraw))
	zettel := domain.Zettel{
		Meta:    m,
		Content: domain.NewContent(vn.Content),
	}
	e.transcludeCount++
	zn := e.evaluateEmbeddedZettel(zettel)
	return zn.Ast
}

func getSyntax(a *ast.Attributes, defSyntax string) string {
	if a != nil {
		if val, ok := a.Get(api.KeySyntax); ok {
			return val
		}
		if val, ok := a.Get(""); ok {
			return val
		}
	}
	return defSyntax
}

func (e *evaluator) evalTransclusionNode(tn *ast.TranscludeNode) ast.BlockNode {
	ref := tn.Ref

	// To prevent e.embedCount from counting
	if errText := e.checkMaxTransclusions(ref); errText != nil {
		return makeBlockNode(errText)
	}
	switch ref.State {
	case ast.RefStateInvalid, ast.RefStateBroken:
		e.transcludeCount++
		return makeBlockNode(createInlineErrorText(ref, "Invalid", "or", "broken", "transclusion", "reference:"))
	case ast.RefStateZettel, ast.RefStateFound:
	case ast.RefStateSelf:
		e.transcludeCount++
		return makeBlockNode(createInlineErrorText(ref, "Self", "transclusion", "reference:"))
	case ast.RefStateHosted, ast.RefStateBased, ast.RefStateExternal:
		return tn
	default:
		panic(fmt.Sprintf("Unknown state %v for reference %v", ref.State, ref))
	}

	zid, err := id.Parse(ref.URL.Path)
	if err != nil {
		panic(err)
	}

	cost, ok := e.costMap[zid]
	zn := cost.zn
	if zn == e.marker {
		e.transcludeCount++
		return makeBlockNode(createInlineErrorText(ref, "Recursive", "transclusion:"))
	}
	if !ok {
		zettel, err1 := e.port.GetZettel(box.NoEnrichContext(e.ctx), zid)
		if err1 != nil {
			e.transcludeCount++
			return makeBlockNode(createInlineErrorText(ref, "Unable", "to", "get", "zettel:"))
		}
		ec := e.transcludeCount
		e.costMap[zid] = transcludeCost{zn: e.marker, ec: ec}
		zn = e.evaluateEmbeddedZettel(zettel)
		e.costMap[zid] = transcludeCost{zn: zn, ec: e.transcludeCount - ec}
		e.transcludeCount = 0 // No stack needed, because embedding is done left-recursive, depth-first.
	}
	e.transcludeCount++
	if ec := cost.ec; ec > 0 {
		e.transcludeCount += cost.ec
	}
	return zn.Ast
}

func (e *evaluator) checkMaxTransclusions(ref *ast.Reference) ast.InlineNode {
	if maxTrans := e.transcludeMax; e.transcludeCount > maxTrans {
		e.transcludeCount = maxTrans + 1
		return createInlineErrorText(ref,
			"Too", "many", "transclusions", "(must", "be", "at", "most", strconv.Itoa(maxTrans)+",",
			"see", "runtime", "configuration", "key", "max-transclusions)")
	}
	return nil
}

func makeBlockNode(in ast.InlineNode) ast.BlockNode {
	return &ast.ParaNode{
		Inlines: &ast.InlineListNode{
			List: []ast.InlineNode{in},
		},
	}
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
		case *ast.EmbedRefNode:
			embedNode(iln, i, e.evalEmbedRefNode(n))
		case *ast.LiteralNode:
			embedNode(iln, i, e.evalLiteralNode(n))
		}
	}
}

func embedNode(iln *ast.InlineListNode, i int, in ast.InlineNode) {
	if ln, ok := in.(*ast.InlineListNode); ok {
		iln.List = replaceWithInlineNodes(iln.List, i, ln.List)
		i += len(ln.List) - 1
	} else {
		iln.List[i] = in
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

func (e *evaluator) evalEmbedRefNode(en *ast.EmbedRefNode) ast.InlineNode {
	ref := en.Ref

	// To prevent e.embedCount from counting
	if errText := e.checkMaxTransclusions(ref); errText != nil {
		return errText
	}

	switch ref.State {
	case ast.RefStateInvalid, ast.RefStateBroken:
		e.transcludeCount++
		return e.createInlineErrorImage(en)
	case ast.RefStateZettel, ast.RefStateFound:
	case ast.RefStateSelf:
		e.transcludeCount++
		return createInlineErrorText(ref, "Self", "embed", "reference:")
	case ast.RefStateHosted, ast.RefStateBased, ast.RefStateExternal:
		return en
	default:
		panic(fmt.Sprintf("Unknown state %v for reference %v", ref.State, ref))
	}

	zid, err := id.Parse(ref.URL.Path)
	if err != nil {
		panic(err)
	}
	zettel, err := e.port.GetZettel(box.NoEnrichContext(e.ctx), zid)
	if err != nil {
		e.transcludeCount++
		return e.createInlineErrorImage(en)
	}

	if syntax := e.getSyntax(zettel.Meta); parser.IsImageFormat(syntax) {
		return e.embedImage(en, zettel)
	} else if !parser.IsTextParser(syntax) {
		// Not embeddable.
		e.transcludeCount++
		return createInlineErrorText(ref, "Not", "embeddable (syntax="+syntax+"):")
	}

	cost, ok := e.costMap[zid]
	zn := cost.zn
	if zn == e.marker {
		e.transcludeCount++
		return createInlineErrorText(ref, "Recursive", "transclusion:")
	}
	if !ok {
		ec := e.transcludeCount
		e.costMap[zid] = transcludeCost{zn: e.marker, ec: ec}
		zn = e.evaluateEmbeddedZettel(zettel)
		e.costMap[zid] = transcludeCost{zn: zn, ec: e.transcludeCount - ec}
		e.transcludeCount = 0 // No stack needed, because embedding is done left-recursive, depth-first.
	}
	e.transcludeCount++

	result, ok := e.embedMap[ref.Value]
	if !ok {
		// Search for text to be embedded.
		result = findInlineList(zn.Ast, ref.URL.Fragment)
		e.embedMap[ref.Value] = result
	}
	if result.IsEmpty() {
		return &ast.LiteralNode{
			Kind: ast.LiteralComment,
			Text: "Nothing to transclude: " + ref.String(),
		}
	}

	if ec := cost.ec; ec > 0 {
		e.transcludeCount += cost.ec
	}
	return result
}

func (e *evaluator) evalLiteralNode(ln *ast.LiteralNode) ast.InlineNode {
	if ln.Kind != ast.LiteralZettel {
		return ln
	}
	m := meta.New(id.Invalid)
	m.Set(api.KeySyntax, getSyntax(ln.Attrs, api.ValueSyntaxDraw))
	zettel := domain.Zettel{
		Meta:    m,
		Content: domain.NewContent([]byte(ln.Text)),
	}
	e.transcludeCount++
	zn := e.evaluateEmbeddedZettel(zettel)
	result := findInlineList(zn.Ast, "")
	if result.IsEmpty() {
		return &ast.LiteralNode{
			Kind: ast.LiteralComment,
			Text: "Nothing to transclude",
		}
	}
	return result
}

func (e *evaluator) getSyntax(m *meta.Meta) string {
	if cfg := e.rtConfig; cfg != nil {
		return config.GetSyntax(m, cfg)
	}
	return m.GetDefault(api.KeySyntax, "")
}

func (e *evaluator) getTitle(m *meta.Meta) string {
	if cfg := e.rtConfig; cfg != nil {
		return config.GetTitle(m, cfg)
	}
	return m.GetDefault(api.KeyTitle, "")
}

func (e *evaluator) createInlineErrorImage(en *ast.EmbedRefNode) ast.InlineEmbedNode {
	errorZid := id.EmojiZid
	if gim := e.env.GetImageMaterial; gim != nil {
		zettel, err := e.port.GetZettel(box.NoEnrichContext(e.ctx), errorZid)
		if err != nil {
			panic(err)
		}
		inlines := en.Inlines
		if inlines == nil {
			if title := e.getTitle(zettel.Meta); title != "" {
				inlines = parser.ParseMetadata(title)
			}
		}
		result := gim(zettel, e.getSyntax(zettel.Meta))
		switch er := result.(type) {
		case *ast.EmbedRefNode:
			er.Inlines = inlines
			er.Attrs = en.Attrs
		case *ast.EmbedBLOBNode:
			er.Inlines = inlines
			er.Attrs = en.Attrs
		}
		return result
	}
	en.Ref = ast.ParseReference(errorZid.String())
	if en.Inlines == nil {
		en.Inlines = parser.ParseMetadata("Error placeholder")
	}
	return en
}

func (e *evaluator) embedImage(en *ast.EmbedRefNode, zettel domain.Zettel) ast.InlineEmbedNode {
	if gim := e.env.GetImageMaterial; gim != nil {
		result := gim(zettel, e.getSyntax(zettel.Meta))
		switch er := result.(type) {
		case *ast.EmbedRefNode:
			er.Inlines = en.Inlines
			er.Attrs = en.Attrs
		case *ast.EmbedBLOBNode:
			er.Inlines = en.Inlines
			er.Attrs = en.Attrs
		}
		return result
	}
	return en
}

func createInlineErrorText(ref *ast.Reference, msgWords ...string) ast.InlineNode {
	text := ast.CreateInlineListNodeFromWords(msgWords...)
	if ref != nil {
		ln := linkNodeToReference(ref)
		text.Append(&ast.TextNode{Text: ":"}, &ast.SpaceNode{Lexeme: " "}, ln, &ast.TextNode{Text: "."}, &ast.SpaceNode{Lexeme: " "})
	}
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

func linkNodeToReference(ref *ast.Reference) *ast.LinkNode {
	ln := &ast.LinkNode{
		Ref:     ref,
		Inlines: ast.CreateInlineListNodeFromWords(ref.String()),
		OnlyRef: true,
	}
	return ln
}

func (e *evaluator) evaluateEmbeddedZettel(zettel domain.Zettel) *ast.ZettelNode {
	zn := parser.ParseZettel(zettel, e.getSyntax(zettel.Meta), e.rtConfig)
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

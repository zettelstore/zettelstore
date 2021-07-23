//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package adapter provides handlers for web requests.
package adapter

import (
	"context"
	"errors"
	"strings"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/server"
)

// ErrNoSuchEncoding signals an unsupported encoding encoding
var ErrNoSuchEncoding = errors.New("no such encoding")

// EncodeInlines returns a string representation of the inline slice.
func EncodeInlines(is ast.InlineSlice, enc api.EncodingEnum, env *encoder.Environment) (string, error) {
	encdr := encoder.Create(enc, env)
	if encdr == nil {
		return "", ErrNoSuchEncoding
	}

	var content strings.Builder
	_, err := encdr.WriteInlines(&content, is)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

// MakeLinkAdapter creates an adapter to change a link node during encoding.
func MakeLinkAdapter(
	ctx context.Context,
	b server.Builder,
	key byte,
	getMeta usecase.GetMeta,
	part string,
	enc api.EncodingEnum,
) func(*ast.LinkNode) ast.InlineNode {
	return func(origLink *ast.LinkNode) ast.InlineNode {
		origRef := origLink.Ref
		if origRef == nil {
			return origLink
		}
		if origRef.State == ast.RefStateBased {
			newLink := *origLink
			urlPrefix := b.GetURLPrefix()
			newRef := ast.ParseReference(urlPrefix + origRef.Value[1:])
			newRef.State = ast.RefStateHosted
			newLink.Ref = newRef
			return &newLink
		}
		if origRef.State != ast.RefStateZettel {
			return origLink
		}
		zid, err := id.Parse(origRef.URL.Path)
		if err != nil {
			panic(err)
		}
		_, err = getMeta.Run(box.NoEnrichContext(ctx), zid)
		if errors.Is(err, &box.ErrNotAllowed{}) {
			return &ast.FormatNode{
				Kind:    ast.FormatSpan,
				Attrs:   origLink.Attrs,
				Inlines: origLink.Inlines,
			}
		}
		var newRef *ast.Reference
		if err == nil {
			ub := b.NewURLBuilder(key).SetZid(zid)
			if part != "" {
				ub.AppendQuery(api.QueryKeyPart, part)
			}
			if enc != api.EncoderUnknown {
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
		newLink := *origLink
		newLink.Ref = newRef
		return &newLink
	}
}

// MakeImageAdapter creates an adapter to change an image node during encoding.
func MakeImageAdapter(ctx context.Context, b server.Builder, getMeta usecase.GetMeta) func(*ast.ImageNode) ast.InlineNode {
	return func(origImage *ast.ImageNode) ast.InlineNode {
		if origImage.Ref == nil {
			return origImage
		}
		switch origImage.Ref.State {
		case ast.RefStateInvalid:
			return createZettelImage(b, origImage, id.EmojiZid, ast.RefStateInvalid)
		case ast.RefStateZettel:
			zid, err := id.Parse(origImage.Ref.Value)
			if err != nil {
				panic(err)
			}
			_, err = getMeta.Run(box.NoEnrichContext(ctx), zid)
			if err != nil {
				return createZettelImage(b, origImage, id.EmojiZid, ast.RefStateBroken)
			}
			return createZettelImage(b, origImage, zid, ast.RefStateFound)
		}
		return origImage
	}
}

func createZettelImage(b server.Builder, origImage *ast.ImageNode, zid id.Zid, state ast.RefState) *ast.ImageNode {
	newImage := *origImage
	newImage.Ref = ast.ParseReference(
		b.NewURLBuilder('z').SetZid(zid).AppendQuery(api.QueryKeyPart, api.PartContent).AppendQuery(
			api.QueryKeyEncoding, api.EncodingRaw).String())
	newImage.Ref.State = state
	return &newImage
}

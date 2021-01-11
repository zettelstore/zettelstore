//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
)

// ErrNoSuchFormat signals an unsupported encoding format
var ErrNoSuchFormat = errors.New("no such format")

// FormatInlines returns a string representation of the inline slice.
func FormatInlines(
	is ast.InlineSlice, format string, options ...encoder.Option) (string, error) {
	enc := encoder.Create(format, options...)
	if enc == nil {
		return "", ErrNoSuchFormat
	}

	var content strings.Builder
	_, err := enc.WriteInlines(&content, is)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

// MakeLinkAdapter creates an adapter to change a link node during encoding.
func MakeLinkAdapter(
	ctx context.Context,
	key byte,
	getMeta usecase.GetMeta,
	part, format string,
) func(*ast.LinkNode) ast.InlineNode {
	return func(origLink *ast.LinkNode) ast.InlineNode {
		origRef := origLink.Ref
		if origRef == nil || origRef.State != ast.RefStateZettel {
			return origLink
		}
		zid, err := id.Parse(origRef.URL.Path)
		if err != nil {
			panic(err)
		}
		_, err = getMeta.Run(ctx, zid)
		newLink := *origLink
		if err == nil {
			u := NewURLBuilder(key).SetZid(zid)
			if part != "" {
				u.AppendQuery("_part", part)
			}
			if format != "" {
				u.AppendQuery("_format", format)
			}
			if fragment := origRef.URL.EscapedFragment(); len(fragment) > 0 {
				u.SetFragment(fragment)
			}
			newRef := ast.ParseReference(u.String())
			newRef.State = ast.RefStateZettelFound
			newLink.Ref = newRef
			return &newLink
		}
		if place.IsErrNotAllowed(err) {
			return &ast.FormatNode{
				Code:    ast.FormatSpan,
				Attrs:   origLink.Attrs,
				Inlines: origLink.Inlines,
			}
		}
		newRef := ast.ParseReference(origRef.Value)
		newRef.State = ast.RefStateZettelBroken
		newLink.Ref = newRef
		return &newLink
	}
}

// MakeImageAdapter creates an adapter to change an image node during encoding.
func MakeImageAdapter() func(*ast.ImageNode) ast.InlineNode {
	return func(origImage *ast.ImageNode) ast.InlineNode {
		if origImage.Ref == nil || origImage.Ref.State != ast.RefStateZettel {
			return origImage
		}
		newImage := *origImage
		zid, err := id.Parse(newImage.Ref.Value)
		if err != nil {
			panic(err)
		}
		newImage.Ref = ast.ParseReference(
			NewURLBuilder('z').SetZid(zid).AppendQuery("_part", "content").AppendQuery(
				"_format", "raw").String())
		newImage.Ref.State = ast.RefStateZettelFound
		return &newImage
	}
}

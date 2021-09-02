//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package encoder provides a generic interface to encode the abstract syntax
// tree into some text form.
package encoder

import (
	"errors"
	"fmt"
	"io"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
)

// Encoder is an interface that allows to encode different parts of a zettel.
type Encoder interface {
	WriteZettel(io.Writer, *ast.ZettelNode, EvalMetaFunc) (int, error)
	WriteMeta(io.Writer, *meta.Meta, EvalMetaFunc) (int, error)
	WriteContent(io.Writer, *ast.ZettelNode) (int, error)
	WriteBlocks(io.Writer, *ast.BlockListNode) (int, error)
	WriteInlines(io.Writer, *ast.InlineListNode) (int, error)
}

// EvalMetaFunc is a function that takes a string of metadata and returns
// a list of syntax elements.
type EvalMetaFunc func(string) *ast.InlineListNode

// Some errors to signal when encoder methods are not implemented.
var (
	ErrNoWriteZettel  = errors.New("method WriteZettel is not implemented")
	ErrNoWriteMeta    = errors.New("method WriteMeta is not implemented")
	ErrNoWriteContent = errors.New("method WriteContent is not implemented")
	ErrNoWriteBlocks  = errors.New("method WriteBlocks is not implemented")
	ErrNoWriteInlines = errors.New("method WriteInlines is not implemented")
)

// Create builds a new encoder with the given options.
func Create(enc api.EncodingEnum, env *Environment) Encoder {
	if info, ok := registry[enc]; ok {
		return info.Create(env)
	}
	return nil
}

// Info stores some data about an encoder.
type Info struct {
	Create  func(*Environment) Encoder
	Default bool
}

var registry = map[api.EncodingEnum]Info{}
var defEncoding api.EncodingEnum

// Register the encoder for later retrieval.
func Register(enc api.EncodingEnum, info Info) {
	if _, ok := registry[enc]; ok {
		panic(fmt.Sprintf("Encoder %q already registered", enc))
	}
	if info.Default {
		if defEncoding != api.EncoderUnknown && defEncoding != enc {
			panic(fmt.Sprintf("Default encoder already set: %q, new encoding: %q", defEncoding, enc))
		}
		defEncoding = enc
	}
	registry[enc] = info
}

// GetEncodings returns all registered encodings, ordered by encoding value.
func GetEncodings() []api.EncodingEnum {
	result := make([]api.EncodingEnum, 0, len(registry))
	for enc := range registry {
		result = append(result, enc)
	}
	return result
}

// GetDefaultEncoding returns the encoding that should be used as default.
func GetDefaultEncoding() api.EncodingEnum {
	if defEncoding != api.EncoderUnknown {
		return defEncoding
	}
	if _, ok := registry[api.EncoderDJSON]; ok {
		return api.EncoderDJSON
	}
	panic("No default encoding given")
}

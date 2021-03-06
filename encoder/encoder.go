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
	"io"
	"log"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
)

// Encoder is an interface that allows to encode different parts of a zettel.
type Encoder interface {
	WriteZettel(io.Writer, *ast.ZettelNode, bool) (int, error)
	WriteMeta(io.Writer, *meta.Meta) (int, error)
	WriteContent(io.Writer, *ast.ZettelNode) (int, error)
	WriteBlocks(io.Writer, ast.BlockSlice) (int, error)
	WriteInlines(io.Writer, ast.InlineSlice) (int, error)
}

// Some errors to signal when encoder methods are not implemented.
var (
	ErrNoWriteZettel  = errors.New("method WriteZettel is not implemented")
	ErrNoWriteMeta    = errors.New("method WriteMeta is not implemented")
	ErrNoWriteContent = errors.New("method WriteContent is not implemented")
	ErrNoWriteBlocks  = errors.New("method WriteBlocks is not implemented")
	ErrNoWriteInlines = errors.New("method WriteInlines is not implemented")
)

// Create builds a new encoder with the given options.
func Create(format api.EncodingEnum, env *Environment) Encoder {
	if info, ok := registry[format]; ok {
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
var defFormat api.EncodingEnum

// Register the encoder for later retrieval.
func Register(format api.EncodingEnum, info Info) {
	if _, ok := registry[format]; ok {
		log.Fatalf("Writer with format %q already registered", format)
	}
	if info.Default {
		if defFormat != api.EncoderUnknown && defFormat != format {
			log.Fatalf("Default format already set: %q, new format: %q", defFormat, format)
		}
		defFormat = format
	}
	registry[format] = info
}

// GetFormats returns all registered formats, ordered by format code.
func GetFormats() []api.EncodingEnum {
	result := make([]api.EncodingEnum, 0, len(registry))
	for format := range registry {
		result = append(result, format)
	}
	return result
}

// GetDefaultFormat returns the format that should be used as default.
func GetDefaultFormat() api.EncodingEnum {
	if defFormat != api.EncoderUnknown {
		return defFormat
	}
	if _, ok := registry[api.EncoderJSON]; ok {
		return api.EncoderJSON
	}
	log.Fatalf("No default format given")
	return api.EncoderUnknown
}

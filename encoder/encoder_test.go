//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package encoder_test

import (
	"fmt"
	"strings"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"

	_ "zettelstore.de/z/encoder/djsonenc"  // Allow to use DJSON encoder.
	_ "zettelstore.de/z/encoder/htmlenc"   // Allow to use HTML encoder.
	_ "zettelstore.de/z/encoder/nativeenc" // Allow to use native encoder.
	_ "zettelstore.de/z/encoder/textenc"   // Allow to use text encoder.
	_ "zettelstore.de/z/encoder/zmkenc"    // Allow to use zmk encoder.
	_ "zettelstore.de/z/parser/zettelmark" // Allow to use zettelmark parser.
)

type zmkTestCase struct {
	descr  string
	zmk    string
	inline bool
	expect expectMap
}

type expectMap map[api.EncodingEnum]string

const useZmk = "\000"

const (
	encoderDJSON  = api.EncoderDJSON
	encoderHTML   = api.EncoderHTML
	encoderNative = api.EncoderNative
	encoderText   = api.EncoderText
	encoderZmk    = api.EncoderZmk
)

func TestEncoder(t *testing.T) {
	for i := range tcsInline {
		tcsInline[i].inline = true
	}
	executeTestCases(t, append(tcsBlock, tcsInline...))
}

func executeTestCases(t *testing.T, testCases []zmkTestCase) {
	t.Helper()
	for testNum, tc := range testCases {
		inp := input.NewInput(tc.zmk)
		var pe parserEncoder
		if tc.inline {
			pe = &peInlines{iln: parser.ParseInlines(inp, api.ValueSyntaxZmk)}
		} else {
			pe = &peBlocks{bln: parser.ParseBlocks(inp, nil, api.ValueSyntaxZmk)}
		}
		checkEncodings(t, testNum, pe, tc.descr, tc.expect, tc.zmk)
	}
}

func checkEncodings(t *testing.T, testNum int, pe parserEncoder, descr string, expected expectMap, zmkDefault string) {
	t.Helper()
	for enc, exp := range expected {
		encdr := encoder.Create(enc, nil)
		got, err := pe.encode(encdr)
		if err != nil {
			t.Error(err)
			continue
		}
		if enc == api.EncoderZmk && exp == "\000" {
			exp = zmkDefault
		}
		if got != exp {
			prefix := fmt.Sprintf("Test #%d", testNum)
			if d := descr; d != "" {
				prefix += "\nReason:   " + d
			}
			prefix += "\nMode:     " + pe.mode()
			t.Errorf("%s\nEncoder:  %s\nExpected: %q\nGot:      %q", prefix, enc, exp, got)
		}
	}
}

type parserEncoder interface {
	encode(encoder.Encoder) (string, error)
	mode() string
}

type peInlines struct {
	iln *ast.InlineListNode
}

func (in peInlines) encode(encdr encoder.Encoder) (string, error) {
	var sb strings.Builder
	if _, err := encdr.WriteInlines(&sb, in.iln); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (peInlines) mode() string { return "inline" }

type peBlocks struct {
	bln *ast.BlockListNode
}

func (bl peBlocks) encode(encdr encoder.Encoder) (string, error) {
	var sb strings.Builder
	if _, err := encdr.WriteBlocks(&sb, bl.bln); err != nil {
		return "", err
	}
	return sb.String(), nil

}
func (peBlocks) mode() string { return "block" }

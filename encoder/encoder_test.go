//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package encoder_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/input"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"

	_ "zettelstore.de/z/encoder/htmlenc"  // Allow to use HTML encoder.
	_ "zettelstore.de/z/encoder/sexprenc" // Allow to use sexpr encoder.
	_ "zettelstore.de/z/encoder/textenc"  // Allow to use text encoder.
	_ "zettelstore.de/z/encoder/zjsonenc" // Allow to use ZJSON encoder.
	_ "zettelstore.de/z/encoder/zmkenc"   // Allow to use zmk encoder.
	"zettelstore.de/z/parser/cleaner"
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
	encoderZJSON = api.EncoderZJSON
	encoderHTML  = api.EncoderHTML
	encoderSexpr = api.EncoderSexpr
	encoderText  = api.EncoderText
	encoderZmk   = api.EncoderZmk
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
		inp := input.NewInput([]byte(tc.zmk))
		var pe parserEncoder
		if tc.inline {
			is := parser.ParseInlines(inp, api.ValueSyntaxZmk)
			cleaner.CleanInlineSlice(&is)
			pe = &peInlines{is: is}
		} else {
			pe = &peBlocks{bs: parser.ParseBlocks(inp, nil, api.ValueSyntaxZmk)}
		}
		checkEncodings(t, testNum, pe, tc.descr, tc.expect, tc.zmk)
		checkSexpr(t, testNum, pe, tc.descr)
	}
}

func checkEncodings(t *testing.T, testNum int, pe parserEncoder, descr string, expected expectMap, zmkDefault string) {
	t.Helper()
	for enc, exp := range expected {
		encdr := encoder.Create(enc)
		got, err := pe.encode(encdr)
		if err != nil {
			t.Error(err)
			continue
		}
		if enc == api.EncoderZmk && exp == useZmk {
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

func checkSexpr(t *testing.T, testNum int, pe parserEncoder, descr string) {
	t.Helper()
	encdr := encoder.Create(encoderSexpr)
	exp, err := pe.encode(encdr)
	if err != nil {
		t.Error(err)
		return
	}
	val, err := sxpf.ReadString(exp)
	if err != nil {
		t.Error(err)
		return
	}
	got := val.String()
	if exp != got {
		prefix := fmt.Sprintf("Test #%d", testNum)
		if d := descr; d != "" {
			prefix += "\nReason:   " + d
		}
		prefix += "\nMode:     " + pe.mode()
		t.Errorf("%s\n\nExpected: %q\nGot:      %q", prefix, exp, got)
	}
}

type parserEncoder interface {
	encode(encoder.Encoder) (string, error)
	mode() string
}

type peInlines struct {
	is ast.InlineSlice
}

func (in peInlines) encode(encdr encoder.Encoder) (string, error) {
	var buf bytes.Buffer
	if _, err := encdr.WriteInlines(&buf, &in.is); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (peInlines) mode() string { return "inline" }

type peBlocks struct {
	bs ast.BlockSlice
}

func (bl peBlocks) encode(encdr encoder.Encoder) (string, error) {
	var buf bytes.Buffer
	if _, err := encdr.WriteBlocks(&buf, &bl.bs); err != nil {
		return "", err
	}
	return buf.String(), nil

}
func (peBlocks) mode() string { return "block" }

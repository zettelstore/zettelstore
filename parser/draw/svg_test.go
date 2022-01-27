//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// This file was originally created by the ASCIIToSVG contributors under an MIT
// license, but later changed to fulfil the needs of Zettelstore. The following
// statements affects the original code as found on
// https://github.com/asciitosvg/asciitosvg (Commit:
// ca82a5ce41e2190a05e07af6e8b3ea4e3256a283, 2020-11-20):
//
// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.
//-----------------------------------------------------------------------------

package draw

import (
	"strings"
	"testing"
)

func TestCanvasToSVG(t *testing.T) {
	t.Parallel()
	data := []struct {
		input  []string
		length int
	}{
		// 0 Box with dashed corners and text
		{
			[]string{
				"+--.",
				"|Hi:",
				"+--+",
			},
			494,
		},

		// 1 Box with non-existent ref
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
			},
			580,
		},

		// 2 Box with ref, change background color of container with #RRGGBB
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				"[a]: {\"fill\":\"#000000\"}",
			},
			675,
		},

		// 3 Box with ref && fill, change label
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				"[a]: {\"fill\":\"#000000\",\"a2s:label\":\"abcdefg\"}",
			},
			639,
		},

		// 4 Box with ref && fill && label, remove ref
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				"[a]: {\"fill\":\"#000000\",\"a2s:label\":\"abcd\",\"a2s:delref\":1}",
			},
			581,
		},

		// 5 Ticks and dots in lines.
		{
			[]string{
				" ------x----->",
				"",
				" <-----*------",
			},
			1072,
		},

		// 6 Just text
		{
			[]string{
				" foo",
			},
			261,
		},

		// 7 Just text with a deleting reference
		{
			[]string{
				" foo",
				"[1,0]: {\"a2s:delref\":1,\"a2s:label\":\"foo\"}",
			},
			262,
		},

		// 8 Just text with a link
		{
			[]string{
				" foo",
				"[1,0]: {\"a2s:delref\":1, \"a2s:link\":\"https://github.com/asciitosvg/asciitosvg\"}",
			},
			306,
		},
	}
	for i, line := range data {
		canvas, err := newCanvas([]byte(strings.Join(line.input, "\n")), 9)
		if err != nil {
			t.Fatalf("Error creating canvas: %s", err)
		}
		actual := string(CanvasToSVG(canvas, "", 9, 16))
		// TODO(dhobsd): Use golden file? Worth postponing once output is actually
		// nice.
		if line.length != len(actual) {
			t.Errorf("%d: expected length %d, but got %d\n%q", i, line.length, len(actual), actual)
		}
	}
}

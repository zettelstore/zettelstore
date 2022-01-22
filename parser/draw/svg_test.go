// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

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
			1545,
		},

		// 1 Box with non-existent ref
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
			},
			1631,
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
			1726,
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
			1694,
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
			1632,
		},

		// 5 Ticks and dots in lines.
		{
			[]string{
				" ------x----->",
				"",
				" <-----o------",
			},
			1836,
		},

		// 6 Just text
		{
			[]string{
				" foo",
			},
			1344,
		},

		// 7 Just text with a deleting reference
		{
			[]string{
				" foo",
				"[1,0]: {\"a2s:delref\":1,\"a2s:label\":\"foo\"}",
			},
			1345,
		},

		// 8 Just text with a link
		{
			[]string{
				" foo",
				"[1,0]: {\"a2s:delref\":1, \"a2s:link\":\"https://github.com/asciitosvg/asciitosvg\"}",
			},
			1389,
		},
	}
	for i, line := range data {
		canvas, err := NewCanvas([]byte(strings.Join(line.input, "\n")), 9, false)
		if err != nil {
			t.Fatalf("Error creating canvas: %s", err)
		}
		actual := string(CanvasToSVG(canvas, false, "", 9, 16))
		// TODO(dhobsd): Use golden file? Worth postponing once output is actually
		// nice.
		if line.length != len(actual) {
			t.Errorf("%d: expected length %d, but got %d", i, line.length, len(actual))
		}
	}
}

// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package draw

import (
	"testing"
)

func TestParseHexColor(t *testing.T) {
	t.Parallel()
	data := []struct {
		color   string
		rgb     []int
		isError bool
	}{
		{"#123", []int{17, 34, 51}, false},
		{"#fff", []int{255, 255, 255}, false},
		{"#FFF", []int{255, 255, 255}, false},
		{"#ffffff", []int{255, 255, 255}, false},
		{"#FFFFFF", []int{255, 255, 255}, false},
		{"#fFfFFf", []int{255, 255, 255}, false},
		{"#notacolor", nil, true},
		{"alsonotacolor", nil, true},
		{"#ffg", nil, true},
		{"#FFG", nil, true},
		{"#fffffg", nil, true},
		{"#FFFFFG", nil, true},
	}

	for i, v := range data {
		r, g, b, err := colorToRGB(v.color)
		if v.isError {
			if err == nil {
				t.Errorf("%d: colorToRGB(%q) expected error, but got none", i, v.color)
			}
			continue
		}

		if err != nil {
			t.Errorf("%d: colorToRGB(%q) got error %v", i, v.color, err)
			continue
		}

		if r != v.rgb[0] || g != v.rgb[1] || b != v.rgb[2] {
			t.Errorf("%d: colorToRGB(%q) expected %v, but got [%v,%v,%v]", i, v.color, v.rgb, r, g, b)
		}
	}
}

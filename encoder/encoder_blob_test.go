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
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"

	_ "zettelstore.de/z/parser/blob" // Allow to use BLOB parser.
)

type blobTestCase struct {
	descr  string
	blob   []byte
	expect expectMap
}

var pngTestCases = []blobTestCase{
	{
		descr: "Minimal PNG",
		blob: []byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x00, 0x00, 0x00, 0x00, 0x3a, 0x7e, 0x9b,
			0x55, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x62, 0x00, 0x00, 0x00,
			0x06, 0x00, 0x03, 0x36, 0x37, 0x7c, 0xa8, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
			0x42, 0x60, 0x82,
		},
		expect: expectMap{
			encoderZJSON:  `[{"":"Blob","q":"PNG","s":"png","o":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGNiAAAABgADNjd8qAAAAABJRU5ErkJggg=="}]`,
			encoderHTML:   `<img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGNiAAAABgADNjd8qAAAAABJRU5ErkJggg==" title="PNG">`,
			encoderNative: `[BLOB "PNG" "png" "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGNiAAAABgADNjd8qAAAAABJRU5ErkJggg=="]`,
			encoderText:   "",
			encoderZmk:    `%% Unable to display BLOB with title 'PNG' and syntax 'png'.`,
		},
	},
}

func TestBlob(t *testing.T) {
	m := meta.New(id.Invalid)
	m.Set(api.KeyTitle, "PNG")
	for testNum, tc := range pngTestCases {
		inp := input.NewInput(tc.blob)
		pe := &peBlocks{bs: parser.ParseBlocks(inp, m, "png")}
		checkEncodings(t, testNum, pe, tc.descr, tc.expect, "???")
	}
}

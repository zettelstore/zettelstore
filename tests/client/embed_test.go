//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore-client.
//
// Zettelstore client is licensed under the latest version of the EUPL
// (European Union Public License). Please see file LICENSE.txt for your rights
// and obligations under this license.
//-----------------------------------------------------------------------------

package client_test

import (
	"context"
	"strings"
	"testing"

	"zettelstore.de/c/api"
)

func TestEmbeddedZettel(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")

	const (
		abcZid      = api.ZettelID("20211020121000")
		abc10000Zid = api.ZettelID("20211020121400")
	)
	contentMap := map[api.ZettelID]int{
		abcZid:                         1,
		api.ZettelID("20211020121100"): 10,
		api.ZettelID("20211020121145"): 100,
		api.ZettelID("20211020121300"): 1000,
	}
	zettelData, err := c.GetZettelJSON(context.Background(), abcZid)
	if err != nil {
		t.Error(err)
		return
	}
	if encoding := zettelData.Encoding; encoding != "" {
		t.Errorf("No encoding expected for zettel %q, but got %q", abcZid, encoding)
	}
	baseContent := zettelData.Content
	for zid, siz := range contentMap {
		content, err := c.GetEvaluatedZettel(context.Background(), zid, api.EncoderHTML)
		if err != nil {
			t.Error(err)
			continue
		}
		prefix := "<p>"
		if !strings.HasPrefix(content, prefix) {
			t.Errorf("Content of zettel %q does not start with %q: %q", zid, prefix, stringHead(content))
			continue
		}
		suffix := "</p>"
		if !strings.HasSuffix(content, suffix) {
			t.Errorf("Content of zettel %q does not end with %q: %q", zid, suffix, stringTail(content))
			continue
		}
		got := content[len(prefix) : len(content)-len(suffix)]
		if expect := strings.Repeat(baseContent, siz); expect != got {
			t.Errorf("Unexpected content for zettel %q\nExpect: %q\nGot:    %q", zid, expect, got)
		}
	}
}

func stringHead(s string) string {
	const maxLen = 40
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func stringTail(s string) string {
	const maxLen = 40
	if len(s) <= maxLen {
		return s
	}
	return "..." + s[len(s)-maxLen-3:]
}

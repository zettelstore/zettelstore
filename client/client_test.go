//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package client provides a client for accessing the Zettelstore via its API.
package client_test

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"testing"

	"zettelstore.de/z/api"
	"zettelstore.de/z/client"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func TestList(t *testing.T) {
	testdata := []struct {
		user string
		exp  int
	}{
		{"", 7},
		{"creator", 8},
		{"reader", 12},
		{"writer", 12},
		{"owner", 34},
	}

	t.Parallel()
	c := getClient()
	query := url.Values{api.QueryKeyFormat: {"html"}} // Client must remove "html"
	for i, tc := range testdata {
		t.Run(fmt.Sprintf("User %d/%q", i, tc.user), func(tt *testing.T) {
			c.SetAuth(tc.user, tc.user)
			l, err := c.ListZettel(context.Background(), query)
			if err != nil {
				tt.Error(err)
				return
			}
			got := len(l)
			if got != tc.exp {
				tt.Errorf("List of length %d expected, but got %d\n%v", tc.exp, got, l)
			}
		})
	}
	l, err := c.ListZettel(context.Background(), url.Values{meta.KeyRole: {meta.ValueRoleConfiguration}})
	if err != nil {
		t.Error(err)
		return
	}
	got := len(l)
	if got != 27 {
		t.Errorf("List of length %d expected, but got %d\n%v", 27, got, l)
	}
}
func TestGetZettel(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	z, err := c.GetZettelJSON(context.Background(), id.DefaultHomeZid, url.Values{api.QueryKeyPart: {api.PartContent}})
	if err != nil {
		t.Error(err)
		return
	}
	if got := z.ID; got != id.DefaultHomeZid.String() {
		t.Errorf("Expected Zid %s, got %s", id.DefaultHomeZid, got)
	}
	if m := z.Meta; len(m) > 0 {
		t.Errorf("Exptected empty meta, but got %v", z.Meta)
	}
	if z.Content == "" || z.Encoding != "" {
		t.Errorf("Expect non-empty content, but empty encoding (got %q)", z.Encoding)
	}
}

func TestListTags(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	tm, err := c.ListTags(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	tags := []struct {
		key  string
		size int
	}{
		{"#invisible", 1},
		{"#user", 4},
		{"#test", 4},
	}
	if len(tm) != len(tags) {
		t.Errorf("Expected %d different tags, but got only %d (%v)", len(tags), len(tm), tm)
	}
	for _, tag := range tags {
		if zl, ok := tm[tag.key]; !ok {
			t.Errorf("No tag %v: %v", tag.key, tm)
		} else if len(zl) != tag.size {
			t.Errorf("Expected %d zettel with tag %v, but got %v", tag.size, tag.key, zl)
		}
	}
	for i, id := range tm["#user"] {
		if id != tm["#test"][i] {
			t.Errorf("Tags #user and #test have different content: %v vs %v", tm["#user"], tm["#test"])
		}
	}
}

func TestListRoles(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	rl, err := c.ListRoles(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	exp := []string{"configuration", "user", "zettel"}
	if len(rl) != len(exp) {
		t.Errorf("Expected %d different tags, but got only %d (%v)", len(exp), len(rl), rl)
	}
	for i, id := range exp {
		if id != rl[i] {
			t.Errorf("Role list pos %d: expected %q, got %q", i, id, rl[i])
		}
	}
}

var baseURL string

func init() {
	flag.StringVar(&baseURL, "base-url", "", "Base URL")
}

func getClient() *client.Client { return client.NewClient(baseURL) }

// TestMain controls whether client API tests should run or not.
func TestMain(m *testing.M) {
	flag.Parse()
	if baseURL != "" {
		m.Run()
	}
}

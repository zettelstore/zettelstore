//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore-client.
//
// Zettelstore client is licensed under the latest version of the EUPL
// (European Union Public License). Please see file LICENSE.txt for your rights
// and obligations under this license.
//-----------------------------------------------------------------------------

// Package client provides a client for accessing the Zettelstore via its API.
package client_test

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/c/client"
)

func nextZid(zid api.ZettelID) api.ZettelID {
	numVal, err := strconv.Atoi(string(zid))
	if err != nil {
		panic(err)
	}
	return api.ZettelID(fmt.Sprintf("%014d", numVal+1))
}

func TestNextZid(t *testing.T) {
	testCases := []struct {
		zid, exp api.ZettelID
	}{
		{api.ZettelID("00000000000000"), api.ZettelID("00000000000001")},
	}
	for i, tc := range testCases {
		if got := nextZid(tc.zid); got != tc.exp {
			t.Errorf("%d: zid=%q, exp=%q, got=%q", i, tc.zid, tc.exp, got)
		}

	}
}
func TestCreateGetRenameDeleteZettel(t *testing.T) {
	// Is not to be allowed to run in parallel with other tests.
	zettel := `title: A Test

Example content.`
	c := getClient()
	c.SetAuth("owner", "owner")
	zid, err := c.CreateZettel(context.Background(), zettel)
	if err != nil {
		t.Error("Cannot create zettel:", err)
		return
	}
	if !zid.IsValid() {
		t.Error("Invalid zettel ID", zid)
		return
	}
	data, err := c.GetZettel(context.Background(), zid, api.PartZettel)
	if err != nil {
		t.Error("Cannot read zettel", zid, err)
		return
	}
	exp := `title: A Test
role: zettel
syntax: zmk

Example content.`
	if data != exp {
		t.Errorf("Expected zettel data: %q, but got %q", exp, data)
	}
	newZid := nextZid(zid)
	err = c.RenameZettel(context.Background(), zid, newZid)
	if err != nil {
		t.Error("Cannot rename", zid, ":", err)
		newZid = zid
	}
	err = c.DeleteZettel(context.Background(), newZid)
	if err != nil {
		t.Error("Cannot delete", zid, ":", err)
		return
	}
}

func TestCreateRenameDeleteZettelJSON(t *testing.T) {
	// Is not to be allowed to run in parallel with other tests.
	c := getClient()
	c.SetAuth("creator", "creator")
	zid, err := c.CreateZettelJSON(context.Background(), &api.ZettelDataJSON{
		Meta:     nil,
		Encoding: "",
		Content:  "Example",
	})
	if err != nil {
		t.Error("Cannot create zettel:", err)
		return
	}
	if !zid.IsValid() {
		t.Error("Invalid zettel ID", zid)
		return
	}
	newZid := nextZid(zid)
	c.SetAuth("owner", "owner")
	err = c.RenameZettel(context.Background(), zid, newZid)
	if err != nil {
		t.Error("Cannot rename", zid, ":", err)
		newZid = zid
	}
	err = c.DeleteZettel(context.Background(), newZid)
	if err != nil {
		t.Error("Cannot delete", zid, ":", err)
		return
	}
}

func TestUpdateZettel(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	z, err := c.GetZettel(context.Background(), api.ZidDefaultHome, api.PartZettel)
	if err != nil {
		t.Error(err)
		return
	}
	if !strings.HasPrefix(z, "title: Home\n") {
		t.Error("Got unexpected zettel", z)
		return
	}
	newZettel := `title: New Home
role: zettel
syntax: zmk

Empty`
	err = c.UpdateZettel(context.Background(), api.ZidDefaultHome, newZettel)
	if err != nil {
		t.Error(err)
		return
	}
	zt, err := c.GetZettel(context.Background(), api.ZidDefaultHome, api.PartZettel)
	if err != nil {
		t.Error(err)
		return
	}
	if zt != newZettel {
		t.Errorf("Expected zettel %q, got %q", newZettel, zt)
	}
	// Must delete to clean up for next tests
	err = c.DeleteZettel(context.Background(), api.ZidDefaultHome)
	if err != nil {
		t.Error("Cannot delete", api.ZidDefaultHome, ":", err)
		return
	}
}

func TestUpdateZettelJSON(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("writer", "writer")
	z, err := c.GetZettelJSON(context.Background(), api.ZidDefaultHome)
	if err != nil {
		t.Error(err)
		return
	}
	if got := z.Meta[api.KeyTitle]; got != "Home" {
		t.Errorf("Title of zettel is not \"Home\", but %q", got)
		return
	}
	newTitle := "New Home"
	z.Meta[api.KeyTitle] = newTitle
	err = c.UpdateZettelJSON(context.Background(), api.ZidDefaultHome, z)
	if err != nil {
		t.Error(err)
		return
	}
	zt, err := c.GetZettelJSON(context.Background(), api.ZidDefaultHome)
	if err != nil {
		t.Error(err)
		return
	}
	if got := zt.Meta[api.KeyTitle]; got != newTitle {
		t.Errorf("Title of zettel is not %q, but %q", newTitle, got)
	}
	// No need to clean up, because we just changed the title.
}

func TestListZettel(t *testing.T) {
	testdata := []struct {
		user string
		exp  int
	}{
		{"", 7},
		{"creator", 10},
		{"reader", 12},
		{"writer", 12},
		{"owner", 34},
	}

	t.Parallel()
	c := getClient()
	query := url.Values{api.QueryKeyEncoding: {api.EncodingHTML}} // Client must remove "html"
	for i, tc := range testdata {
		t.Run(fmt.Sprintf("User %d/%q", i, tc.user), func(tt *testing.T) {
			c.SetAuth(tc.user, tc.user)
			l, err := c.ListZettelJSON(context.Background(), query)
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
	l, err := c.ListZettelJSON(context.Background(), url.Values{api.KeyRole: {api.ValueRoleConfiguration}})
	if err != nil {
		t.Error(err)
		return
	}
	got := len(l)
	if got != 27 {
		t.Errorf("List of length %d expected, but got %d\n%v", 27, got, l)
	}

	pl, err := c.ListZettel(context.Background(), url.Values{api.KeyRole: {api.ValueRoleConfiguration}})
	if err != nil {
		t.Error(err)
		return
	}
	lines := strings.Split(pl, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) != len(l) {
		t.Errorf("Different list lenght: Plain=%d, JSON=%d", len(lines), len(l))
	} else {
		for i, line := range lines {
			if got := api.ZettelID(line[:14]); got != l[i].ID {
				t.Errorf("%d: JSON=%q, got=%q", i, l[i].ID, got)
			}
		}
	}
}

func TestGetZettelJSON(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	z, err := c.GetZettelJSON(context.Background(), api.ZidDefaultHome)
	if err != nil {
		t.Error(err)
		return
	}
	if m := z.Meta; len(m) == 0 {
		t.Errorf("Exptected non-empty meta, but got %v", z.Meta)
	}
	if z.Content == "" || z.Encoding != "" {
		t.Errorf("Expect non-empty content, but empty encoding (got %q)", z.Encoding)
	}
}

func TestGetParsedEvaluatedZettel(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	encodings := []api.EncodingEnum{
		api.EncoderDJSON,
		api.EncoderHTML,
		api.EncoderNative,
		api.EncoderText,
	}
	for _, enc := range encodings {
		content, err := c.GetParsedZettel(context.Background(), api.ZidDefaultHome, enc)
		if err != nil {
			t.Error(err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("Empty content for parsed encoding %v", enc)
		}
		content, err = c.GetEvaluatedZettel(context.Background(), api.ZidDefaultHome, enc)
		if err != nil {
			t.Error(err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("Empty content for evaluated encoding %v", enc)
		}
	}
}

func checkZid(t *testing.T, expected, got api.ZettelID) bool {
	t.Helper()
	if expected != got {
		t.Errorf("Expected a Zid %q, but got %q", expected, got)
		return false
	}
	return true
}

func checkListZid(t *testing.T, l []api.ZidMetaJSON, pos int, expected api.ZettelID) {
	t.Helper()
	if got := api.ZettelID(l[pos].ID); got != expected {
		t.Errorf("Expected result[%d]=%v, but got %v", pos, expected, got)
	}
}

func TestGetZettelOrder(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	rl, err := c.GetZettelOrder(context.Background(), api.ZidTOCNewTemplate)
	if err != nil {
		t.Error(err)
		return
	}
	if !checkZid(t, api.ZidTOCNewTemplate, rl.ID) {
		return
	}
	l := rl.List
	if got := len(l); got != 2 {
		t.Errorf("Expected list of length 2, got %d", got)
		return
	}
	checkListZid(t, l, 0, api.ZidTemplateNewZettel)
	checkListZid(t, l, 1, api.ZidTemplateNewUser)
}

func TestGetZettelContext(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	rl, err := c.GetZettelContext(context.Background(), api.ZidVersion, client.DirBoth, 0, 3)
	if err != nil {
		t.Error(err)
		return
	}
	if !checkZid(t, api.ZidVersion, rl.ID) {
		return
	}
	l := rl.List
	if got := len(l); got != 3 {
		t.Errorf("Expected list of length 3, got %d", got)
		return
	}
	checkListZid(t, l, 0, api.ZidDefaultHome)
	checkListZid(t, l, 1, api.ZidOperatingSystem)
	checkListZid(t, l, 2, api.ZidStartupConfiguration)

	rl, err = c.GetZettelContext(context.Background(), api.ZidVersion, client.DirBackward, 0, 0)
	if err != nil {
		t.Error(err)
		return
	}
	if !checkZid(t, api.ZidVersion, rl.ID) {
		return
	}
	l = rl.List
	if got := len(l); got != 1 {
		t.Errorf("Expected list of length 1, got %d", got)
		return
	}
	checkListZid(t, l, 0, api.ZidDefaultHome)
}

func TestGetZettelLinks(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	zl, err := c.GetZettelLinks(context.Background(), api.ZidDefaultHome)
	if err != nil {
		t.Error(err)
		return
	}
	if !checkZid(t, api.ZidDefaultHome, zl.ID) {
		return
	}
	if got := len(zl.Linked.Outgoing); got != 4 {
		t.Errorf("Expected 4 outgoing links, got %d", got)
	}
	if got := len(zl.Linked.Local); got != 1 {
		t.Errorf("Expected 1 local link, got %d", got)
	}
	if got := len(zl.Linked.External); got != 4 {
		t.Errorf("Expected 4 external link, got %d", got)
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

//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
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
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/c/client"
	"zettelstore.de/z/kernel"
)

func nextZid(zid api.ZettelID) api.ZettelID {
	numVal, err := strconv.ParseUint(string(zid), 10, 64)
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

func TestListZettel(t *testing.T) {
	const (
		ownerZettel      = 47
		configRoleZettel = 29
		writerZettel     = ownerZettel - 23
		readerZettel     = ownerZettel - 23
		creatorZettel    = 7
		publicZettel     = 4
	)

	testdata := []struct {
		user string
		exp  int
	}{
		{"", publicZettel},
		{"creator", creatorZettel},
		{"reader", readerZettel},
		{"writer", writerZettel},
		{"owner", ownerZettel},
	}

	t.Parallel()
	c := getClient()
	for i, tc := range testdata {
		t.Run(fmt.Sprintf("User %d/%q", i, tc.user), func(tt *testing.T) {
			c.SetAuth(tc.user, tc.user)
			q, h, l, err := c.ListZettelJSON(context.Background(), "")
			if err != nil {
				tt.Error(err)
				return
			}
			if q != "" {
				tt.Errorf("Query should be empty, but is %q", q)
			}
			if h != "" {
				tt.Errorf("Human should be empty, but is %q", q)
			}
			got := len(l)
			if got != tc.exp {
				tt.Errorf("List of length %d expected, but got %d\n%v", tc.exp, got, l)
			}
		})
	}
	search := api.KeyRole + api.SearchOperatorHas + api.ValueRoleConfiguration + " ORDER id"
	q, h, l, err := c.ListZettelJSON(context.Background(), search)
	if err != nil {
		t.Error(err)
		return
	}
	expQ := "role:configuration ORDER id"
	if q != expQ {
		t.Errorf("Query should be %q, but is %q", expQ, q)
	}
	expH := "role HAS configuration ORDER id"
	if h != expH {
		t.Errorf("Human should be %q, but is %q", expH, h)
	}
	got := len(l)
	if got != configRoleZettel {
		t.Errorf("List of length %d expected, but got %d\n%v", configRoleZettel, got, l)
	}

	pl, err := c.ListZettel(context.Background(), search)
	if err != nil {
		t.Error(err)
		return
	}
	compareZettelList(t, pl, l)
}

func compareZettelList(t *testing.T, pl [][]byte, l []api.ZidMetaJSON) {
	t.Helper()
	if len(pl) != len(l) {
		t.Errorf("Different list lenght: Plain=%d, JSON=%d", len(pl), len(l))
	} else {
		for i, line := range pl {
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

	m, err := c.GetMeta(context.Background(), api.ZidDefaultHome)
	if err != nil {
		t.Error(err)
		return
	}
	if len(m) != len(z.Meta) {
		t.Errorf("Pure meta differs from zettel meta: %s vs %s", m, z.Meta)
		return
	}
	for k, v := range z.Meta {
		got, ok := m[k]
		if !ok {
			t.Errorf("Pure meta has no key %q", k)
			continue
		}
		if got != v {
			t.Errorf("Pure meta has different value for key %q: %q vs %q", k, got, v)
		}
	}
}

func TestGetParsedEvaluatedZettel(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	encodings := []api.EncodingEnum{
		api.EncoderHTML,
		api.EncoderSz,
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

// func TestGetZettelContext(t *testing.T) {
// 	const (
// 		allUserZid = api.ZettelID("20211019200500")
// 		ownerZid   = api.ZettelID("20210629163300")
// 		writerZid  = api.ZettelID("20210629165000")
// 		readerZid  = api.ZettelID("20210629165024")
// 		creatorZid = api.ZettelID("20210629165050")
// 		limitAll   = 3
// 	)
// 	t.Parallel()
// 	c := getClient()
// 	c.SetAuth("owner", "owner")
// 	rl, err := c.GetZettelContext(context.Background(), ownerZid, client.DirBoth, 0, limitAll)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	if !checkZid(t, ownerZid, rl.ID) {
// 		return
// 	}
// 	l := rl.List
// 	if got := len(l); got != limitAll {
// 		t.Errorf("Expected list of length %d, got %d", limitAll, got)
// 		t.Error(rl)
// 		return
// 	}
// 	checkListZid(t, l, 0, allUserZid)
// 	// checkListZid(t, l, 1, writerZid)
// 	// checkListZid(t, l, 2, readerZid)
// 	checkListZid(t, l, 1, creatorZid)

// 	rl, err = c.GetZettelContext(context.Background(), ownerZid, client.DirBackward, 0, 0)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	if !checkZid(t, ownerZid, rl.ID) {
// 		return
// 	}
// 	l = rl.List
// 	if got, exp := len(l), 4; got != exp {
// 		t.Errorf("Expected list of length %d, got %d", exp, got)
// 		return
// 	}
// 	checkListZid(t, l, 0, allUserZid)
// }

func TestGetUnlinkedReferences(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	zl, err := c.GetUnlinkedReferences(context.Background(), api.ZidDefaultHome, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if !checkZid(t, api.ZidDefaultHome, zl.ID) {
		return
	}
	l := zl.List
	if got := len(l); got != 1 {
		t.Errorf("Expected list of length 1, got %d", got)
		return
	}
}

func failNoErrorOrNoCode(t *testing.T, err error, goodCode int) bool {
	if err != nil {
		if cErr, ok := err.(*client.Error); ok {
			if cErr.StatusCode == goodCode {
				return false
			}
			t.Errorf("Expect status code %d, but got client error %v", goodCode, cErr)
		} else {
			t.Errorf("Expect status code %d, but got non-client error %v", goodCode, err)
		}
	} else {
		t.Errorf("No error returned, but status code %d expected", goodCode)
	}
	return true
}

func TestExecuteCommand(t *testing.T) {
	c := getClient()
	err := c.ExecuteCommand(context.Background(), api.Command("xyz"))
	failNoErrorOrNoCode(t, err, http.StatusBadRequest)
	err = c.ExecuteCommand(context.Background(), api.CommandAuthenticated)
	failNoErrorOrNoCode(t, err, http.StatusUnauthorized)
	err = c.ExecuteCommand(context.Background(), api.CommandRefresh)
	failNoErrorOrNoCode(t, err, http.StatusForbidden)

	c.SetAuth("owner", "owner")
	err = c.ExecuteCommand(context.Background(), api.CommandAuthenticated)
	if err != nil {
		t.Error(err)
	}
	err = c.ExecuteCommand(context.Background(), api.CommandRefresh)
	if err != nil {
		t.Error(err)
	}
}

func TestListTags(t *testing.T) {
	t.Parallel()
	c := getClient()
	c.SetAuth("owner", "owner")
	tm, err := c.QueryMapMeta(context.Background(), api.ActionSeparator+api.KeyTags)
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
	rl, err := c.QueryMapMeta(context.Background(), api.ActionSeparator+api.KeyRole)
	if err != nil {
		t.Error(err)
		return
	}
	exp := []string{"configuration", "user", "zettel"}
	if len(rl) != len(exp) {
		t.Errorf("Expected %d different tags, but got only %d (%v)", len(exp), len(rl), rl)
	}
	for _, id := range exp {
		if _, found := rl[id]; !found {
			t.Errorf("Role map expected key %q", id)
		}
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()
	c := getClient()
	ver, err := c.GetVersionJSON(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	if ver.Major != -1 || ver.Minor != -1 || ver.Patch != -1 || ver.Info != kernel.CoreDefaultVersion || ver.Hash != "" {
		t.Error(ver)
	}
}

var baseURL string

func init() {
	flag.StringVar(&baseURL, "base-url", "", "Base URL")
}

func getClient() *client.Client {
	u, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}
	return client.NewClient(u)
}

// TestMain controls whether client API tests should run or not.
func TestMain(m *testing.M) {
	flag.Parse()
	if baseURL != "" {
		m.Run()
	}
}

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

// ---------------------------------------------------------------------------
// Tests that change the Zettelstore must nor run parallel to other tests.

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
	newZettel := `title: Empty Home
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

	// Must delete to clean up for next tests
	c.SetAuth("owner", "owner")
	err = c.DeleteZettel(context.Background(), api.ZidDefaultHome)
	if err != nil {
		t.Error("Cannot delete", api.ZidDefaultHome, ":", err)
		return
	}
}

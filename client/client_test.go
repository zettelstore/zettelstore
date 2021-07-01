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
	"testing"

	"zettelstore.de/z/client"
)

func TestList(t *testing.T) {
	testdata := []struct {
		user string
		exp  int
	}{
		{"reader", 13},
		{"writer", 13},
		{"owner", 34},
		{"", 7},
	}

	t.Parallel()
	for i, tc := range testdata {
		t.Run(fmt.Sprintf("User %d/%q", i, tc.user), func(tt *testing.T) {
			c := getClient()
			c.SetAuth(tc.user, tc.user)
			l, err := c.ListZettel(context.Background())
			if err != nil {
				tt.Error(err)
				return
			}
			got := len(l.List)
			if got != tc.exp {
				tt.Errorf("List of length %d expected, but got %d\n%v", tc.exp, got, l.List)
			}
		})
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

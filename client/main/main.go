//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package main is the starting point for the zettelstore client.
package main

import (
	"context"
	"time"

	"zettelstore.de/z/client"
)

func main() {
	c := client.NewClient("http://127.0.0.1:23123")
	c.SetAuth("abc", "abc")
	c.ListZettel(context.Background())
	time.Sleep(time.Second * 50)
	c.ListZettel(context.Background())
}

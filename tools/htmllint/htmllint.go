//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2023-present Detlef Stern
//-----------------------------------------------------------------------------

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/client.fossil/client"
	"zettelstore.de/z/tools"
)

func main() {
	flag.BoolVar(&tools.Verbose, "v", false, "Verbose output")
	flag.Parse()

	if err := cmdValidateHTML(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
func cmdValidateHTML(args []string) error {
	rawURL := "http://localhost:23123"
	if len(args) > 1 {
		rawURL = args[1]
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	client := client.NewClient(u)
	_, _, metaList, err := client.QueryZettelData(context.Background(), "")
	if err != nil {
		return err
	}
	zids := make([]string, len(metaList))
	for i, m := range metaList {
		zids[i] = string(m.ID)
	}
	sort.Strings(zids)
	for _, zid := range zids {
		err = validateHTML(client, api.ZettelID(zid))
		if err != nil {
			log.Printf("Error while validating zettel %v: %v", zid, err)
		}
	}
	return nil
}

func validateHTML(client *client.Client, zid api.ZettelID) error {
	data, err := client.GetParsedZettel(context.Background(), zid, api.EncoderHTML)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	log.Println("DATA", zid, len(data))
	result, err := tools.ExecuteFilter(data, nil, "tidy", "-e", "-q")
	if err != nil {
		return err
	}
	log.Println("TIDY", result)
	return nil
}

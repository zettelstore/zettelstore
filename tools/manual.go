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
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/input"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func cmdManual() error {
	base := getReleaseVersionData()
	return createManualZip(".", base)
}

func createManualZip(path, base string) error {
	manualPath := filepath.Join("docs", "manual")
	entries, err := os.ReadDir(manualPath)
	if err != nil {
		return err
	}
	zipName := filepath.Join(path, "manual-"+base+".zip")
	zipFile, err := os.OpenFile(zipName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, entry := range entries {
		if err = createManualZipEntry(manualPath, entry, zipWriter); err != nil {
			return err
		}
	}
	return nil
}

const versionZid = "00001000000001"

func createManualZipEntry(path string, entry fs.DirEntry, zipWriter *zip.Writer) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	fh, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	name := entry.Name()
	fh.Name = name
	fh.Method = zip.Deflate
	w, err := zipWriter.CreateHeader(fh)
	if err != nil {
		return err
	}
	manualFile, err := os.Open(filepath.Join(path, name))
	if err != nil {
		return err
	}
	defer manualFile.Close()

	if name != versionZid+".zettel" {
		_, err = io.Copy(w, manualFile)
		return err
	}

	data, err := io.ReadAll(manualFile)
	if err != nil {
		return err
	}
	inp := input.NewInput(data)
	m := meta.NewFromInput(id.MustParse(versionZid), inp)
	m.SetNow(api.KeyModified)

	var buf bytes.Buffer
	if _, err = fmt.Fprintf(&buf, "id: %s\n", versionZid); err != nil {
		return err
	}
	if _, err = m.WriteComputed(&buf); err != nil {
		return err
	}
	version := getVersion()
	if _, err = fmt.Fprintf(&buf, "\n%s", version); err != nil {
		return err
	}
	_, err = io.Copy(w, &buf)
	return err
}

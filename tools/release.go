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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func cmdRelease() error {
	if err := cmdCheck(true); err != nil {
		return err
	}
	base := getReleaseVersionData()
	releases := []struct {
		arch string
		os   string
		env  []string
		name string
	}{
		{"amd64", "linux", nil, "zettelstore"},
		{"arm", "linux", []string{"GOARM=6"}, "zettelstore"},
		{"amd64", "darwin", nil, "zettelstore"},
		{"arm64", "darwin", nil, "zettelstore"},
		{"amd64", "windows", nil, "zettelstore.exe"},
	}
	for _, rel := range releases {
		env := append([]string{}, rel.env...)
		env = append(env, "GOARCH="+rel.arch, "GOOS="+rel.os)
		env = append(env, envDirectProxy...)
		env = append(env, envGoVCS...)
		zsName := filepath.Join("releases", rel.name)
		if err := doBuild(env, base, zsName); err != nil {
			return err
		}
		zipName := fmt.Sprintf("zettelstore-%v-%v-%v.zip", base, rel.os, rel.arch)
		if err := createReleaseZip(zsName, zipName, rel.name); err != nil {
			return err
		}
		if err := os.Remove(zsName); err != nil {
			return err
		}
	}
	return createManualZip("releases", base)
}

func getReleaseVersionData() string {
	if fossil := getFossilDirty(); fossil != "" {
		fmt.Fprintln(os.Stderr, "Warning: releasing a dirty version")
	}
	base := getVersion()
	if strings.HasSuffix(base, "dev") {
		return base[:len(base)-3] + "preview-" + time.Now().Local().Format("20060102")
	}
	return base
}

func createReleaseZip(zsName, zipName, fileName string) error {
	zipFile, err := os.OpenFile(filepath.Join("releases", zipName), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zw := zip.NewWriter(zipFile)
	defer zw.Close()
	err = addFileToZip(zw, zsName, fileName)
	if err != nil {
		return err
	}
	err = addFileToZip(zw, "LICENSE.txt", "LICENSE.txt")
	if err != nil {
		return err
	}
	err = addFileToZip(zw, "docs/readmezip.txt", "README.txt")
	return err
}

func addFileToZip(zipFile *zip.Writer, filepath, filename string) error {
	zsFile, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer zsFile.Close()
	stat, err := zsFile.Stat()
	if err != nil {
		return err
	}
	fh, err := zip.FileInfoHeader(stat)
	if err != nil {
		return err
	}
	fh.Name = filename
	fh.Method = zip.Deflate
	w, err := zipFile.CreateHeader(fh)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, zsFile)
	return err
}

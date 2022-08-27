//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package tests provides some higher-level tests.
package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"

	_ "zettelstore.de/z/box/dirbox"
)

var encodings = []api.EncodingEnum{
	api.EncoderHTML,
	api.EncoderSexpr,
	api.EncoderText,
	api.EncoderZJSON,
}

func getFileBoxes(wd, kind string) (root string, boxes []box.ManagedBox) {
	root = filepath.Clean(filepath.Join(wd, "..", "testdata", kind))
	entries, err := os.ReadDir(root)
	if err != nil {
		panic(err)
	}

	cdata := manager.ConnectData{
		Number:   0,
		Config:   testConfig,
		Enricher: &noEnrich{},
		Notify:   nil,
	}
	for _, entry := range entries {
		if entry.IsDir() {
			u, err2 := url.Parse("dir://" + filepath.Join(root, entry.Name()) + "?type=" + kernel.BoxDirTypeSimple)
			if err2 != nil {
				panic(err2)
			}
			box, err2 := manager.Connect(u, &noAuth{}, &cdata)
			if err2 != nil {
				panic(err2)
			}
			boxes = append(boxes, box)
		}
	}
	return root, boxes
}

type noEnrich struct{}

func (*noEnrich) Enrich(context.Context, *meta.Meta, int) {}
func (*noEnrich) Remove(context.Context, *meta.Meta)      {}

type noAuth struct{}

func (*noAuth) IsReadonly() bool { return false }

func trimLastEOL(s string) string {
	if lastPos := len(s) - 1; lastPos >= 0 && s[lastPos] == '\n' {
		return s[:lastPos]
	}
	return s
}

func resultFile(file string) (data string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	src, err := io.ReadAll(f)
	return string(src), err
}

func checkFileContent(t *testing.T, filename, gotContent string) {
	t.Helper()
	wantContent, err := resultFile(filename)
	if err != nil {
		t.Error(err)
		return
	}
	gotContent = trimLastEOL(gotContent)
	wantContent = trimLastEOL(wantContent)
	if gotContent != wantContent {
		t.Errorf("\nWant: %q\nGot:  %q", wantContent, gotContent)
	}
}

func getBoxName(p box.ManagedBox, root string) string {
	u, err := url.Parse(p.Location())
	if err != nil {
		panic("Unable to parse URL '" + p.Location() + "': " + err.Error())
	}
	return u.Path[len(root):]
}

func checkMetaFile(t *testing.T, resultName string, zn *ast.ZettelNode, enc api.EncodingEnum) {
	t.Helper()

	if enc := encoder.Create(enc); enc != nil {
		var buf bytes.Buffer
		enc.WriteMeta(&buf, zn.Meta, parser.ParseMetadata)
		checkFileContent(t, resultName, buf.String())
		return
	}
	panic(fmt.Sprintf("Unknown writer encoding %q", enc))
}

func checkMetaBox(t *testing.T, p box.ManagedBox, wd, boxName string) {
	ss := p.(box.StartStopper)
	if err := ss.Start(context.Background()); err != nil {
		panic(err)
	}
	metaList := []*meta.Meta{}
	err := p.ApplyMeta(context.Background(), func(m *meta.Meta) { metaList = append(metaList, m) }, nil)
	if err != nil {
		panic(err)
	}
	for _, meta := range metaList {
		zettel, err2 := p.GetZettel(context.Background(), meta.Zid)
		if err2 != nil {
			panic(err2)
		}
		z := parser.ParseZettel(context.Background(), zettel, "", testConfig)
		for _, enc := range encodings {
			t.Run(fmt.Sprintf("%s::%d(%s)", p.Location(), meta.Zid, enc), func(st *testing.T) {
				resultName := filepath.Join(wd, "result", "meta", boxName, z.Zid.String()+"."+enc.String())
				checkMetaFile(st, resultName, z, enc)
			})
		}
	}
	ss.Stop(context.Background())
}

type myConfig struct{}

func (*myConfig) Get(context.Context, *meta.Meta, string) string { return "" }
func (*myConfig) AddDefaultValues(_ context.Context, m *meta.Meta) *meta.Meta {
	return m
}
func (*myConfig) GetHomeZettel() id.Zid         { return id.Invalid }
func (*myConfig) GetListPageSize() int          { return 0 }
func (*myConfig) GetSiteName() string           { return "" }
func (*myConfig) GetYAMLHeader() bool           { return false }
func (*myConfig) GetZettelFileSyntax() []string { return nil }

func (*myConfig) GetSimpleMode() bool                      { return false }
func (*myConfig) GetExpertMode() bool                      { return false }
func (*myConfig) GetVisibility(*meta.Meta) meta.Visibility { return meta.VisibilityPublic }
func (*myConfig) GetMaxTransclusions() int                 { return 1024 }

var testConfig = &myConfig{}

func TestMetaRegression(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, boxes := getFileBoxes(wd, "meta")
	for _, p := range boxes {
		checkMetaBox(t, p, wd, getBoxName(p, root))
	}
}

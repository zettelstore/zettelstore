//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package tests provides some higher-level tests.
package tests

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"

	_ "zettelstore.de/z/box/dirbox"
	_ "zettelstore.de/z/encoder/djsonenc"
	_ "zettelstore.de/z/encoder/htmlenc"
	_ "zettelstore.de/z/encoder/nativeenc"
	_ "zettelstore.de/z/encoder/textenc"
	_ "zettelstore.de/z/encoder/zmkenc"
	_ "zettelstore.de/z/parser/blob"
	_ "zettelstore.de/z/parser/zettelmark"
)

var encodings = []api.EncodingEnum{
	api.EncoderHTML,
	api.EncoderDJSON,
	api.EncoderNative,
	api.EncoderText,
}

func getFileBoxes(wd, kind string) (root string, boxes []box.ManagedBox) {
	root = filepath.Clean(filepath.Join(wd, "..", "testdata", kind))
	entries, err := os.ReadDir(root)
	if err != nil {
		panic(err)
	}

	cdata := manager.ConnectData{Config: testConfig, Enricher: &noEnrich{}, Notify: nil}
	for _, entry := range entries {
		if entry.IsDir() {
			u, err := url.Parse("dir://" + filepath.Join(root, entry.Name()) + "?type=" + kernel.BoxDirTypeSimple)
			if err != nil {
				panic(err)
			}
			box, err := manager.Connect(u, &noAuth{}, &cdata)
			if err != nil {
				panic(err)
			}
			boxes = append(boxes, box)
		}
	}
	return root, boxes
}

type noEnrich struct{}

func (nf *noEnrich) Enrich(context.Context, *meta.Meta, int) {}
func (nf *noEnrich) Remove(context.Context, *meta.Meta)      {}

type noAuth struct{}

func (na *noAuth) IsReadonly() bool { return false }

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

func checkBlocksFile(t *testing.T, resultName string, zn *ast.ZettelNode, enc api.EncodingEnum) {
	t.Helper()
	var env encoder.Environment
	if enc := encoder.Create(enc, &env); enc != nil {
		var sb strings.Builder
		enc.WriteBlocks(&sb, zn.Ast)
		checkFileContent(t, resultName, sb.String())
		return
	}
	panic(fmt.Sprintf("Unknown writer encoding %q", enc))
}

func checkZmkEncoder(t *testing.T, zn *ast.ZettelNode) {
	zmkEncoder := encoder.Create(api.EncoderZmk, nil)
	var sb strings.Builder
	zmkEncoder.WriteBlocks(&sb, zn.Ast)
	gotFirst := sb.String()
	sb.Reset()

	newZettel := parser.ParseZettel(domain.Zettel{
		Meta: zn.Meta, Content: domain.NewContent("\n" + gotFirst)}, "", testConfig)
	zmkEncoder.WriteBlocks(&sb, newZettel.Ast)
	gotSecond := sb.String()
	sb.Reset()

	if gotFirst != gotSecond {
		t.Errorf("\n1st: %q\n2nd: %q", gotFirst, gotSecond)
	}
}

func getBoxName(p box.ManagedBox, root string) string {
	u, err := url.Parse(p.Location())
	if err != nil {
		panic("Unable to parse URL '" + p.Location() + "': " + err.Error())
	}
	return u.Path[len(root):]
}

func match(*meta.Meta) bool { return true }

func checkContentBox(t *testing.T, p box.ManagedBox, wd, boxName string) {
	ss := p.(box.StartStopper)
	if err := ss.Start(context.Background()); err != nil {
		panic(err)
	}
	metaList, err := p.SelectMeta(context.Background(), match)
	if err != nil {
		panic(err)
	}
	for _, meta := range metaList {
		zettel, err := p.GetZettel(context.Background(), meta.Zid)
		if err != nil {
			panic(err)
		}
		z := parser.ParseZettel(zettel, "", testConfig)
		for _, enc := range encodings {
			t.Run(fmt.Sprintf("%s::%d(%s)", p.Location(), meta.Zid, enc), func(st *testing.T) {
				resultName := filepath.Join(wd, "result", "content", boxName, z.Zid.String()+"."+enc.String())
				checkBlocksFile(st, resultName, z, enc)
			})
		}
		t.Run(fmt.Sprintf("%s::%d", p.Location(), meta.Zid), func(st *testing.T) {
			checkZmkEncoder(st, z)
		})
	}
	if err := ss.Stop(context.Background()); err != nil {
		panic(err)
	}
}

func TestContentRegression(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, boxes := getFileBoxes(wd, "content")
	for _, p := range boxes {
		checkContentBox(t, p, wd, getBoxName(p, root))
	}
}

func checkMetaFile(t *testing.T, resultName string, zn *ast.ZettelNode, enc api.EncodingEnum) {
	t.Helper()

	if enc := encoder.Create(enc, nil); enc != nil {
		var sb strings.Builder
		enc.WriteMeta(&sb, zn.Meta)
		checkFileContent(t, resultName, sb.String())
		return
	}
	panic(fmt.Sprintf("Unknown writer encoding %q", enc))
}

func checkMetaBox(t *testing.T, p box.ManagedBox, wd, boxName string) {
	ss := p.(box.StartStopper)
	if err := ss.Start(context.Background()); err != nil {
		panic(err)
	}
	metaList, err := p.SelectMeta(context.Background(), match)
	if err != nil {
		panic(err)
	}
	for _, meta := range metaList {
		zettel, err := p.GetZettel(context.Background(), meta.Zid)
		if err != nil {
			panic(err)
		}
		z := parser.ParseZettel(zettel, "", testConfig)
		for _, enc := range encodings {
			t.Run(fmt.Sprintf("%s::%d(%s)", p.Location(), meta.Zid, enc), func(st *testing.T) {
				resultName := filepath.Join(wd, "result", "meta", boxName, z.Zid.String()+"."+enc.String())
				checkMetaFile(st, resultName, z, enc)
			})
		}
	}
	if err := ss.Stop(context.Background()); err != nil {
		panic(err)
	}
}

type myConfig struct{}

func (cfg *myConfig) AddDefaultValues(m *meta.Meta) *meta.Meta { return m }
func (cfg *myConfig) GetDefaultTitle() string                  { return "" }
func (cfg *myConfig) GetDefaultRole() string                   { return meta.ValueRoleZettel }
func (cfg *myConfig) GetDefaultSyntax() string                 { return meta.ValueSyntaxZmk }
func (cfg *myConfig) GetDefaultLang() string                   { return "" }
func (cfg *myConfig) GetDefaultVisibility() meta.Visibility    { return meta.VisibilityPublic }
func (cfg *myConfig) GetFooterHTML() string                    { return "" }
func (cfg *myConfig) GetHomeZettel() id.Zid                    { return id.Invalid }
func (cfg *myConfig) GetListPageSize() int                     { return 0 }
func (cfg *myConfig) GetMarkerExternal() string                { return "" }
func (cfg *myConfig) GetSiteName() string                      { return "" }
func (cfg *myConfig) GetYAMLHeader() bool                      { return false }
func (cfg *myConfig) GetZettelFileSyntax() []string            { return nil }

func (cfg *myConfig) GetExpertMode() bool                      { return false }
func (cfg *myConfig) GetVisibility(*meta.Meta) meta.Visibility { return cfg.GetDefaultVisibility() }

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

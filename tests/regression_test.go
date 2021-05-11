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

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
	"zettelstore.de/z/service"

	_ "zettelstore.de/z/encoder/htmlenc"
	_ "zettelstore.de/z/encoder/jsonenc"
	_ "zettelstore.de/z/encoder/nativeenc"
	_ "zettelstore.de/z/encoder/textenc"
	_ "zettelstore.de/z/encoder/zmkenc"
	_ "zettelstore.de/z/parser/blob"
	_ "zettelstore.de/z/parser/zettelmark"
	_ "zettelstore.de/z/place/dirplace"
)

var formats = []string{"html", "djson", "native", "text"}

func getFilePlaces(wd string, kind string) (root string, places []place.ManagedPlace) {
	root = filepath.Clean(filepath.Join(wd, "..", "testdata", kind))
	entries, err := os.ReadDir(root)
	if err != nil {
		panic(err)
	}

	cdata := manager.ConnectData{Enricher: &noEnrich{}, Notify: nil}
	for _, entry := range entries {
		if entry.IsDir() {
			place, err := manager.Connect(
				"dir://"+filepath.Join(root, entry.Name())+"?type="+service.PlaceDirTypeSimple,
				false,
				&cdata,
			)
			if err != nil {
				panic(err)
			}
			places = append(places, place)
		}
	}
	return root, places
}

type noEnrich struct{}

func (nf *noEnrich) Enrich(ctx context.Context, m *meta.Meta) {}
func (nf *noEnrich) Remove(ctx context.Context, m *meta.Meta) {}

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

func checkFileContent(t *testing.T, filename string, gotContent string) {
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

func checkBlocksFile(t *testing.T, resultName string, zn *ast.ZettelNode, format string) {
	t.Helper()
	var env encoder.Environment
	if enc := encoder.Create(format, &env); enc != nil {
		var sb strings.Builder
		enc.WriteBlocks(&sb, zn.Ast)
		checkFileContent(t, resultName, sb.String())
		return
	}
	panic(fmt.Sprintf("Unknown writer format %q", format))
}

func checkZmkEncoder(t *testing.T, zn *ast.ZettelNode) {
	zmkEncoder := encoder.Create("zmk", nil)
	var sb strings.Builder
	zmkEncoder.WriteBlocks(&sb, zn.Ast)
	gotFirst := sb.String()
	sb.Reset()

	newZettel := parser.ParseZettel(domain.Zettel{
		Meta: zn.Meta, Content: domain.NewContent("\n" + gotFirst)}, "")
	zmkEncoder.WriteBlocks(&sb, newZettel.Ast)
	gotSecond := sb.String()
	sb.Reset()

	if gotFirst != gotSecond {
		t.Errorf("\n1st: %q\n2nd: %q", gotFirst, gotSecond)
	}
}

func getPlaceName(p place.ManagedPlace, root string) string {
	u, err := url.Parse(p.Location())
	if err != nil {
		panic("Unable to parse URL '" + p.Location() + "': " + err.Error())
	}
	return u.Path[len(root):]
}

func match(*meta.Meta) bool { return true }

func checkContentPlace(t *testing.T, p place.ManagedPlace, wd, placeName string) {
	ss := p.(place.StartStopper)
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
		z := parser.ParseZettel(zettel, "")
		for _, format := range formats {
			t.Run(fmt.Sprintf("%s::%d(%s)", p.Location(), meta.Zid, format), func(st *testing.T) {
				resultName := filepath.Join(wd, "result", "content", placeName, z.Zid.String()+"."+format)
				checkBlocksFile(st, resultName, z, format)
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
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, places := getFilePlaces(wd, "content")
	for _, p := range places {
		checkContentPlace(t, p, wd, getPlaceName(p, root))
	}
}

func checkMetaFile(t *testing.T, resultName string, zn *ast.ZettelNode, format string) {
	t.Helper()

	if enc := encoder.Create(format, nil); enc != nil {
		var sb strings.Builder
		enc.WriteMeta(&sb, zn.Meta)
		checkFileContent(t, resultName, sb.String())
		return
	}
	panic(fmt.Sprintf("Unknown writer format %q", format))
}

func checkMetaPlace(t *testing.T, p place.ManagedPlace, wd, placeName string) {
	ss := p.(place.StartStopper)
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
		z := parser.ParseZettel(zettel, "")
		for _, format := range formats {
			t.Run(fmt.Sprintf("%s::%d(%s)", p.Location(), meta.Zid, format), func(st *testing.T) {
				resultName := filepath.Join(wd, "result", "meta", placeName, z.Zid.String()+"."+format)
				checkMetaFile(st, resultName, z, format)
			})
		}
	}
	if err := ss.Stop(context.Background()); err != nil {
		panic(err)
	}
}

func TestMetaRegression(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, places := getFilePlaces(wd, "meta")
	for _, p := range places {
		checkMetaPlace(t, p, wd, getPlaceName(p, root))
	}
}

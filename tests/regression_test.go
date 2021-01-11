//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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
	"io/ioutil"
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

	_ "zettelstore.de/z/encoder/htmlenc"
	_ "zettelstore.de/z/encoder/jsonenc"
	_ "zettelstore.de/z/encoder/nativeenc"
	_ "zettelstore.de/z/encoder/textenc"
	_ "zettelstore.de/z/encoder/zmkenc"
	_ "zettelstore.de/z/parser/blob"
	_ "zettelstore.de/z/parser/zettelmark"
	_ "zettelstore.de/z/place/dirplace"
	"zettelstore.de/z/place/manager"
)

var formats = []string{"html", "djson", "native", "text"}

func getFilePlaces(wd string, kind string) (root string, places []place.Place) {
	root = filepath.Clean(filepath.Join(wd, "..", "testdata", kind))
	infos, err := ioutil.ReadDir(root)
	if err != nil {
		panic(err)
	}

	for _, info := range infos {
		if info.Mode().IsDir() {
			place, err := manager.Connect(
				"dir://"+filepath.Join(root, info.Name()),
				false,
				&noFilter{},
			)
			if err != nil {
				panic(err)
			}
			places = append(places, place)
		}
	}
	return root, places
}

type noFilter struct{}

func (nf *noFilter) UpdateProperties(m *meta.Meta) {}
func (nf *noFilter) RemoveProperties(m *meta.Meta) {}

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
	src, err := ioutil.ReadAll(f)
	return string(src), err
}

func checkFileContent(t *testing.T, filename string, gotContent string) {
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
	if enc := encoder.Create(format); enc != nil {
		var sb strings.Builder
		enc.WriteBlocks(&sb, zn.Ast)
		checkFileContent(t, resultName, sb.String())
		return
	}
	panic(fmt.Sprintf("Unknown writer format %q", format))
}

func checkZmkEncoder(t *testing.T, zn *ast.ZettelNode) {
	zmkEncoder := encoder.Create("zmk")
	var sb strings.Builder
	zmkEncoder.WriteBlocks(&sb, zn.Ast)
	gotFirst := sb.String()
	sb.Reset()

	newZettel := parser.ParseZettel(domain.Zettel{
		Meta: zn.Zettel.Meta, Content: domain.NewContent("\n" + gotFirst)}, "")
	zmkEncoder.WriteBlocks(&sb, newZettel.Ast)
	gotSecond := sb.String()
	sb.Reset()

	if gotFirst != gotSecond {
		t.Errorf("\n1st: %q\n2nd: %q", gotFirst, gotSecond)
	}
}

func TestContentRegression(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, places := getFilePlaces(wd, "content")
	for _, place := range places {
		if err := place.Start(context.Background()); err != nil {
			panic(err)
		}
		placeName := place.Location()[len("dir://")+len(root):]
		metaList, err := place.SelectMeta(context.Background(), nil, nil)
		if err != nil {
			panic(err)
		}
		for _, meta := range metaList {
			zettel, err := place.GetZettel(context.Background(), meta.Zid)
			if err != nil {
				panic(err)
			}
			z := parser.ParseZettel(zettel, "")
			for _, format := range formats {
				t.Run(fmt.Sprintf("%s::%d(%s)", place.Location(), meta.Zid, format), func(st *testing.T) {
					resultName := filepath.Join(wd, "result", "content", placeName, z.Zid.String()+"."+format)
					checkBlocksFile(st, resultName, z, format)
				})
			}
			t.Run(fmt.Sprintf("%s::%d", place.Location(), meta.Zid), func(st *testing.T) {
				checkZmkEncoder(st, z)
			})
		}
		if err := place.Stop(context.Background()); err != nil {
			panic(err)
		}
	}
}

func checkMetaFile(t *testing.T, resultName string, zn *ast.ZettelNode, format string) {
	t.Helper()

	if enc := encoder.Create(format); enc != nil {
		var sb strings.Builder
		enc.WriteMeta(&sb, zn.Zettel.Meta)
		checkFileContent(t, resultName, sb.String())
		return
	}
	panic(fmt.Sprintf("Unknown writer format %q", format))
}

func TestMetaRegression(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, places := getFilePlaces(wd, "meta")
	for _, place := range places {
		if err := place.Start(context.Background()); err != nil {
			panic(err)
		}
		placeName := place.Location()[len("dir://")+len(root):]
		metaList, err := place.SelectMeta(context.Background(), nil, nil)
		if err != nil {
			panic(err)
		}
		for _, meta := range metaList {
			zettel, err := place.GetZettel(context.Background(), meta.Zid)
			if err != nil {
				panic(err)
			}
			z := parser.ParseZettel(zettel, "")
			for _, format := range formats {
				t.Run(fmt.Sprintf("%s::%d(%s)", place.Location(), meta.Zid, format), func(st *testing.T) {
					resultName := filepath.Join(wd, "result", "meta", placeName, z.Zid.String()+"."+format)
					checkMetaFile(st, resultName, z, format)
				})
			}
		}
		if err := place.Stop(context.Background()); err != nil {
			panic(err)
		}
	}
}

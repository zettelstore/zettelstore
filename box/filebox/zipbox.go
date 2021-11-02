//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package filebox provides boxes that are stored in a file.
package filebox

import (
	"archive/zip"
	"context"
	"io"
	"regexp"
	"strings"

	"zettelstore.de/z/box"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
)

var validFileName = regexp.MustCompile(`^(\d{14}).*(\.(.+))$`)

func matchValidFileName(name string) []string {
	return validFileName.FindStringSubmatch(name)
}

type zipEntry struct {
	metaName     string
	contentName  string
	contentExt   string // (normalized) file extension of zettel content
	metaInHeader bool
}

type zipBox struct {
	number   int
	name     string
	enricher box.Enricher
	zettel   map[id.Zid]*zipEntry // no lock needed, because read-only after creation
}

func (zp *zipBox) Location() string {
	if strings.HasPrefix(zp.name, "/") {
		return "file://" + zp.name
	}
	return "file:" + zp.name
}

func (zp *zipBox) Start(context.Context) error {
	reader, err := zip.OpenReader(zp.name)
	if err != nil {
		return err
	}
	defer reader.Close()
	zp.zettel = make(map[id.Zid]*zipEntry)
	for _, f := range reader.File {
		match := matchValidFileName(f.Name)
		if len(match) < 1 {
			continue
		}
		zid, err := id.Parse(match[1])
		if err != nil {
			continue
		}
		zp.addFile(zid, f.Name, match[3])
	}
	return nil
}

func (zp *zipBox) addFile(zid id.Zid, name, ext string) {
	entry := zp.zettel[zid]
	if entry == nil {
		entry = &zipEntry{}
		zp.zettel[zid] = entry
	}
	switch ext {
	case "zettel":
		if entry.contentExt == "" {
			entry.contentName = name
			entry.contentExt = ext
			entry.metaInHeader = true
		}
	case "meta":
		entry.metaName = name
		entry.metaInHeader = false
	default:
		if entry.contentExt == "" {
			entry.contentExt = ext
			entry.contentName = name
		}
	}
}

func (zp *zipBox) Stop(context.Context) error {
	zp.zettel = nil
	return nil
}

func (*zipBox) CanCreateZettel(context.Context) bool { return false }

func (*zipBox) CreateZettel(context.Context, domain.Zettel) (id.Zid, error) {
	return id.Invalid, box.ErrReadOnly
}

func (zp *zipBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	entry, ok := zp.zettel[zid]
	if !ok {
		return domain.Zettel{}, box.ErrNotFound
	}
	reader, err := zip.OpenReader(zp.name)
	if err != nil {
		return domain.Zettel{}, err
	}
	defer reader.Close()

	var m *meta.Meta
	var src []byte
	var inMeta bool
	if entry.metaInHeader {
		src, err = readZipFileContent(reader, entry.contentName)
		if err != nil {
			return domain.Zettel{}, err
		}
		inp := input.NewInput(src)
		m = meta.NewFromInput(zid, inp)
		src = src[inp.Pos:]
	} else if metaName := entry.metaName; metaName != "" {
		m, err = readZipMetaFile(reader, zid, metaName)
		if err != nil {
			return domain.Zettel{}, err
		}
		src, err = readZipFileContent(reader, entry.contentName)
		if err != nil {
			return domain.Zettel{}, err
		}
		inMeta = true
	} else {
		m = CalcDefaultMeta(zid, entry.contentExt)
	}
	CleanupMeta(m, zid, entry.contentExt, inMeta, false)
	return domain.Zettel{Meta: m, Content: domain.NewContent(src)}, nil
}

func (zp *zipBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	entry, ok := zp.zettel[zid]
	if !ok {
		return nil, box.ErrNotFound
	}
	reader, err := zip.OpenReader(zp.name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return readZipMeta(reader, zid, entry)
}

func (zp *zipBox) ApplyZid(_ context.Context, handle box.ZidFunc) error {
	for zid := range zp.zettel {
		handle(zid)
	}
	return nil
}

func (zp *zipBox) ApplyMeta(ctx context.Context, handle box.MetaFunc) error {
	reader, err := zip.OpenReader(zp.name)
	if err != nil {
		return err
	}
	defer reader.Close()
	for zid, entry := range zp.zettel {
		m, err := readZipMeta(reader, zid, entry)
		if err != nil {
			continue
		}
		zp.enricher.Enrich(ctx, m, zp.number)
		handle(m)
	}
	return nil
}

func (*zipBox) CanUpdateZettel(context.Context, domain.Zettel) bool { return false }

func (*zipBox) UpdateZettel(context.Context, domain.Zettel) error { return box.ErrReadOnly }

func (zp *zipBox) AllowRenameZettel(_ context.Context, zid id.Zid) bool {
	_, ok := zp.zettel[zid]
	return !ok
}

func (zp *zipBox) RenameZettel(_ context.Context, curZid, _ id.Zid) error {
	if _, ok := zp.zettel[curZid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}

func (*zipBox) CanDeleteZettel(context.Context, id.Zid) bool { return false }

func (zp *zipBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	if _, ok := zp.zettel[zid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}

func (zp *zipBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = len(zp.zettel)
}

func readZipMeta(reader *zip.ReadCloser, zid id.Zid, entry *zipEntry) (m *meta.Meta, err error) {
	var inMeta bool
	if entry.metaInHeader {
		m, err = readZipMetaFile(reader, zid, entry.contentName)
	} else if metaName := entry.metaName; metaName != "" {
		m, err = readZipMetaFile(reader, zid, entry.metaName)
		inMeta = true
	} else {
		m = CalcDefaultMeta(zid, entry.contentExt)
	}
	if err == nil {
		CleanupMeta(m, zid, entry.contentExt, inMeta, false)
	}
	return m, err
}

func readZipMetaFile(reader *zip.ReadCloser, zid id.Zid, name string) (*meta.Meta, error) {
	src, err := readZipFileContent(reader, name)
	if err != nil {
		return nil, err
	}
	inp := input.NewInput(src)
	return meta.NewFromInput(zid, inp), nil
}

func readZipFileContent(reader *zip.ReadCloser, name string) ([]byte, error) {
	f, err := reader.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

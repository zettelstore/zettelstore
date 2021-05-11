//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package fileplace provides places that are stored in a file.
package fileplace

import (
	"archive/zip"
	"context"
	"io"
	"regexp"
	"strings"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/place"
	"zettelstore.de/z/search"
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

type zipPlace struct {
	name     string
	enricher place.Enricher
	zettel   map[id.Zid]*zipEntry // no lock needed, because read-only after creation
}

func (zp *zipPlace) Location() string {
	if strings.HasPrefix(zp.name, "/") {
		return "file://" + zp.name
	}
	return "file:" + zp.name
}

func (zp *zipPlace) Start(ctx context.Context) error {
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

func (zp *zipPlace) addFile(zid id.Zid, name, ext string) {
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

func (zp *zipPlace) Stop(ctx context.Context) error {
	zp.zettel = nil
	return nil
}

func (zp *zipPlace) CanCreateZettel(ctx context.Context) bool { return false }

func (zp *zipPlace) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	return id.Invalid, place.ErrReadOnly
}

func (zp *zipPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	entry, ok := zp.zettel[zid]
	if !ok {
		return domain.Zettel{}, place.ErrNotFound
	}
	reader, err := zip.OpenReader(zp.name)
	if err != nil {
		return domain.Zettel{}, err
	}
	defer reader.Close()

	var m *meta.Meta
	var src string
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

func (zp *zipPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	entry, ok := zp.zettel[zid]
	if !ok {
		return nil, place.ErrNotFound
	}
	reader, err := zip.OpenReader(zp.name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return readZipMeta(reader, zid, entry)
}

func (zp *zipPlace) FetchZids(ctx context.Context) (id.Set, error) {
	result := id.NewSetCap(len(zp.zettel))
	for zid := range zp.zettel {
		result[zid] = true
	}
	return result, nil
}

func (zp *zipPlace) SelectMeta(ctx context.Context, match search.MetaMatchFunc) (res []*meta.Meta, err error) {
	reader, err := zip.OpenReader(zp.name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	for zid, entry := range zp.zettel {
		m, err := readZipMeta(reader, zid, entry)
		if err != nil {
			continue
		}
		zp.enricher.Enrich(ctx, m)
		if match(m) {
			res = append(res, m)
		}
	}
	return res, nil
}

func (zp *zipPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return false
}

func (zp *zipPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	return place.ErrReadOnly
}

func (zp *zipPlace) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	_, ok := zp.zettel[zid]
	return !ok
}

func (zp *zipPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if _, ok := zp.zettel[curZid]; ok {
		return place.ErrReadOnly
	}
	return place.ErrNotFound
}

func (zp *zipPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool { return false }

func (zp *zipPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	if _, ok := zp.zettel[zid]; ok {
		return place.ErrReadOnly
	}
	return place.ErrNotFound
}

func (zp *zipPlace) ReadStats(st *place.Stats) {
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

func readZipFileContent(reader *zip.ReadCloser, name string) (string, error) {
	f, err := reader.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

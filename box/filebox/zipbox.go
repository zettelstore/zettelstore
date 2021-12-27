//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package filebox

import (
	"archive/zip"
	"context"
	"io"
	"regexp"
	"strings"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/notify"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/logger"
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

type zipBox struct {
	log      *logger.Logger
	number   int
	name     string
	enricher box.Enricher
	zettel   map[id.Zid]*zipEntry // no lock needed, because read-only after creation
	notify   chan<- box.UpdateInfo
	dirSrv   *notify.DirService
}

func (zb *zipBox) Location() string {
	if strings.HasPrefix(zb.name, "/") {
		return "file://" + zb.name
	}
	return "file:" + zb.name
}

func (zb *zipBox) Start(context.Context) error {
	reader, err := zip.OpenReader(zb.name)
	if err != nil {
		return err
	}
	reader.Close()
	zipNotifier, err := notify.NewSimpleZipNotifier(zb.log, zb.name)
	if err != nil {
		return err
	}
	zb.dirSrv = notify.NewDirService(zb.log, zipNotifier, zb.notify)
	zb.zettel = make(map[id.Zid]*zipEntry)
	for _, f := range reader.File {
		match := matchValidFileName(f.Name)
		if len(match) < 1 {
			continue
		}
		zid, err2 := id.Parse(match[1])
		if err2 != nil {
			continue
		}
		zb.addFile(zid, f.Name, match[3])
	}
	return nil
}

func (zb *zipBox) addFile(zid id.Zid, name, ext string) {
	entry := zb.zettel[zid]
	if entry == nil {
		entry = &zipEntry{}
		zb.zettel[zid] = entry
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

func (zb *zipBox) Stop(context.Context) {
	zb.zettel = nil
}

func (*zipBox) CanCreateZettel(context.Context) bool { return false }

func (zb *zipBox) CreateZettel(context.Context, domain.Zettel) (id.Zid, error) {
	zb.log.Trace().Err(box.ErrReadOnly).Msg("CreateZettel")
	return id.Invalid, box.ErrReadOnly
}

func (zb *zipBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	entry, ok := zb.zettel[zid]
	if !ok {
		return domain.Zettel{}, box.ErrNotFound
	}
	reader, err := zip.OpenReader(zb.name)
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
	zb.log.Trace().Msg("GetZettel")
	return domain.Zettel{Meta: m, Content: domain.NewContent(src)}, nil
}

func (zb *zipBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	entry, ok := zb.zettel[zid]
	if !ok {
		return nil, box.ErrNotFound
	}
	reader, err := zip.OpenReader(zb.name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	m, err := readZipMeta(reader, zid, entry)
	zb.log.Trace().Err(err).Msg("GetMeta")
	return m, err
}

func (zb *zipBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint search.RetrievePredicate) error {
	zb.log.Trace().Int("entries", int64(len(zb.zettel))).Msg("ApplyZid")
	for zid := range zb.zettel {
		if constraint(zid) {
			handle(zid)
		}
	}
	return nil
}

func (zb *zipBox) ApplyMeta(ctx context.Context, handle box.MetaFunc, constraint search.RetrievePredicate) error {
	reader, err := zip.OpenReader(zb.name)
	if err != nil {
		return err
	}
	defer reader.Close()
	zb.log.Trace().Int("entries", int64(len(zb.zettel))).Msg("ApplyMeta")
	for zid, entry := range zb.zettel {
		if !constraint(zid) {
			continue
		}
		m, err2 := readZipMeta(reader, zid, entry)
		if err2 != nil {
			continue
		}
		zb.enricher.Enrich(ctx, m, zb.number)
		handle(m)
	}
	return nil
}

func (*zipBox) CanUpdateZettel(context.Context, domain.Zettel) bool { return false }

func (zb *zipBox) UpdateZettel(context.Context, domain.Zettel) error {
	zb.log.Trace().Err(box.ErrReadOnly).Msg("UpdateZettel")
	return box.ErrReadOnly
}

func (zb *zipBox) AllowRenameZettel(_ context.Context, zid id.Zid) bool {
	_, ok := zb.zettel[zid]
	return !ok
}

func (zb *zipBox) RenameZettel(_ context.Context, curZid, _ id.Zid) error {
	err := box.ErrNotFound
	if _, ok := zb.zettel[curZid]; ok {
		err = box.ErrReadOnly
	}
	zb.log.Trace().Err(err).Msg("RenameZettel")
	return err
}

func (*zipBox) CanDeleteZettel(context.Context, id.Zid) bool { return false }

func (zb *zipBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	err := box.ErrNotFound
	if _, ok := zb.zettel[zid]; ok {
		err = box.ErrReadOnly
	}
	zb.log.Trace().Err(err).Msg("DeleteZettel")
	return err
}

func (zb *zipBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = len(zb.zettel)
	zb.log.Trace().Int("zettel", int64(st.Zettel)).Msg("ReadStats")
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

//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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

type zipBox struct {
	log      *logger.Logger
	number   int
	name     string
	enricher box.Enricher
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
	zb.dirSrv.Start()
	return nil
}

func (zb *zipBox) Refresh(_ context.Context) {
	zb.dirSrv.Refresh()
	zb.log.Trace().Msg("Refresh")
}

func (zb *zipBox) Stop(context.Context) {
	zb.dirSrv.Stop()
}

func (*zipBox) CanCreateZettel(context.Context) bool { return false }

func (zb *zipBox) CreateZettel(context.Context, domain.Zettel) (id.Zid, error) {
	err := box.ErrReadOnly
	zb.log.Trace().Err(err).Msg("CreateZettel")
	return id.Invalid, err
}

func (zb *zipBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	entry := zb.dirSrv.GetDirEntry(zid)
	if !entry.IsValid() {
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

	contentName := entry.ContentName
	if metaName := entry.MetaName; metaName == "" {
		if contentName == "" {
			zb.log.Panic().Zid(zid).Msg("No meta, no content in zipBox.GetZettel")
		}
		src, err = readZipFileContent(reader, entry.ContentName)
		if err != nil {
			return domain.Zettel{}, err
		}
		if entry.HasMetaInContent() {
			inp := input.NewInput(src)
			m = meta.NewFromInput(zid, inp)
			src = src[inp.Pos:]
		} else {
			m = CalcDefaultMeta(zid, entry.ContentExt)
		}
	} else {
		m, err = readZipMetaFile(reader, zid, metaName)
		if err != nil {
			return domain.Zettel{}, err
		}
		inMeta = true
		if contentName != "" {
			src, err = readZipFileContent(reader, entry.ContentName)
			if err != nil {
				return domain.Zettel{}, err
			}
		}
	}

	CleanupMeta(m, zid, entry.ContentExt, inMeta, entry.UselessFiles)
	zb.log.Trace().Zid(zid).Msg("GetZettel")
	return domain.Zettel{Meta: m, Content: domain.NewContent(src)}, nil
}

func (zb *zipBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	entry := zb.dirSrv.GetDirEntry(zid)
	if !entry.IsValid() {
		return nil, box.ErrNotFound
	}
	reader, err := zip.OpenReader(zb.name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	m, err := zb.readZipMeta(reader, zid, entry)
	zb.log.Trace().Err(err).Zid(zid).Msg("GetMeta")
	return m, err
}

func (zb *zipBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint search.RetrievePredicate) error {
	entries := zb.dirSrv.GetDirEntries(constraint)
	zb.log.Trace().Int("entries", int64(len(entries))).Msg("ApplyZid")
	for _, entry := range entries {
		handle(entry.Zid)
	}
	return nil
}

func (zb *zipBox) ApplyMeta(ctx context.Context, handle box.MetaFunc, constraint search.RetrievePredicate) error {
	reader, err := zip.OpenReader(zb.name)
	if err != nil {
		return err
	}
	defer reader.Close()
	entries := zb.dirSrv.GetDirEntries(constraint)
	zb.log.Trace().Int("entries", int64(len(entries))).Msg("ApplyMeta")
	for _, entry := range entries {
		if !constraint(entry.Zid) {
			continue
		}
		m, err2 := zb.readZipMeta(reader, entry.Zid, entry)
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
	err := box.ErrReadOnly
	zb.log.Trace().Err(err).Msg("UpdateZettel")
	return err
}

func (zb *zipBox) AllowRenameZettel(_ context.Context, zid id.Zid) bool {
	entry := zb.dirSrv.GetDirEntry(zid)
	return !entry.IsValid()
}

func (zb *zipBox) RenameZettel(_ context.Context, curZid, newZid id.Zid) error {
	err := box.ErrReadOnly
	if curZid == newZid {
		err = nil
	}
	curEntry := zb.dirSrv.GetDirEntry(curZid)
	if !curEntry.IsValid() {
		err = box.ErrNotFound
	}
	zb.log.Trace().Err(err).Msg("RenameZettel")
	return err
}

func (*zipBox) CanDeleteZettel(context.Context, id.Zid) bool { return false }

func (zb *zipBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	err := box.ErrReadOnly
	entry := zb.dirSrv.GetDirEntry(zid)
	if !entry.IsValid() {
		err = box.ErrNotFound
	}
	zb.log.Trace().Err(err).Msg("DeleteZettel")
	return err
}

func (zb *zipBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = zb.dirSrv.NumDirEntries()
	zb.log.Trace().Int("zettel", int64(st.Zettel)).Msg("ReadStats")
}

func (zb *zipBox) readZipMeta(reader *zip.ReadCloser, zid id.Zid, entry *notify.DirEntry) (m *meta.Meta, err error) {
	var inMeta bool
	if metaName := entry.MetaName; metaName == "" {
		contentName := entry.ContentName
		contentExt := entry.ContentExt
		if contentName == "" || contentExt == "" {
			zb.log.Panic().Zid(zid).Msg("No meta, no content in getMeta")
		}
		if entry.HasMetaInContent() {
			m, err = readZipMetaFile(reader, zid, contentName)
		} else {
			m = CalcDefaultMeta(zid, contentExt)
		}
	} else {
		m, err = readZipMetaFile(reader, zid, metaName)
	}
	if err == nil {
		CleanupMeta(m, zid, entry.ContentExt, inMeta, entry.UselessFiles)
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

//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package dirbox provides a directory-based zettel box.
package dirbox

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/dirbox/directory"
	"zettelstore.de/z/box/filebox"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

func init() {
	manager.Register("dir", func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
		path := getDirPath(u)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		dirSrvSpec, defWorker, maxWorker := getDirSrvInfo(u.Query().Get("type"))
		dp := dirBox{
			number:     cdata.Number,
			location:   u.String(),
			readonly:   getQueryBool(u, "readonly"),
			cdata:      *cdata,
			dir:        path,
			dirRescan:  time.Duration(getQueryInt(u, "rescan", 60, 3600, 30*24*60*60)) * time.Second,
			dirSrvSpec: dirSrvSpec,
			fSrvs:      uint32(getQueryInt(u, "worker", 1, defWorker, maxWorker)),
		}
		return &dp, nil
	})
}

type directoryServiceSpec int

const (
	_ directoryServiceSpec = iota
	dirSrvAny
	dirSrvSimple
	dirSrvNotify
)

func getDirPath(u *url.URL) string {
	if u.Opaque != "" {
		return filepath.Clean(u.Opaque)
	}
	return filepath.Clean(u.Path)
}

func getQueryBool(u *url.URL, key string) bool {
	_, ok := u.Query()[key]
	return ok
}

func getQueryInt(u *url.URL, key string, min, def, max int) int {
	sVal := u.Query().Get(key)
	if sVal == "" {
		return def
	}
	iVal, err := strconv.Atoi(sVal)
	if err != nil {
		return def
	}
	if iVal < min {
		return min
	}
	if iVal > max {
		return max
	}
	return iVal
}

// dirBox uses a directory to store zettel as files.
type dirBox struct {
	number     int
	location   string
	readonly   bool
	cdata      manager.ConnectData
	dir        string
	dirRescan  time.Duration
	dirSrvSpec directoryServiceSpec
	dirSrv     directory.Service
	mustNotify bool
	fSrvs      uint32
	fCmds      []chan fileCmd
	mxCmds     sync.RWMutex
}

func (dp *dirBox) Location() string {
	return dp.location
}

func (dp *dirBox) Start(ctx context.Context) error {
	dp.mxCmds.Lock()
	dp.fCmds = make([]chan fileCmd, 0, dp.fSrvs)
	for i := uint32(0); i < dp.fSrvs; i++ {
		cc := make(chan fileCmd)
		go fileService(i, cc)
		dp.fCmds = append(dp.fCmds, cc)
	}
	dp.setupDirService()
	dp.mxCmds.Unlock()
	if dp.dirSrv == nil {
		panic("No directory service")
	}
	return dp.dirSrv.Start()
}

func (dp *dirBox) Stop(ctx context.Context) error {
	dirSrv := dp.dirSrv
	dp.dirSrv = nil
	err := dirSrv.Stop()
	for _, c := range dp.fCmds {
		close(c)
	}
	return err
}

func (dp *dirBox) notifyChanged(reason box.UpdateReason, zid id.Zid) {
	if dp.mustNotify {
		if chci := dp.cdata.Notify; chci != nil {
			chci <- box.UpdateInfo{Reason: reason, Zid: zid}
		}
	}
}

func (dp *dirBox) getFileChan(zid id.Zid) chan fileCmd {
	// Based on https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function
	sum := 2166136261 ^ uint32(zid)
	sum *= 16777619
	sum ^= uint32(zid >> 32)
	sum *= 16777619

	dp.mxCmds.RLock()
	defer dp.mxCmds.RUnlock()
	return dp.fCmds[sum%dp.fSrvs]
}

func (dp *dirBox) CanCreateZettel(ctx context.Context) bool {
	return !dp.readonly
}

func (dp *dirBox) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	if dp.readonly {
		return id.Invalid, box.ErrReadOnly
	}

	entry, err := dp.dirSrv.GetNew()
	if err != nil {
		return id.Invalid, err
	}
	meta := zettel.Meta
	meta.Zid = entry.Zid
	dp.updateEntryFromMeta(entry, meta)

	err = setZettel(dp, entry, zettel)
	if err == nil {
		dp.dirSrv.UpdateEntry(entry)
	}
	dp.notifyChanged(box.OnUpdate, meta.Zid)
	return meta.Zid, err
}

func (dp *dirBox) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	entry, err := dp.dirSrv.GetEntry(zid)
	if err != nil || !entry.IsValid() {
		return domain.Zettel{}, box.ErrNotFound
	}
	m, c, err := getMetaContent(dp, entry, zid)
	if err != nil {
		return domain.Zettel{}, err
	}
	dp.cleanupMeta(ctx, m)
	zettel := domain.Zettel{Meta: m, Content: domain.NewContent(c)}
	return zettel, nil
}

func (dp *dirBox) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	entry, err := dp.dirSrv.GetEntry(zid)
	if err != nil || !entry.IsValid() {
		return nil, box.ErrNotFound
	}
	m, err := getMeta(dp, entry, zid)
	if err != nil {
		return nil, err
	}
	dp.cleanupMeta(ctx, m)
	return m, nil
}

func (dp *dirBox) FetchZids(ctx context.Context) (id.Set, error) {
	entries, err := dp.dirSrv.GetEntries()
	if err != nil {
		return nil, err
	}
	result := id.NewSetCap(len(entries))
	for _, entry := range entries {
		result[entry.Zid] = true
	}
	return result, nil
}

func (dp *dirBox) SelectMeta(ctx context.Context, match search.MetaMatchFunc) (res []*meta.Meta, err error) {
	entries, err := dp.dirSrv.GetEntries()
	if err != nil {
		return nil, err
	}
	res = make([]*meta.Meta, 0, len(entries))
	// The following loop could be parallelized if needed for performance.
	for _, entry := range entries {
		m, err1 := getMeta(dp, entry, entry.Zid)
		err = err1
		if err != nil {
			continue
		}
		dp.cleanupMeta(ctx, m)
		dp.cdata.Enricher.Enrich(ctx, m, dp.number)

		if match(m) {
			res = append(res, m)
		}
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (dp *dirBox) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return !dp.readonly
}

func (dp *dirBox) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	if dp.readonly {
		return box.ErrReadOnly
	}

	meta := zettel.Meta
	if !meta.Zid.IsValid() {
		return &box.ErrInvalidID{Zid: meta.Zid}
	}
	entry, err := dp.dirSrv.GetEntry(meta.Zid)
	if err != nil {
		return err
	}
	if !entry.IsValid() {
		// Existing zettel, but new in this box.
		entry = &directory.Entry{Zid: meta.Zid}
		dp.updateEntryFromMeta(entry, meta)
	} else if entry.MetaSpec == directory.MetaSpecNone {
		defaultMeta := filebox.CalcDefaultMeta(entry.Zid, entry.ContentExt)
		if !meta.Equal(defaultMeta, true) {
			dp.updateEntryFromMeta(entry, meta)
			dp.dirSrv.UpdateEntry(entry)
		}
	}
	err = setZettel(dp, entry, zettel)
	if err == nil {
		dp.notifyChanged(box.OnUpdate, meta.Zid)
	}
	return err
}

func (dp *dirBox) updateEntryFromMeta(entry *directory.Entry, meta *meta.Meta) {
	entry.MetaSpec, entry.ContentExt = dp.calcSpecExt(meta)
	basePath := dp.calcBasePath(entry)
	if entry.MetaSpec == directory.MetaSpecFile {
		entry.MetaPath = basePath + ".meta"
	}
	entry.ContentPath = basePath + "." + entry.ContentExt
	entry.Duplicates = false
}

func (dp *dirBox) calcBasePath(entry *directory.Entry) string {
	p := entry.ContentPath
	if p == "" {
		return filepath.Join(dp.dir, entry.Zid.String())
	}
	// ContentPath w/o the file extension
	return p[0 : len(p)-len(filepath.Ext(p))]
}

func (dp *dirBox) calcSpecExt(m *meta.Meta) (directory.MetaSpec, string) {
	if m.YamlSep {
		return directory.MetaSpecHeader, "zettel"
	}
	syntax := m.GetDefault(meta.KeySyntax, "bin")
	switch syntax {
	case meta.ValueSyntaxNone, meta.ValueSyntaxZmk:
		return directory.MetaSpecHeader, "zettel"
	}
	for _, s := range dp.cdata.Config.GetZettelFileSyntax() {
		if s == syntax {
			return directory.MetaSpecHeader, "zettel"
		}
	}
	return directory.MetaSpecFile, syntax
}

func (dp *dirBox) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	return !dp.readonly
}

func (dp *dirBox) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if dp.readonly {
		return box.ErrReadOnly
	}
	if curZid == newZid {
		return nil
	}
	curEntry, err := dp.dirSrv.GetEntry(curZid)
	if err != nil || !curEntry.IsValid() {
		return box.ErrNotFound
	}

	// Check whether zettel with new ID already exists in this box.
	if _, err = dp.GetMeta(ctx, newZid); err == nil {
		return &box.ErrInvalidID{Zid: newZid}
	}

	oldMeta, oldContent, err := getMetaContent(dp, curEntry, curZid)
	if err != nil {
		return err
	}

	newEntry := directory.Entry{
		Zid:         newZid,
		MetaSpec:    curEntry.MetaSpec,
		MetaPath:    renamePath(curEntry.MetaPath, curZid, newZid),
		ContentPath: renamePath(curEntry.ContentPath, curZid, newZid),
		ContentExt:  curEntry.ContentExt,
	}

	if err = dp.dirSrv.RenameEntry(curEntry, &newEntry); err != nil {
		return err
	}
	oldMeta.Zid = newZid
	newZettel := domain.Zettel{Meta: oldMeta, Content: domain.NewContent(oldContent)}
	if err = setZettel(dp, &newEntry, newZettel); err != nil {
		// "Rollback" rename. No error checking...
		dp.dirSrv.RenameEntry(&newEntry, curEntry)
		return err
	}
	err = deleteZettel(dp, curEntry, curZid)
	if err == nil {
		dp.notifyChanged(box.OnDelete, curZid)
		dp.notifyChanged(box.OnUpdate, newZid)
	}
	return err
}

func (dp *dirBox) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	if dp.readonly {
		return false
	}
	entry, err := dp.dirSrv.GetEntry(zid)
	return err == nil && entry.IsValid()
}

func (dp *dirBox) DeleteZettel(ctx context.Context, zid id.Zid) error {
	if dp.readonly {
		return box.ErrReadOnly
	}

	entry, err := dp.dirSrv.GetEntry(zid)
	if err != nil || !entry.IsValid() {
		return box.ErrNotFound
	}
	dp.dirSrv.DeleteEntry(zid)
	err = deleteZettel(dp, entry, zid)
	if err == nil {
		dp.notifyChanged(box.OnDelete, zid)
	}
	return err
}

func (dp *dirBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = dp.readonly
	st.Zettel, _ = dp.dirSrv.NumEntries()
}

func (dp *dirBox) cleanupMeta(ctx context.Context, m *meta.Meta) {
	if role, ok := m.Get(meta.KeyRole); !ok || role == "" {
		m.Set(meta.KeyRole, dp.cdata.Config.GetDefaultRole())
	}
	if syntax, ok := m.Get(meta.KeySyntax); !ok || syntax == "" {
		m.Set(meta.KeySyntax, dp.cdata.Config.GetDefaultSyntax())
	}
}

func renamePath(path string, curID, newID id.Zid) string {
	dir, file := filepath.Split(path)
	if cur := curID.String(); strings.HasPrefix(file, cur) {
		file = newID.String() + file[len(cur):]
		return filepath.Join(dir, file)
	}
	return path
}

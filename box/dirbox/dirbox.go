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

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/filebox"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/box/notify"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/search"
)

func init() {
	manager.Register("dir", func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
		var log *logger.Logger
		if krnl := kernel.Main; krnl != nil {
			log = krnl.GetLogger(kernel.BoxService).Clone().Str("box", "dir").Int("boxnum", int64(cdata.Number)).Child()
		}
		path := getDirPath(u)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		dp := dirBox{
			log:        log,
			number:     cdata.Number,
			location:   u.String(),
			readonly:   getQueryBool(u, "readonly"),
			cdata:      *cdata,
			dir:        path,
			notifySpec: getDirSrvInfo(log, u.Query().Get("type")),
			fSrvs:      makePrime(uint32(getQueryInt(u, "worker", 1, 7, 1499))),
		}
		return &dp, nil
	})
}

func makePrime(n uint32) uint32 {
	for !isPrime(n) {
		n++
	}
	return n
}

func isPrime(n uint32) bool {
	if n == 0 {
		return false
	}
	if n <= 3 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	for i := uint32(3); i*i <= n; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

type notifyTypeSpec int

const (
	_ notifyTypeSpec = iota
	dirNotifyAny
	dirNotifySimple
	dirNotifyFS
)

func getDirSrvInfo(log *logger.Logger, notifyType string) notifyTypeSpec {
	for count := 0; count < 2; count++ {
		switch notifyType {
		case kernel.BoxDirTypeNotify:
			return dirNotifyFS
		case kernel.BoxDirTypeSimple:
			return dirNotifySimple
		default:
			notifyType = kernel.Main.GetConfig(kernel.BoxService, kernel.BoxDefaultDirType).(string)
		}
	}
	log.Error().Str("notifyType", notifyType).Msg("Unable to set notify type, using a default")
	return dirNotifySimple
}

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
	log        *logger.Logger
	number     int
	location   string
	readonly   bool
	cdata      manager.ConnectData
	dir        string
	notifySpec notifyTypeSpec
	dirSrv     *notify.DirService
	fSrvs      uint32
	fCmds      []chan fileCmd
	mxCmds     sync.RWMutex
}

func (dp *dirBox) Location() string {
	return dp.location
}

func (dp *dirBox) Start(context.Context) error {
	dp.mxCmds.Lock()
	defer dp.mxCmds.Unlock()
	dp.fCmds = make([]chan fileCmd, 0, dp.fSrvs)
	for i := uint32(0); i < dp.fSrvs; i++ {
		cc := make(chan fileCmd)
		go fileService(i, dp.log.Clone().Str("sub", "file").Uint("fn", uint64(i)).Child(), dp.dir, cc)
		dp.fCmds = append(dp.fCmds, cc)
	}

	var notifier notify.Notifier
	var err error
	switch dp.notifySpec {
	case dirNotifySimple:
		notifier, err = notify.NewSimpleDirNotifier(dp.log.Clone().Str("notify", "simple").Child(), dp.dir)
	default:
		notifier, err = notify.NewFSDirNotifier(dp.log.Clone().Str("notify", "fs").Child(), dp.dir)
	}
	if err != nil {
		dp.log.Fatal().Err(err).Msg("Unable to create directory supervisor")
		dp.stopFileServices()
		return err
	}
	dp.dirSrv = notify.NewDirService(
		dp.log.Clone().Str("sub", "dirsrv").Child(),
		notifier,
		dp.cdata.Notify,
	)
	dp.dirSrv.Start()
	return nil
}

func (dp *dirBox) Refresh(_ context.Context) {
	dp.dirSrv.Refresh()
	dp.log.Trace().Msg("Refresh")
}

func (dp *dirBox) Stop(_ context.Context) {
	dirSrv := dp.dirSrv
	dp.dirSrv = nil
	if dirSrv != nil {
		dirSrv.Stop()
	}
	dp.stopFileServices()
}

func (dp *dirBox) stopFileServices() {
	for _, c := range dp.fCmds {
		close(c)
	}
}

func (dp *dirBox) notifyChanged(reason box.UpdateReason, zid id.Zid) {
	if chci := dp.cdata.Notify; chci != nil {
		dp.log.Trace().Zid(zid).Uint("reason", uint64(reason)).Msg("notifyChanged")
		chci <- box.UpdateInfo{Reason: reason, Zid: zid}
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

func (dp *dirBox) CanCreateZettel(_ context.Context) bool {
	return !dp.readonly
}

func (dp *dirBox) CreateZettel(_ context.Context, zettel domain.Zettel) (id.Zid, error) {
	if dp.readonly {
		return id.Invalid, box.ErrReadOnly
	}

	newZid, err := dp.dirSrv.SetNewDirEntry()
	if err != nil {
		return id.Invalid, err
	}
	meta := zettel.Meta
	meta.Zid = newZid
	entry := notify.DirEntry{Zid: newZid}
	dp.updateEntryFromMeta(&entry, meta)

	err = setZettel(dp, &entry, zettel)
	if err == nil {
		err = dp.dirSrv.UpdateDirEntry(&entry)
	}
	dp.notifyChanged(box.OnUpdate, meta.Zid)
	dp.log.Trace().Err(err).Zid(meta.Zid).Msg("CreateZettel")
	return meta.Zid, err
}

func (dp *dirBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	entry := dp.dirSrv.GetDirEntry(zid)
	if !entry.IsValid() {
		return domain.Zettel{}, box.ErrNotFound
	}
	m, c, err := getMetaContent(dp, entry, zid)
	if err != nil {
		return domain.Zettel{}, err
	}
	dp.cleanupMeta(m)
	zettel := domain.Zettel{Meta: m, Content: domain.NewContent(c)}
	dp.log.Trace().Msg("GetZettel")
	return zettel, nil
}

func (dp *dirBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	entry := dp.dirSrv.GetDirEntry(zid)
	if !entry.IsValid() {
		return nil, box.ErrNotFound
	}
	m, err := getMeta(dp, entry, zid)
	if err != nil {
		return nil, err
	}
	dp.cleanupMeta(m)
	dp.log.Trace().Msg("GetZettel")
	return m, nil
}

func (dp *dirBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint search.RetrievePredicate) error {
	entries := dp.dirSrv.GetDirEntries(constraint)
	dp.log.Trace().Int("entries", int64(len(entries))).Msg("ApplyZid")
	for _, entry := range entries {
		handle(entry.Zid)
	}
	return nil
}

func (dp *dirBox) ApplyMeta(ctx context.Context, handle box.MetaFunc, constraint search.RetrievePredicate) error {
	entries := dp.dirSrv.GetDirEntries(constraint)
	dp.log.Trace().Int("entries", int64(len(entries))).Msg("ApplyMeta")

	// The following loop could be parallelized if needed for performance.
	for _, entry := range entries {
		m, err := getMeta(dp, entry, entry.Zid)
		if err != nil {
			dp.log.Trace().Err(err).Msg("ApplyMeta/getMeta")
			return err
		}
		dp.cleanupMeta(m)
		dp.cdata.Enricher.Enrich(ctx, m, dp.number)
		handle(m)
	}
	return nil
}

func (dp *dirBox) CanUpdateZettel(context.Context, domain.Zettel) bool {
	return !dp.readonly
}

func (dp *dirBox) UpdateZettel(_ context.Context, zettel domain.Zettel) error {
	if dp.readonly {
		return box.ErrReadOnly
	}

	meta := zettel.Meta
	if !meta.Zid.IsValid() {
		return &box.ErrInvalidID{Zid: meta.Zid}
	}
	entry := dp.dirSrv.GetDirEntry(meta.Zid)
	if !entry.IsValid() {
		// Existing zettel, but new in this box.
		entry = &notify.DirEntry{Zid: meta.Zid}
		dp.updateEntryFromMeta(entry, meta)
		dp.dirSrv.UpdateDirEntry(entry)
	} else if entry.MetaSpec == notify.DirMetaSpecNone {
		defaultMeta := filebox.CalcDefaultMeta(entry.Zid, entry.ContentExt)
		if !meta.Equal(defaultMeta, true) {
			dp.updateEntryFromMeta(entry, meta)
			dp.dirSrv.UpdateDirEntry(entry)
		}
	}
	err := setZettel(dp, entry, zettel)
	if err == nil {
		dp.notifyChanged(box.OnUpdate, meta.Zid)
	}
	dp.log.Trace().Err(err).Msg("UpdateZettel")
	return err
}

func (dp *dirBox) updateEntryFromMeta(entry *notify.DirEntry, meta *meta.Meta) {
	entry.MetaSpec, entry.ContentExt = dp.calcSpecExt(meta)

	var baseName string
	if p := entry.ContentName; p != "" {
		// ContentName w/o the file extension
		baseName = p[0 : len(p)-len(filepath.Ext(p))]
	} else {
		baseName = entry.Zid.String()
	}

	if entry.MetaSpec == notify.DirMetaSpecFile {
		entry.MetaName = baseName + ".meta"
	}
	entry.ContentName = baseName + "." + entry.ContentExt
	entry.Duplicates = false
}

func (dp *dirBox) calcSpecExt(m *meta.Meta) (notify.DirMetaSpec, string) {
	if m.YamlSep {
		return notify.DirMetaSpecHeader, "zettel"
	}
	syntax := m.GetDefault(api.KeySyntax, "bin")
	switch syntax {
	case api.ValueSyntaxNone, api.ValueSyntaxZmk:
		return notify.DirMetaSpecHeader, "zettel"
	}
	for _, s := range dp.cdata.Config.GetZettelFileSyntax() {
		if s == syntax {
			return notify.DirMetaSpecHeader, "zettel"
		}
	}
	return notify.DirMetaSpecFile, syntax
}

func (dp *dirBox) AllowRenameZettel(context.Context, id.Zid) bool {
	return !dp.readonly
}

func (dp *dirBox) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if curZid == newZid {
		return nil
	}
	curEntry := dp.dirSrv.GetDirEntry(curZid)
	if !curEntry.IsValid() {
		return box.ErrNotFound
	}
	if dp.readonly {
		return box.ErrReadOnly
	}

	// Check whether zettel with new ID already exists in this box.
	if _, err := dp.GetMeta(ctx, newZid); err == nil {
		return &box.ErrInvalidID{Zid: newZid}
	}

	oldMeta, oldContent, err := getMetaContent(dp, curEntry, curZid)
	if err != nil {
		return err
	}

	newEntry := notify.DirEntry{
		Zid:         newZid,
		MetaSpec:    curEntry.MetaSpec,
		MetaName:    renameFilename(curEntry.MetaName, curZid, newZid),
		ContentName: renameFilename(curEntry.ContentName, curZid, newZid),
		ContentExt:  curEntry.ContentExt,
	}

	if err = dp.dirSrv.RenameDirEntry(curEntry, &newEntry); err != nil {
		return err
	}
	oldMeta.Zid = newZid
	newZettel := domain.Zettel{Meta: oldMeta, Content: domain.NewContent(oldContent)}
	if err = setZettel(dp, &newEntry, newZettel); err != nil {
		// "Rollback" rename. No error checking...
		dp.dirSrv.RenameDirEntry(&newEntry, curEntry)
		return err
	}
	err = deleteZettel(dp, curEntry, curZid)
	if err == nil {
		dp.notifyChanged(box.OnDelete, curZid)
		dp.notifyChanged(box.OnUpdate, newZid)
	}
	dp.log.Trace().Err(err).Msg("RenameZettel")
	return err
}

func (dp *dirBox) CanDeleteZettel(_ context.Context, zid id.Zid) bool {
	if dp.readonly {
		return false
	}
	entry := dp.dirSrv.GetDirEntry(zid)
	return entry.IsValid()
}

func (dp *dirBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	if dp.readonly {
		return box.ErrReadOnly
	}

	entry := dp.dirSrv.GetDirEntry(zid)
	if !entry.IsValid() {
		return box.ErrNotFound
	}
	err := dp.dirSrv.DeleteDirEntry(zid)
	if err != nil {
		return nil
	}
	err = deleteZettel(dp, entry, zid)
	if err == nil {
		dp.notifyChanged(box.OnDelete, zid)
	}
	dp.log.Trace().Err(err).Msg("DeleteZettel")
	return err
}

func (dp *dirBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = dp.readonly
	st.Zettel = dp.dirSrv.NumDirEntries()
	dp.log.Trace().Int("zettel", int64(st.Zettel)).Msg("ReadStats")
}

func (dp *dirBox) cleanupMeta(m *meta.Meta) {
	if role, ok := m.Get(api.KeyRole); !ok || role == "" {
		m.Set(api.KeyRole, dp.cdata.Config.GetDefaultRole())
	}
	if syntax, ok := m.Get(api.KeySyntax); !ok || syntax == "" {
		m.Set(api.KeySyntax, dp.cdata.Config.GetDefaultSyntax())
	}
}

func renameFilename(name string, curID, newID id.Zid) string {
	if cur := curID.String(); strings.HasPrefix(name, cur) {
		name = newID.String() + name[len(cur):]
	}
	return name
}

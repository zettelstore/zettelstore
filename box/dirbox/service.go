//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package dirbox

import (
	"os"
	"path/filepath"

	"zettelstore.de/z/box/filebox"
	"zettelstore.de/z/box/notify"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
)

func fileService(i uint32, log *logger.Logger, dirPath string, cmds <-chan fileCmd) {
	// Something may panic. Ensure a running service.
	defer func() {
		if r := recover(); r != nil {
			kernel.Main.LogRecover("FileService", r)
			go fileService(i, log, dirPath, cmds)
		}
	}()

	log.Trace().Uint("i", uint64(i)).Str("dirpath", dirPath).Msg("File service started")
	for cmd := range cmds {
		cmd.run(log, dirPath)
	}
	log.Trace().Uint("i", uint64(i)).Str("dirpath", dirPath).Msg("File service stopped")
}

type fileCmd interface {
	run(*logger.Logger, string)
}

// COMMAND: getMeta ----------------------------------------
//
// Retrieves the meta data from a zettel.

func getMeta(dp *dirBox, entry *notify.DirEntry, zid id.Zid) (*meta.Meta, error) {
	rc := make(chan resGetMeta)
	dp.getFileChan(zid) <- &fileGetMeta{entry, rc}
	res := <-rc
	close(rc)
	return res.meta, res.err
}

type fileGetMeta struct {
	entry *notify.DirEntry
	rc    chan<- resGetMeta
}
type resGetMeta struct {
	meta *meta.Meta
	err  error
}

func (cmd *fileGetMeta) run(log *logger.Logger, dirPath string) {
	var m *meta.Meta
	var err error

	entry := cmd.entry
	zid := entry.Zid
	if metaName := entry.MetaName; metaName == "" {
		contentName := entry.ContentName
		contentExt := entry.ContentExt
		if contentName == "" || contentExt == "" {
			log.Panic().Zid(zid).Msg("No meta, no content in getMeta")
		}
		if entry.HasMetaInContent() {
			m, _, err = parseMetaContentFile(zid, filepath.Join(dirPath, contentName))
		} else {
			m = filebox.CalcDefaultMeta(zid, contentExt)
		}
	} else {
		m, err = parseMetaFile(zid, filepath.Join(dirPath, metaName))
	}
	if err == nil {
		cmdCleanupMeta(m, entry)
	}
	cmd.rc <- resGetMeta{m, err}
}

// COMMAND: getMetaContent ----------------------------------------
//
// Retrieves the meta data and the content of a zettel.

func getMetaContent(dp *dirBox, entry *notify.DirEntry, zid id.Zid) (*meta.Meta, []byte, error) {
	rc := make(chan resGetMetaContent)
	dp.getFileChan(zid) <- &fileGetMetaContent{entry, rc}
	res := <-rc
	close(rc)
	return res.meta, res.content, res.err
}

type fileGetMetaContent struct {
	entry *notify.DirEntry
	rc    chan<- resGetMetaContent
}
type resGetMetaContent struct {
	meta    *meta.Meta
	content []byte
	err     error
}

func (cmd *fileGetMetaContent) run(log *logger.Logger, dirPath string) {
	var m *meta.Meta
	var content []byte
	var err error

	entry := cmd.entry
	zid := entry.Zid
	contentName := entry.ContentName
	contentExt := entry.ContentExt
	contentPath := filepath.Join(dirPath, contentName)
	if metaName := entry.MetaName; metaName == "" {
		if contentName == "" || contentExt == "" {
			log.Panic().Zid(zid).Msg("No meta, no content in getMetaContent")
		}
		if entry.HasMetaInContent() {
			m, content, err = parseMetaContentFile(zid, contentPath)
		} else {
			m = filebox.CalcDefaultMeta(zid, contentExt)
			content, err = os.ReadFile(contentPath)
		}
	} else {
		m, err = parseMetaFile(zid, filepath.Join(dirPath, metaName))
		if contentName != "" {
			var err1 error
			content, err1 = os.ReadFile(contentPath)
			if err == nil {
				err = err1
			}
		}
	}
	if err == nil {
		cmdCleanupMeta(m, entry)
	}
	cmd.rc <- resGetMetaContent{m, content, err}
}

// COMMAND: setZettel ----------------------------------------
//
// Writes a new or exsting zettel.

func setZettel(dp *dirBox, entry *notify.DirEntry, zettel domain.Zettel) error {
	rc := make(chan resSetZettel)
	dp.getFileChan(zettel.Meta.Zid) <- &fileSetZettel{entry, zettel, rc}
	err := <-rc
	close(rc)
	return err
}

type fileSetZettel struct {
	entry  *notify.DirEntry
	zettel domain.Zettel
	rc     chan<- resSetZettel
}
type resSetZettel = error

func (cmd *fileSetZettel) run(log *logger.Logger, dirPath string) {
	entry := cmd.entry
	zid := entry.Zid
	contentName := entry.ContentName
	m := cmd.zettel.Meta
	content := cmd.zettel.Content.AsBytes()
	metaName := entry.MetaName
	if metaName == "" {
		if contentName == "" {
			log.Panic().Zid(zid).Msg("No meta, no content in setZettel")
		}
		contentPath := filepath.Join(dirPath, contentName)
		if entry.HasMetaInContent() {
			err := writeZettelFile(contentPath, m, content)
			cmd.rc <- err
			return
		}
		err := writeFileContent(contentPath, content)
		cmd.rc <- err
		return
	}

	err := writeMetaFile(filepath.Join(dirPath, metaName), m)
	if err == nil && contentName != "" {
		err = writeFileContent(filepath.Join(dirPath, contentName), content)
	}
	cmd.rc <- err
}

func writeMetaFile(metaPath string, m *meta.Meta) error {
	metaFile, err := openFileWrite(metaPath)
	if err != nil {
		return err
	}
	err = writeFileZid(metaFile, m.Zid)
	if err == nil {
		_, err = m.WriteComputed(metaFile)
	}
	if err1 := metaFile.Close(); err == nil {
		err = err1
	}
	return err
}

func writeZettelFile(contentPath string, m *meta.Meta, content []byte) error {
	zettelFile, err := openFileWrite(contentPath)
	if err != nil {
		return err
	}
	err = writeFileZid(zettelFile, m.Zid)
	if err == nil {
		_, err = m.WriteAsHeader(zettelFile)
	}
	if err == nil {
		_, err = zettelFile.Write(content)
	}
	if err1 := zettelFile.Close(); err == nil {
		err = err1
	}
	return err
}

// COMMAND: deleteZettel ----------------------------------------
//
// Deletes an existing zettel.

func deleteZettel(dp *dirBox, entry *notify.DirEntry, zid id.Zid) error {
	rc := make(chan resDeleteZettel)
	dp.getFileChan(zid) <- &fileDeleteZettel{entry, rc}
	err := <-rc
	close(rc)
	return err
}

type fileDeleteZettel struct {
	entry *notify.DirEntry
	rc    chan<- resDeleteZettel
}
type resDeleteZettel = error

func (cmd *fileDeleteZettel) run(log *logger.Logger, dirPath string) {
	var err error

	entry := cmd.entry
	contentName := entry.ContentName
	contentPath := filepath.Join(dirPath, contentName)
	if metaName := entry.MetaName; metaName == "" {
		if contentName == "" {
			log.Panic().Zid(entry.Zid).Msg("No meta, no content in getMetaContent")
		}
		err = os.Remove(contentPath)
	} else {
		if contentName != "" {
			err = os.Remove(contentPath)
		}
		err1 := os.Remove(filepath.Join(dirPath, metaName))
		if err == nil {
			err = err1
		}
	}
	for _, dupName := range entry.UselessFiles {
		err1 := os.Remove(filepath.Join(dirPath, dupName))
		if err == nil {
			err = err1
		}
	}
	cmd.rc <- err
}

// Utility functions ----------------------------------------

func parseMetaFile(zid id.Zid, path string) (*meta.Meta, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	inp := input.NewInput(src)
	return meta.NewFromInput(zid, inp), nil
}

func parseMetaContentFile(zid id.Zid, path string) (*meta.Meta, []byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	inp := input.NewInput(src)
	meta := meta.NewFromInput(zid, inp)
	return meta, src[inp.Pos:], nil
}

func cmdCleanupMeta(m *meta.Meta, entry *notify.DirEntry) {
	filebox.CleanupMeta(
		m,
		entry.Zid, entry.ContentExt,
		entry.MetaSpec == notify.DirMetaSpecFile,
		entry.UselessFiles,
	)
}

func openFileWrite(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
}

func writeFileZid(f *os.File, zid id.Zid) error {
	_, err := f.WriteString("id: ")
	if err == nil {
		_, err = f.Write(zid.Bytes())
		if err == nil {
			_, err = f.WriteString("\n")
		}
	}
	return err
}

func writeFileContent(path string, content []byte) error {
	f, err := openFileWrite(path)
	if err == nil {
		_, err = f.Write(content)
		if err1 := f.Close(); err == nil {
			err = err1
		}
	}
	return err
}

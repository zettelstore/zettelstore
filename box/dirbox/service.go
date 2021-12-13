//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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

	"zettelstore.de/z/box/dirbox/directory"
	"zettelstore.de/z/box/filebox"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
)

func fileService(log *logger.Logger, cmds <-chan fileCmd) {
	// Something may panic. Ensure a running service.
	defer func() {
		if r := recover(); r != nil {
			kernel.Main.LogRecover("FileService", r)
			go fileService(log, cmds)
		}
	}()

	for cmd := range cmds {
		cmd.run(log)
	}
}

type fileCmd interface {
	run(*logger.Logger)
}

// COMMAND: getMeta ----------------------------------------
//
// Retrieves the meta data from a zettel.

func getMeta(dp *dirBox, entry *directory.Entry, zid id.Zid) (*meta.Meta, error) {
	rc := make(chan resGetMeta)
	dp.getFileChan(zid) <- &fileGetMeta{entry, rc}
	res := <-rc
	close(rc)
	return res.meta, res.err
}

type fileGetMeta struct {
	entry *directory.Entry
	rc    chan<- resGetMeta
}
type resGetMeta struct {
	meta *meta.Meta
	err  error
}

func (cmd *fileGetMeta) run(*logger.Logger) {
	entry := cmd.entry
	var m *meta.Meta
	var err error
	switch entry.MetaSpec {
	case directory.MetaSpecFile:
		m, err = parseMetaFile(entry.Zid, entry.MetaPath)
	case directory.MetaSpecHeader:
		m, _, err = parseMetaContentFile(entry.Zid, entry.ContentPath)
	default:
		m = filebox.CalcDefaultMeta(entry.Zid, entry.ContentExt)
	}
	if err == nil {
		cmdCleanupMeta(m, entry)
	}
	cmd.rc <- resGetMeta{m, err}
}

// COMMAND: getMetaContent ----------------------------------------
//
// Retrieves the meta data and the content of a zettel.

func getMetaContent(dp *dirBox, entry *directory.Entry, zid id.Zid) (*meta.Meta, []byte, error) {
	rc := make(chan resGetMetaContent)
	dp.getFileChan(zid) <- &fileGetMetaContent{entry, rc}
	res := <-rc
	close(rc)
	return res.meta, res.content, res.err
}

type fileGetMetaContent struct {
	entry *directory.Entry
	rc    chan<- resGetMetaContent
}
type resGetMetaContent struct {
	meta    *meta.Meta
	content []byte
	err     error
}

func (cmd *fileGetMetaContent) run(*logger.Logger) {
	var m *meta.Meta
	var content []byte
	var err error

	entry := cmd.entry
	switch entry.MetaSpec {
	case directory.MetaSpecFile:
		m, err = parseMetaFile(entry.Zid, entry.MetaPath)
		if err == nil {
			content, err = readFileContent(entry.ContentPath)
		}
	case directory.MetaSpecHeader:
		m, content, err = parseMetaContentFile(entry.Zid, entry.ContentPath)
	default:
		m = filebox.CalcDefaultMeta(entry.Zid, entry.ContentExt)
		content, err = readFileContent(entry.ContentPath)
	}
	if err == nil {
		cmdCleanupMeta(m, entry)
	}
	cmd.rc <- resGetMetaContent{m, content, err}
}

// COMMAND: setZettel ----------------------------------------
//
// Writes a new or exsting zettel.

func setZettel(dp *dirBox, entry *directory.Entry, zettel domain.Zettel) error {
	rc := make(chan resSetZettel)
	dp.getFileChan(zettel.Meta.Zid) <- &fileSetZettel{entry, zettel, rc}
	err := <-rc
	close(rc)
	return err
}

type fileSetZettel struct {
	entry  *directory.Entry
	zettel domain.Zettel
	rc     chan<- resSetZettel
}
type resSetZettel = error

func (cmd *fileSetZettel) run(log *logger.Logger) {
	var err error
	switch ms := cmd.entry.MetaSpec; ms {
	case directory.MetaSpecFile:
		err = cmd.runMetaSpecFile()
	case directory.MetaSpecHeader:
		err = cmd.runMetaSpecHeader()
	case directory.MetaSpecNone:
		// TODO: if meta has some additional infos: write meta to new .meta;
		// update entry in dir
		err = writeFileContent(cmd.entry.ContentPath, cmd.zettel.Content.AsString())
	default:
		log.Panic().Uint("metaspec", uint64(ms)).Msg("set")
		panic("TODO: ???")
	}
	cmd.rc <- err
}

func (cmd *fileSetZettel) runMetaSpecFile() error {
	f, err := openFileWrite(cmd.entry.MetaPath)
	if err == nil {
		err = writeFileZid(f, cmd.zettel.Meta.Zid)
		if err == nil {
			_, err = cmd.zettel.Meta.Write(f, true)
			if err1 := f.Close(); err == nil {
				err = err1
			}
			if err == nil {
				err = writeFileContent(cmd.entry.ContentPath, cmd.zettel.Content.AsString())
			}
		}
	}
	return err
}

func (cmd *fileSetZettel) runMetaSpecHeader() error {
	f, err := openFileWrite(cmd.entry.ContentPath)
	if err == nil {
		err = writeFileZid(f, cmd.zettel.Meta.Zid)
		if err == nil {
			_, err = cmd.zettel.Meta.WriteAsHeader(f, true)
			if err == nil {
				_, err = f.WriteString(cmd.zettel.Content.AsString())
				if err1 := f.Close(); err == nil {
					err = err1
				}
			}
		}
	}
	return err
}

// COMMAND: deleteZettel ----------------------------------------
//
// Deletes an existing zettel.

func deleteZettel(dp *dirBox, entry *directory.Entry, zid id.Zid) error {
	rc := make(chan resDeleteZettel)
	dp.getFileChan(zid) <- &fileDeleteZettel{entry, rc}
	err := <-rc
	close(rc)
	return err
}

type fileDeleteZettel struct {
	entry *directory.Entry
	rc    chan<- resDeleteZettel
}
type resDeleteZettel = error

func (cmd *fileDeleteZettel) run(log *logger.Logger) {
	var err error

	switch ms := cmd.entry.MetaSpec; ms {
	case directory.MetaSpecFile:
		err1 := os.Remove(cmd.entry.MetaPath)
		err = os.Remove(cmd.entry.ContentPath)
		if err == nil {
			err = err1
		}
	case directory.MetaSpecHeader:
		err = os.Remove(cmd.entry.ContentPath)
	case directory.MetaSpecNone:
		err = os.Remove(cmd.entry.ContentPath)
	default:
		log.Panic().Uint("metaspec", uint64(ms)).Msg("delete")
		panic("TODO: ???")
	}
	cmd.rc <- err
}

// Utility functions ----------------------------------------

func readFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func parseMetaFile(zid id.Zid, path string) (*meta.Meta, error) {
	src, err := readFileContent(path)
	if err != nil {
		return nil, err
	}
	inp := input.NewInput(src)
	return meta.NewFromInput(zid, inp), nil
}

func parseMetaContentFile(zid id.Zid, path string) (*meta.Meta, []byte, error) {
	src, err := readFileContent(path)
	if err != nil {
		return nil, nil, err
	}
	inp := input.NewInput(src)
	meta := meta.NewFromInput(zid, inp)
	return meta, src[inp.Pos:], nil
}

func cmdCleanupMeta(m *meta.Meta, entry *directory.Entry) {
	filebox.CleanupMeta(
		m,
		entry.Zid, entry.ContentExt,
		entry.MetaSpec == directory.MetaSpecFile,
		entry.Duplicates,
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

func writeFileContent(path, content string) error {
	f, err := openFileWrite(path)
	if err == nil {
		_, err = f.WriteString(content)
		if err1 := f.Close(); err == nil {
			err = err1
		}
	}
	return err
}

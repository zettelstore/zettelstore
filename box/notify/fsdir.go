//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package notify

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"zettelstore.de/z/logger"
)

type fsdirNotifier struct {
	log     *logger.Logger
	events  chan Event
	done    chan struct{}
	refresh chan struct{}
	base    *fsnotify.Watcher
	path    string
	parent  string
}

// NewFSDirNotifier creates a directory based notifier that receives notifications
// from the file system.
func NewFSDirNotifier(log *logger.Logger, path string) (Notifier, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Unable to create absolute path")
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Debug().Err(err).Str("absPath", absPath).Msg("Unable to create watcher")
		return nil, err
	}
	absParentDir := filepath.Dir(absPath)
	errParent := watcher.Add(absParentDir)
	err = watcher.Add(absPath)
	if errParent != nil {
		if err != nil {
			log.Error().
				Str("parentDir", absParentDir).Err(errParent).
				Str("path", absPath).Err(err).
				Msg("Unable to access Zettel directory and its parent directory")
			watcher.Close()
			return nil, err
		}
		log.Warn().
			Str("parentDir", absParentDir).Err(errParent).
			Msg("Parent of Zettel directory cannot be supervised")
		log.Warn().Str("path", absPath).
			Msg("Zettelstore might not detect a deletion or movement of the Zettel directory")
	} else if err != nil {
		// Not a problem, if container is not available. It might become available later.
		log.Warn().Err(err).Str("path", absPath).Msg("Zettel directory not available")
	}

	fsdn := &fsdirNotifier{
		log:     log,
		events:  make(chan Event),
		refresh: make(chan struct{}),
		done:    make(chan struct{}),
		base:    watcher,
		path:    absPath,
		parent:  absParentDir,
	}
	go fsdn.eventLoop()
	return fsdn, nil
}

func (fsdn *fsdirNotifier) Events() <-chan Event {
	return fsdn.events
}

func (fsdn *fsdirNotifier) Refresh() {
	fsdn.refresh <- struct{}{}
}

func (fsdn *fsdirNotifier) eventLoop() {
	defer fsdn.base.Close()
	defer close(fsdn.events)
	defer close(fsdn.refresh)
	if !listDirElements(fsdn.log, fsdn.path, fsdn.events, fsdn.done) {
		return
	}
	for fsdn.readAndProcessEvent() {
	}
}

func (fsdn *fsdirNotifier) readAndProcessEvent() bool {
	select {
	case <-fsdn.done:
		return false
	default:
	}
	select {
	case <-fsdn.done:
		return false
	case <-fsdn.refresh:
		listDirElements(fsdn.log, fsdn.path, fsdn.events, fsdn.done)
	case err, ok := <-fsdn.base.Errors:
		if !ok {
			return false
		}
		select {
		case fsdn.events <- Event{Op: Error, Err: err}:
		case <-fsdn.done:
			return false
		}
	case ev, ok := <-fsdn.base.Events:
		if !ok {
			return false
		}
		if !fsdn.processEvent(&ev) {
			return false
		}
	}
	return true
}

func (fsdn *fsdirNotifier) processEvent(ev *fsnotify.Event) bool {
	if strings.HasPrefix(ev.Name, fsdn.path) {
		if len(ev.Name) == len(fsdn.path) {
			return fsdn.processDirEvent(ev)
		}
		return fsdn.processFileEvent(ev)
	}
	return true
}

const deleteFsOps = fsnotify.Remove | fsnotify.Rename
const updateFsOps = fsnotify.Create | fsnotify.Write

func (fsdn *fsdirNotifier) processDirEvent(ev *fsnotify.Event) bool {
	if ev.Op&deleteFsOps != 0 {
		fsdn.log.Debug().Str("name", fsdn.path).Msg("Directory removed")
		fsdn.base.Remove(fsdn.path)
		select {
		case fsdn.events <- Event{Op: Destroy}:
		case <-fsdn.done:
			return false
		}
		return true
	}
	if ev.Op&fsnotify.Create != 0 {
		err := fsdn.base.Add(fsdn.path)
		if err != nil {
			fsdn.log.IfErr(err).Str("name", fsdn.path).Msg("Unable to add directory")
			select {
			case fsdn.events <- Event{Op: Error, Err: err}:
			case <-fsdn.done:
				return false
			}
		}
		fsdn.log.Debug().Str("name", fsdn.path).Msg("Directory added")
		return listDirElements(fsdn.log, fsdn.path, fsdn.events, fsdn.done)
	}
	return true
}

func (fsdn *fsdirNotifier) processFileEvent(ev *fsnotify.Event) bool {
	if ev.Op&deleteFsOps != 0 {
		fsdn.log.Trace().Str("name", ev.Name).Uint("op", uint64(ev.Op)).Msg("File deleted")
		select {
		case fsdn.events <- Event{Op: Delete, Name: filepath.Base(ev.Name)}:
		case <-fsdn.done:
			return false
		}
		return true
	}
	if ev.Op&updateFsOps != 0 {
		if fi, err := os.Lstat(ev.Name); err != nil || !fi.Mode().IsRegular() {
			return true
		}
		fsdn.log.Trace().Str("name", ev.Name).Uint("op", uint64(ev.Op)).Msg("File updated")
		select {
		case fsdn.events <- Event{Op: Update, Name: filepath.Base(ev.Name)}:
		case <-fsdn.done:
			return false
		}
	}
	return true
}

func (fsdn *fsdirNotifier) Close() {
	close(fsdn.done)
}

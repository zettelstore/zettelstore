//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package notifydir

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/fsnotify/fsnotify"

	"zettelstore.de/z/domain/id"
)

var validFileName = regexp.MustCompile(`^(\d{14}).*(\.(.+))$`)

func matchValidFileName(name string) []string {
	return validFileName.FindStringSubmatch(name)
}

type fileStatus int

const (
	fileStatusNone fileStatus = iota
	fileStatusReloadStart
	fileStatusReloadEnd
	fileStatusError
	fileStatusUpdate
	fileStatusDelete
)

type fileEvent struct {
	status fileStatus
	path   string // Full file path
	zid    id.Zid
	ext    string // File extension
	err    error  // Error if Status == fileStatusError
}

type sendResult int

const (
	sendDone sendResult = iota
	sendReload
	sendExit
)

func watchDirectory(directory string, events chan<- *fileEvent, tick <-chan struct{}) {
	defer close(events)

	var watcher *fsnotify.Watcher
	defer func() {
		if watcher != nil {
			watcher.Close()
		}
	}()

	sendEvent := func(ev *fileEvent) sendResult {
		select {
		case events <- ev:
		case _, ok := <-tick:
			if ok {
				return sendReload
			}
			return sendExit
		}
		return sendDone
	}

	sendError := func(err error) sendResult {
		return sendEvent(&fileEvent{status: fileStatusError, err: err})
	}

	sendFileEvent := func(status fileStatus, path string, match []string) sendResult {
		zid, err := id.Parse(match[1])
		if err != nil {
			return sendDone
		}
		event := &fileEvent{
			status: status,
			path:   path,
			zid:    zid,
			ext:    match[3],
		}
		return sendEvent(event)
	}

	reloadStartEvent := &fileEvent{status: fileStatusReloadStart}
	reloadEndEvent := &fileEvent{status: fileStatusReloadEnd}
	reloadFiles := func() bool {
		entries, err := os.ReadDir(directory)
		if err != nil {
			if res := sendError(err); res != sendDone {
				return res == sendReload
			}
			return true
		}

		if res := sendEvent(reloadStartEvent); res != sendDone {
			return res == sendReload
		}

		if watcher != nil {
			watcher.Close()
		}
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			if res := sendError(err); res != sendDone {
				return res == sendReload
			}
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if info, err1 := entry.Info(); err1 != nil || !info.Mode().IsRegular() {
				continue
			}
			name := entry.Name()
			match := matchValidFileName(name)
			if len(match) > 0 {
				path := filepath.Join(directory, name)
				if res := sendFileEvent(fileStatusUpdate, path, match); res != sendDone {
					return res == sendReload
				}
			}
		}

		if watcher != nil {
			err = watcher.Add(directory)
			if err != nil {
				if res := sendError(err); res != sendDone {
					return res == sendReload
				}
			}
		}
		if res := sendEvent(reloadEndEvent); res != sendDone {
			return res == sendReload
		}
		return true
	}

	handleEvents := func() bool {
		const createOps = fsnotify.Create | fsnotify.Write
		const deleteOps = fsnotify.Remove | fsnotify.Rename

		for {
			select {
			case wevent, ok := <-watcher.Events:
				if !ok {
					return false
				}
				path := filepath.Clean(wevent.Name)
				match := matchValidFileName(filepath.Base(path))
				if len(match) == 0 {
					continue
				}
				if wevent.Op&createOps != 0 {
					if fi, err := os.Lstat(path); err != nil || !fi.Mode().IsRegular() {
						continue
					}
					if res := sendFileEvent(
						fileStatusUpdate, path, match); res != sendDone {
						return res == sendReload
					}
				}
				if wevent.Op&deleteOps != 0 {
					if res := sendFileEvent(
						fileStatusDelete, path, match); res != sendDone {
						return res == sendReload
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return false
				}
				if res := sendError(err); res != sendDone {
					return res == sendReload
				}
			case _, ok := <-tick:
				return ok
			}
		}
	}

	for {
		if !reloadFiles() {
			return
		}
		if watcher == nil {
			if _, ok := <-tick; !ok {
				return
			}
		} else {
			if !handleEvents() {
				return
			}
		}
	}
}

func sendCollectedEvents(out chan<- *fileEvent, events []*fileEvent) {
	for _, ev := range events {
		if ev.status != fileStatusNone {
			out <- ev
		}
	}
}

func addEvent(events []*fileEvent, ev *fileEvent) []*fileEvent {
	switch ev.status {
	case fileStatusNone:
		return events
	case fileStatusReloadStart:
		events = events[0:0]
	case fileStatusUpdate, fileStatusDelete:
		if len(events) > 0 && mergeEvents(events, ev) {
			return events
		}
	}
	return append(events, ev)
}

func mergeEvents(events []*fileEvent, ev *fileEvent) bool {
	for i := len(events) - 1; i >= 0; i-- {
		oev := events[i]
		switch oev.status {
		case fileStatusReloadStart, fileStatusReloadEnd:
			return false
		case fileStatusUpdate, fileStatusDelete:
			if ev.path == oev.path {
				if ev.status == oev.status {
					return true
				}
				oev.status = fileStatusNone
				return false
			}
		}
	}
	return false
}

func collectEvents(out chan<- *fileEvent, in <-chan *fileEvent) {
	defer close(out)

	var sendTime time.Time
	sendTimeSet := false
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	events := make([]*fileEvent, 0, 32)
	buffer := false
	for {
		select {
		case ev, ok := <-in:
			if !ok {
				sendCollectedEvents(out, events)
				return
			}
			if ev.status == fileStatusReloadStart {
				buffer = false
				events = events[0:0]
			}
			if buffer {
				if !sendTimeSet {
					sendTime = time.Now().Add(1500 * time.Millisecond)
					sendTimeSet = true
				}
				events = addEvent(events, ev)
				if len(events) > 1024 {
					sendCollectedEvents(out, events)
					events = events[0:0]
					sendTimeSet = false
				}
				continue
			}
			out <- ev
			if ev.status == fileStatusReloadEnd {
				buffer = true
			}
		case now := <-ticker.C:
			if sendTimeSet && now.After(sendTime) {
				sendCollectedEvents(out, events)
				events = events[0:0]
				sendTimeSet = false
			}
		}
	}
}

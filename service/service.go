//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package service provides the main internal service.
package service

import (
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
)

// Service is the main internal service.
type Service struct {
	started   bool
	mx        sync.RWMutex
	commands  chan workerCommand
	interrupt chan os.Signal
	observer  []chan void
}

var Main = create()

// create and start a new service.
func create() *Service {
	srv := Service{
		commands:  make(chan workerCommand),
		interrupt: make(chan os.Signal, 5),
	}
	signal.Notify(srv.interrupt, os.Interrupt, syscall.SIGTERM)
	srv.start()
	return &srv
}

// started returns the started state of the service.
func (srv *Service) isStarted() bool {
	srv.mx.RLock()
	started := srv.started
	srv.mx.RUnlock()
	return started
}

// start a worker.
func (srv *Service) start() {
	if srv.isStarted() {
		log.Println("Restart internal service")
	}
	go srv.worker()
	srv.mx.Lock()
	srv.started = true
	srv.mx.Unlock()
}

// workerCommand encapsulates a command sent to the worker.
type workerCommand interface {
	run(srv *Service)
}

// send a command to the service.
func (srv *Service) send(cmd workerCommand) {
	srv.commands <- cmd
}

// worker is the background activity.
func (srv *Service) worker() {
	// Something may panic. Ensure a running worker.
	defer func() {
		if r := recover(); r != nil {
			log.Println("recovered from:", r)
			debug.PrintStack()
			srv.start()
		}
	}()

	timerDuration := 15 * time.Second
	timer := time.NewTimer(timerDuration)
	for {
		select {
		case cmd, ok := <-srv.commands:
			if !ok {
				break
			}
			cmd.run(srv)
		case <-srv.interrupt:
			cmd := stopCommand{}
			cmd.execute(srv)
		case _, ok := <-timer.C:
			if !ok {
				return
			}
			timer.Reset(timerDuration)
		}
	}
}

type void struct{}

// Stop the service. Waits for all concurrent activity to stop.
func (srv *Service) Stop() {
	if !srv.isStarted() {
		return
	}

	// Send the stop command
	rc := make(chan stopResult)
	srv.send(&stopCommand{rc})
	<-rc
	close(rc)

	srv.mx.Lock()
	srv.started = false
	close(srv.commands)
	srv.mx.Unlock()
}

type (
	stopCommand struct {
		result chan<- stopResult
	}
	stopResult = void
)

func (cmd *stopCommand) run(srv *Service) {
	cmd.execute(srv)
	cmd.result <- void{}
}

func (cmd *stopCommand) execute(srv *Service) {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	for _, ob := range srv.observer {
		ob <- void{}
		close(ob)
	}
	srv.observer = nil
}

// Notifier returns a channel where the caller gets notified to stop.
func (srv *Service) Notifier() <-chan void {
	srv.mx.Lock()
	result := make(chan void, 1)
	srv.observer = append(srv.observer, result)
	srv.mx.Unlock()
	return result
}

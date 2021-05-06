//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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
	observer  []chan Unit
}

// Main references the main service.
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
		srv.doLog("Trying to restart main service")
		return
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
			srv.doLogRecover("Main", r)
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
			srv.doLog("Shut down Zettelstore ...")
			cmd := shutdownCommand{}
			cmd.execute(srv)
		case _, ok := <-timer.C:
			if !ok {
				return
			}
			timer.Reset(timerDuration)
		}
	}
}

// Unit is a type with just one value.
type Unit struct{}

// --- Shutdown operation ----------------------------------------------------

// Shutdown the service. Waits for all concurrent activity to stop.
func (srv *Service) Shutdown() {
	if !srv.isStarted() {
		return
	}

	// Send the stop command
	rc := make(chan shutdownResult)
	srv.send(&shutdownCommand{rc})
	<-rc
	close(rc)

	srv.mx.Lock()
	srv.started = false
	close(srv.commands)
	srv.mx.Unlock()
}

type (
	shutdownCommand struct {
		result chan<- shutdownResult
	}
	shutdownResult = Unit
)

func (cmd *shutdownCommand) run(srv *Service) {
	cmd.execute(srv)
	cmd.result <- Unit{}
}

func (cmd *shutdownCommand) execute(srv *Service) {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	for _, ob := range srv.observer {
		ob <- Unit{}
		close(ob)
	}
	srv.observer = nil
}

// ShutdownChan is a channel used to signal a system shutdown.
type ShutdownChan <-chan Unit

// ShutdownNotifier returns a channel where the caller gets notified to stop.
func (srv *Service) ShutdownNotifier() ShutdownChan {
	srv.mx.Lock()
	result := make(chan Unit, 1)
	srv.observer = append(srv.observer, result)
	srv.mx.Unlock()
	return result
}

// IgnoreShutdown marks the given channel as to be ignored on shutdown.
func (srv *Service) IgnoreShutdown(ob ShutdownChan) {
	srv.mx.Lock()
	lastIndex := len(srv.observer) - 1
	for i := 0; i <= lastIndex; i++ {
		if srv.observer[i] != ob {
			continue
		}
		close(srv.observer[i])
		srv.observer[i] = srv.observer[lastIndex]
		srv.observer[lastIndex] = nil
		srv.observer = srv.observer[:lastIndex]
		break
	}
	srv.mx.Unlock()
}

// --- Log operation ---------------------------------------------------------

// Log some activity.
func (srv *Service) Log(args ...interface{}) {
	srv.doLog(args...)
}
func (srv *Service) doLog(args ...interface{}) {
	log.Println(args...)
}

// LogRecover outputs some information about the previous panic.
func (srv *Service) LogRecover(name string, recoverInfo interface{}) {
	srv.doLogRecover(name, recoverInfo)
}
func (srv *Service) doLogRecover(name string, recoverInfo interface{}) {
	srv.Log(name, "recovered from:", recoverInfo)
	debug.PrintStack()
}

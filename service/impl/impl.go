//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the main internal service implementation.
package impl

import (
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"zettelstore.de/z/service"
)

// service is the main internal service.
type myService struct {
	started   bool
	wg        sync.WaitGroup
	mx        sync.RWMutex
	commands  chan workerCommand
	interrupt chan os.Signal
	observer  []chan service.Unit
	debug     bool

	web webService
}

// create and start a new service.
func init() {
	service.Main = createAndStart()
}

// create and start a new service.
func createAndStart() service.Service {
	srv := &myService{
		started:   true,
		commands:  make(chan workerCommand),
		interrupt: make(chan os.Signal, 5),
	}
	srv.wg.Add(1)
	signal.Notify(srv.interrupt, os.Interrupt, syscall.SIGTERM)
	go srv.worker()
	return srv
}

// workerCommand encapsulates a command sent to the worker.
type workerCommand interface {
	run(srv *myService)
}

// send a command to the service.
// func (srv *myService) send(cmd workerCommand) {
// 	srv.commands <- cmd
// }

// worker is the background activity.
func (srv *myService) worker() {
	// Something may panic. Ensure a running worker.
	defer func() {
		if r := recover(); r != nil {
			srv.doLogRecover("Main", r)
			go srv.worker()
		}
	}()

	timerDuration := 15 * time.Second
	timer := time.NewTimer(timerDuration)
loop:
	for {
		select {
		case cmd := <-srv.commands:
			cmd.run(srv)
		case <-timer.C:
			timer.Reset(timerDuration)
		case sig := <-srv.interrupt:
			srv.doLog("Shut down Zettelstore:", sig, "...")
			srv.shutdown()
			break loop
		}
	}
	srv.wg.Done()
}

func (srv *myService) shutdown() {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	for _, ob := range srv.observer {
		ob <- service.Unit{}
		close(ob)
	}
	srv.observer = nil
	srv.started = false
	if srv.web.srvw != nil {
		srv.web.srvw.Stop()
	}
}

func (srv *myService) WaitForShutdown() {
	srv.wg.Wait()
}

func (srv *myService) SetDebug(enable bool) bool {
	srv.mx.Lock()
	prevDebug := srv.debug
	srv.debug = enable
	srv.mx.Unlock()
	return prevDebug
}

// --- Shutdown operation ----------------------------------------------------

// Shutdown the service. Waits for all concurrent activity to stop.
func (srv *myService) Shutdown() {
	srv.interrupt <- &shutdownSignal{}
}

type shutdownSignal struct{}

func (s *shutdownSignal) String() string { return "shutdown" }
func (s *shutdownSignal) Signal()        { /* Just a signal */ }

// ShutdownNotifier returns a channel where the caller gets notified to stop.
func (srv *myService) ShutdownNotifier() service.ShutdownChan {
	srv.mx.Lock()
	result := make(chan service.Unit, 1)
	srv.observer = append(srv.observer, result)
	srv.mx.Unlock()
	return result
}

// IgnoreShutdown marks the given channel as to be ignored on shutdown.
func (srv *myService) IgnoreShutdown(ob service.ShutdownChan) {
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
func (srv *myService) Log(args ...interface{}) {
	srv.doLog(args...)
}
func (srv *myService) doLog(args ...interface{}) {
	log.Println(args...)
}

// LogRecover outputs some information about the previous panic.
func (srv *myService) LogRecover(name string, recoverInfo interface{}) {
	srv.doLogRecover(name, recoverInfo)
}
func (srv *myService) doLogRecover(name string, recoverInfo interface{}) {
	srv.Log(name, "recovered from:", recoverInfo)
	debug.PrintStack()
}

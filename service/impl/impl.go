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
	"fmt"
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

	core     coreSub
	auth     authSub
	place    placeSub
	web      webSub
	subs     map[service.Subservice]subServceDescr
	subNames map[string]subService
}

type subServceDescr struct {
	sub  subService
	name string
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
	srv.subs = map[service.Subservice]subServceDescr{
		service.SubCore:  {&srv.core, "core"},
		service.SubAuth:  {&srv.auth, "auth"},
		service.SubPlace: {&srv.place, "place"},
		service.SubWeb:   {&srv.web, "web"},
	}
	srv.subNames = make(map[string]subService, len(srv.subs))
	for key, subDescr := range srv.subs {
		if sub, ok := srv.subNames[subDescr.name]; ok {
			panic(fmt.Sprintf("Key %q already given for sub-service %v", key, sub))
		}
		srv.subNames[subDescr.name] = subDescr.sub
		subDescr.sub.Initialize()
	}
	return srv
}

func (srv *myService) Start(headline bool) {
	for _, sub := range srv.subs {
		sub.sub.Freeze()
	}
	srv.wg.Add(1)
	signal.Notify(srv.interrupt, os.Interrupt, syscall.SIGTERM)
	go srv.worker()
	srv.StartSub(service.SubCore)

	srv.mx.Lock()
	defer srv.mx.Unlock()
	if headline {
		srv.doLog(fmt.Sprintf(
			"%v %v (%v@%v/%v)",
			srv.core.GetConfig(service.CoreProgname),
			srv.core.GetConfig(service.CoreVersion),
			srv.core.GetConfig(service.CoreGoVersion),
			srv.core.GetConfig(service.CoreGoOS),
			srv.core.GetConfig(service.CoreGoArch),
		))
		srv.doLog("Licensed under the latest version of the EUPL (European Union Public License)")
		if srv.auth.GetConfig(service.AuthReadonly).(bool) {
			srv.doLog("Read-only mode")
		}
	}
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
		srv.web.Stop(srv)
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
	srv.mx.Lock()
	defer srv.mx.Unlock()
	srv.doLog(args...)
}
func (srv *myService) doLog(args ...interface{}) {
	log.Println(args...)
}

// LogRecover outputs some information about the previous panic.
func (srv *myService) LogRecover(name string, recoverInfo interface{}) bool {
	// srv.mx.Lock()
	// defer srv.mx.Unlock()
	return srv.doLogRecover(name, recoverInfo)
}
func (srv *myService) doLogRecover(name string, recoverInfo interface{}) bool {
	srv.Log(name, "recovered from:", recoverInfo)
	debug.PrintStack()
	return true
}

// --- Sub-service handling --------------------------------------------------

func (srv *myService) getSubservice(subsrv service.Subservice) subService {
	if sub, ok := srv.subs[subsrv]; ok {
		return sub.sub
	}
	return nil
}
func (srv *myService) getSubserviceByName(name string) subService {
	if sub, ok := srv.subNames[name]; ok {
		return sub
	}
	return nil
}

func (srv *myService) SetConfig(subsrv service.Subservice, key, value string) bool {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	if sub := srv.getSubservice(subsrv); sub != nil {
		return sub.SetConfig(key, value)
	}
	return false
}

func (srv *myService) GetConfig(subsrv service.Subservice, key string) interface{} {
	srv.mx.RLock()
	defer srv.mx.RUnlock()
	if sub := srv.getSubservice(subsrv); sub != nil {
		return sub.GetConfig(key)
	}
	return nil
}

func (srv *myService) GetConfigList(subsrv service.Subservice) []service.KeyDescrValue {
	srv.mx.RLock()
	defer srv.mx.RUnlock()
	if sub := srv.getSubservice(subsrv); sub != nil {
		return sub.GetConfigList(false)
	}
	return nil
}

func (srv *myService) StartSub(subsrv service.Subservice) error {
	srv.mx.Lock()
	defer srv.mx.Unlock()
	if sub := srv.getSubservice(subsrv); sub != nil {
		if err := sub.Start(srv); err != nil {
			return err
		}
		sub.SwitchNextToCur()
	}
	return nil
}
func (srv *myService) StopSub(subsrv service.Subservice) error {
	return nil
}

type subService interface {
	// Initialize the data for the sub-service.
	Initialize()

	// SetConfig stores a configuration value.
	SetConfig(key, value string) bool

	// GetConfig returns the current configuration value.
	GetConfig(key string) interface{}

	// GetNextConfig returns the next configuration value.
	GetNextConfig(key string) interface{}

	// GetConfigList returns a sorted list of configuration data.
	GetConfigList(all bool) []service.KeyDescrValue

	// Freeze disallows to change some fixed configuration values.
	Freeze()

	// Start start the given sub-service.
	Start(srv *myService) error

	// SwitchNextToCur moves next config data to current.
	SwitchNextToCur()

	// Stop stop the given sub-service.
	Stop(srv *myService) error
}

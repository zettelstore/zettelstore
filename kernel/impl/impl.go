//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the kernel implementation.
package impl

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"

	"zettelstore.de/z/kernel"
)

// myKernel is the main internal kernel.
type myKernel struct {
	// started   bool
	wg        sync.WaitGroup
	mx        sync.RWMutex
	interrupt chan os.Signal
	debug     bool

	core coreService
	cfg  configService
	auth authService
	box  boxService
	web  webService

	srvs     map[kernel.Service]serviceDescr
	srvNames map[string]serviceData
	depStart serviceDependency
	depStop  serviceDependency // reverse of depStart
}

type serviceDescr struct {
	srv  service
	name string
}
type serviceData struct {
	srv    service
	srvnum kernel.Service
}
type serviceDependency map[kernel.Service][]kernel.Service

// create and start a new kernel.
func init() {
	kernel.Main = createAndStart()
}

// create and start a new kernel.
func createAndStart() kernel.Kernel {
	kern := &myKernel{
		interrupt: make(chan os.Signal, 5),
	}
	kern.srvs = map[kernel.Service]serviceDescr{
		kernel.CoreService:   {&kern.core, "core"},
		kernel.ConfigService: {&kern.cfg, "config"},
		kernel.AuthService:   {&kern.auth, "auth"},
		kernel.BoxService:    {&kern.box, "box"},
		kernel.WebService:    {&kern.web, "web"},
	}
	kern.srvNames = make(map[string]serviceData, len(kern.srvs))
	for key, srvD := range kern.srvs {
		if _, ok := kern.srvNames[srvD.name]; ok {
			panic(fmt.Sprintf("Key %q already given for service %v", key, srvD.name))
		}
		kern.srvNames[srvD.name] = serviceData{srvD.srv, key}
		srvD.srv.Initialize()
	}
	kern.depStart = serviceDependency{
		kernel.CoreService:   nil,
		kernel.ConfigService: {kernel.CoreService},
		kernel.AuthService:   {kernel.CoreService},
		kernel.BoxService:    {kernel.CoreService, kernel.ConfigService, kernel.AuthService},
		kernel.WebService:    {kernel.ConfigService, kernel.AuthService, kernel.BoxService},
	}
	kern.depStop = make(serviceDependency, len(kern.depStart))
	for srv, deps := range kern.depStart {
		for _, dep := range deps {
			kern.depStop[dep] = append(kern.depStop[dep], srv)
		}
	}
	return kern
}

func (kern *myKernel) Start(headline bool, lineServer bool) {
	for _, srvD := range kern.srvs {
		srvD.srv.Freeze()
	}
	kern.wg.Add(1)
	signal.Notify(kern.interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		// Wait for interrupt.
		sig := <-kern.interrupt
		if strSig := sig.String(); strSig != "" {
			kern.doLog("Shut down Zettelstore:", strSig)
		}
		kern.shutdown()
		kern.wg.Done()
	}()

	kern.StartService(kernel.CoreService)
	if headline {
		kern.doLog(fmt.Sprintf(
			"%v %v (%v@%v/%v)",
			kern.core.GetConfig(kernel.CoreProgname),
			kern.core.GetConfig(kernel.CoreVersion),
			kern.core.GetConfig(kernel.CoreGoVersion),
			kern.core.GetConfig(kernel.CoreGoOS),
			kern.core.GetConfig(kernel.CoreGoArch),
		))
		kern.doLog("Licensed under the latest version of the EUPL (European Union Public License)")
		if kern.auth.GetConfig(kernel.AuthReadonly).(bool) {
			kern.doLog("Read-only mode")
		}
	}
	if lineServer {
		port := kern.core.GetNextConfig(kernel.CorePort).(int)
		if port > 0 {
			listenAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			startLineServer(kern, listenAddr)
		}
	}
}

func (kern *myKernel) shutdown() {
	kern.StopService(kernel.CoreService) // Will stop all other services.
}

func (kern *myKernel) WaitForShutdown() {
	kern.wg.Wait()
}

func (kern *myKernel) SetDebug(enable bool) bool {
	kern.mx.Lock()
	prevDebug := kern.debug
	kern.debug = enable
	kern.mx.Unlock()
	return prevDebug
}

// --- Shutdown operation ----------------------------------------------------

// Shutdown the service. Waits for all concurrent activity to stop.
func (kern *myKernel) Shutdown(silent bool) {
	kern.interrupt <- &shutdownSignal{silent: silent}
}

type shutdownSignal struct{ silent bool }

func (s *shutdownSignal) String() string {
	if s.silent {
		return ""
	}
	return "shutdown"
}
func (s *shutdownSignal) Signal() { /* Just a signal */ }

// --- Log operation ---------------------------------------------------------

// Log some activity.
func (kern *myKernel) Log(args ...interface{}) {
	kern.mx.Lock()
	defer kern.mx.Unlock()
	kern.doLog(args...)
}
func (kern *myKernel) doLog(args ...interface{}) {
	log.Println(args...)
}

// LogRecover outputs some information about the previous panic.
func (kern *myKernel) LogRecover(name string, recoverInfo interface{}) bool {
	return kern.doLogRecover(name, recoverInfo)
}
func (kern *myKernel) doLogRecover(name string, recoverInfo interface{}) bool {
	kern.Log(name, "recovered from:", recoverInfo)
	stack := debug.Stack()
	os.Stderr.Write(stack)
	kern.core.updateRecoverInfo(name, recoverInfo, stack)
	return true
}

// --- Service handling --------------------------------------------------

func (kern *myKernel) SetConfig(srvnum kernel.Service, key, value string) bool {
	kern.mx.Lock()
	defer kern.mx.Unlock()
	if srvD, ok := kern.srvs[srvnum]; ok {
		return srvD.srv.SetConfig(key, value)
	}
	return false
}

func (kern *myKernel) GetConfig(srvnum kernel.Service, key string) interface{} {
	kern.mx.RLock()
	defer kern.mx.RUnlock()
	if srvD, ok := kern.srvs[srvnum]; ok {
		return srvD.srv.GetConfig(key)
	}
	return nil
}

func (kern *myKernel) GetConfigList(srvnum kernel.Service) []kernel.KeyDescrValue {
	kern.mx.RLock()
	defer kern.mx.RUnlock()
	if srvD, ok := kern.srvs[srvnum]; ok {
		return srvD.srv.GetConfigList(false)
	}
	return nil
}
func (kern *myKernel) GetServiceStatistics(srvnum kernel.Service) []kernel.KeyValue {
	kern.mx.RLock()
	defer kern.mx.RUnlock()
	if srvD, ok := kern.srvs[srvnum]; ok {
		return srvD.srv.GetStatistics()
	}
	return nil
}

func (kern *myKernel) StartService(srvnum kernel.Service) error {
	kern.mx.RLock()
	defer kern.mx.RUnlock()
	return kern.doStartService(srvnum)
}
func (kern *myKernel) doStartService(srvnum kernel.Service) error {
	for _, srv := range kern.sortDependency(srvnum, kern.depStart, true) {
		if err := srv.Start(kern); err != nil {
			return err
		}
		srv.SwitchNextToCur()
	}
	return nil
}

func (kern *myKernel) RestartService(srvnum kernel.Service) error {
	kern.mx.RLock()
	defer kern.mx.RUnlock()
	return kern.doRestartService(srvnum)
}
func (kern *myKernel) doRestartService(srvnum kernel.Service) error {
	deps := kern.sortDependency(srvnum, kern.depStop, false)
	for _, srv := range deps {
		if err := srv.Stop(kern); err != nil {
			return err
		}
	}
	for i := len(deps) - 1; i >= 0; i-- {
		srv := deps[i]
		if err := srv.Start(kern); err != nil {
			return err
		}
		srv.SwitchNextToCur()
	}
	return nil
}

func (kern *myKernel) StopService(srvnum kernel.Service) error {
	kern.mx.RLock()
	defer kern.mx.RUnlock()
	return kern.doStopService(srvnum)
}
func (kern *myKernel) doStopService(srvnum kernel.Service) error {
	for _, srv := range kern.sortDependency(srvnum, kern.depStop, false) {
		if err := srv.Stop(kern); err != nil {
			return err
		}
	}
	return nil
}

func (kern *myKernel) sortDependency(
	srvnum kernel.Service,
	srvdeps serviceDependency,
	isStarted bool,
) []service {
	srvD, ok := kern.srvs[srvnum]
	if !ok {
		return nil
	}
	if srvD.srv.IsStarted() == isStarted {
		return nil
	}
	deps := srvdeps[srvnum]
	found := make(map[service]bool, 4)
	result := make([]service, 0, 4)
	for _, dep := range deps {
		srvDeps := kern.sortDependency(dep, srvdeps, isStarted)
		for _, depSrv := range srvDeps {
			if !found[depSrv] {
				result = append(result, depSrv)
				found[depSrv] = true
			}
		}
	}
	return append(result, srvD.srv)
}
func (kern *myKernel) DumpIndex(w io.Writer) {
	kern.box.DumpIndex(w)
}

type service interface {
	// Initialize the data for the service.
	Initialize()

	// ConfigDescriptions returns a sorted list of configuration descriptions.
	ConfigDescriptions() []serviceConfigDescription

	// SetConfig stores a configuration value.
	SetConfig(key, value string) bool

	// GetConfig returns the current configuration value.
	GetConfig(key string) interface{}

	// GetNextConfig returns the next configuration value.
	GetNextConfig(key string) interface{}

	// GetConfigList returns a sorted list of current configuration data.
	GetConfigList(all bool) []kernel.KeyDescrValue

	// GetNextConfigList returns a sorted list of next configuration data.
	GetNextConfigList() []kernel.KeyDescrValue

	// GetStatistics returns a key/value list of statistical data.
	GetStatistics() []kernel.KeyValue

	// Freeze disallows to change some fixed configuration values.
	Freeze()

	// Start the service.
	Start(*myKernel) error

	// SwitchNextToCur moves next config data to current.
	SwitchNextToCur()

	// IsStarted returns true if the service was started successfully.
	IsStarted() bool

	// Stop the service.
	Stop(*myKernel) error
}

type serviceConfigDescription struct{ Key, Descr string }

func (kern *myKernel) SetCreators(
	createAuthManager kernel.CreateAuthManagerFunc,
	createBoxManager kernel.CreateBoxManagerFunc,
	setupWebServer kernel.SetupWebServerFunc,
) {
	kern.auth.createManager = createAuthManager
	kern.box.createManager = createBoxManager
	kern.web.setupServer = setupWebServer
}

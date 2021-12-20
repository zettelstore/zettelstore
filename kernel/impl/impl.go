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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
)

// myKernel is the main internal kernel.
type myKernel struct {
	logger    *logger.Logger
	wg        sync.WaitGroup
	mx        sync.RWMutex
	interrupt chan os.Signal

	profileName string
	fileName    string
	profileFile *os.File
	profile     *pprof.Profile

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
	srv      service
	name     string
	logLevel logger.Level
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
	lw := newKernelLogWriter()
	kern := &myKernel{
		logger:    logger.New(lw, ""),
		interrupt: make(chan os.Signal, 5),
	}
	kern.srvs = map[kernel.Service]serviceDescr{
		kernel.CoreService:   {&kern.core, "core", logger.InfoLevel},
		kernel.ConfigService: {&kern.cfg, "config", logger.InfoLevel},
		kernel.AuthService:   {&kern.auth, "auth", logger.InfoLevel},
		kernel.BoxService:    {&kern.box, "box", logger.InfoLevel},
		kernel.WebService:    {&kern.web, "web", logger.InfoLevel},
	}
	kern.srvNames = make(map[string]serviceData, len(kern.srvs))
	for key, srvD := range kern.srvs {
		if _, ok := kern.srvNames[srvD.name]; ok {
			kern.logger.Panic().Str("service", srvD.name).Msg("Service data already set")
		}
		kern.srvNames[srvD.name] = serviceData{srvD.srv, key}
		l := logger.New(lw, strings.ToUpper(srvD.name)).SetLevel(srvD.logLevel)
		srvD.srv.Initialize(l)
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

func (kern *myKernel) Start(headline, lineServer bool) {
	for _, srvD := range kern.srvs {
		srvD.srv.Freeze()
	}
	kern.wg.Add(1)
	signal.Notify(kern.interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		// Wait for interrupt.
		sig := <-kern.interrupt
		if strSig := sig.String(); strSig != "" {
			kern.logger.Info().Str("signal", strSig).Msg("Shut down Zettelstore")
		}
		kern.doShutdown()
		kern.wg.Done()
	}()

	kern.StartService(kernel.CoreService)
	if headline {
		logger := kern.logger
		logger.Mandatory().Msg(fmt.Sprintf(
			"%v %v (%v@%v/%v)",
			kern.core.GetConfig(kernel.CoreProgname),
			kern.core.GetConfig(kernel.CoreVersion),
			kern.core.GetConfig(kernel.CoreGoVersion),
			kern.core.GetConfig(kernel.CoreGoOS),
			kern.core.GetConfig(kernel.CoreGoArch),
		))
		logger.Mandatory().Msg("Licensed under the latest version of the EUPL (European Union Public License)")
		if kern.core.GetConfig(kernel.CoreDebug).(bool) {
			logger.Warn().Msg("----------------------------------------")
			logger.Warn().Msg("DEBUG MODE, DO NO USE THIS IN PRODUCTION")
			logger.Warn().Msg("----------------------------------------")
		}
		if kern.auth.GetConfig(kernel.AuthReadonly).(bool) {
			logger.Info().Msg("Read-only mode")
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

func (kern *myKernel) doShutdown() {
	kern.StopService(kernel.CoreService) // Will stop all other services.
}

func (kern *myKernel) WaitForShutdown() {
	kern.wg.Wait()
	kern.doStopProfiling()
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
func (*shutdownSignal) Signal() { /* Just a signal */ }

// --- Log operation ---------------------------------------------------------

func (kern *myKernel) GetKernelLogger() *logger.Logger {
	return kern.logger
}

func (kern *myKernel) SetGlobalLogLevel(level logger.Level) {
	if level.IsValid() {
		kern.mx.RLock()
		kern.logger.SetLevel(level)
		for _, srvD := range kern.srvs {
			srvD.srv.GetLogger().SetLevel(level)
		}
		kern.mx.RUnlock()
	}
}

// LogRecover outputs some information about the previous panic.
func (kern *myKernel) LogRecover(name string, recoverInfo interface{}) bool {
	return kern.doLogRecover(name, recoverInfo)
}
func (kern *myKernel) doLogRecover(name string, recoverInfo interface{}) bool {
	stack := debug.Stack()
	kern.logger.Fatal().Str("recovered_from", fmt.Sprint(recoverInfo)).Bytes("stack", stack).Msg(name)
	kern.core.updateRecoverInfo(name, recoverInfo, stack)
	return true
}

// --- Profiling ---------------------------------------------------------

var errProfileInWork = errors.New("already profiling")
var errProfileNotFound = errors.New("profile not found")

func (kern *myKernel) StartProfiling(profileName, fileName string) error {
	kern.mx.Lock()
	defer kern.mx.Unlock()
	return kern.doStartProfiling(profileName, fileName)
}
func (kern *myKernel) doStartProfiling(profileName, fileName string) error {
	if kern.profileName != "" {
		return errProfileInWork
	}
	if profileName == kernel.ProfileCPU {
		f, err := os.Create(fileName)
		if err != nil {
			return err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			f.Close()
			return err
		}
		kern.profileName = profileName
		kern.fileName = fileName
		kern.profileFile = f
		return err
	}
	profile := pprof.Lookup(profileName)
	if profile == nil {
		return errProfileNotFound
	}
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	kern.profileName = profileName
	kern.fileName = fileName
	kern.profile = profile
	kern.profileFile = f
	runtime.GC() // get up-to-date statistics
	profile.WriteTo(f, 0)
	return nil
}

func (kern *myKernel) StopProfiling() error {
	kern.mx.Lock()
	defer kern.mx.Unlock()
	return kern.doStopProfiling()
}
func (kern *myKernel) doStopProfiling() error {
	if kern.profileName == "" {
		return nil // No profile started
	}
	if kern.profileName == kernel.ProfileCPU {
		pprof.StopCPUProfile()
	}
	err := kern.profileFile.Close()
	kern.profileName = ""
	kern.fileName = ""
	kern.profile = nil
	kern.profileFile = nil
	return err
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

func (kern *myKernel) GetLogger(srvnum kernel.Service) *logger.Logger {
	kern.mx.RLock()
	defer kern.mx.RUnlock()
	if srvD, ok := kern.srvs[srvnum]; ok {
		return srvD.srv.GetLogger()
	}
	return kern.GetKernelLogger()
}

func (kern *myKernel) SetLevel(srvnum kernel.Service, level logger.Level) {
	if level.IsValid() {
		kern.mx.RLock()
		if srvD, ok := kern.srvs[srvnum]; ok {
			srvD.srv.GetLogger().SetLevel(level)
		}
		kern.mx.RUnlock()
	}
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
	kern.mx.Lock()
	defer kern.mx.Unlock()
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
	Initialize(*logger.Logger)

	// Get service logger.
	GetLogger() *logger.Logger

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

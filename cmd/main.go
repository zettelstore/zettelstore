//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/auth/impl"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/compbox"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/web/server"
)

const strRunSimple = "run-simple"

func init() {
	RegisterCommand(Command{
		Name: "help",
		Func: func(*flag.FlagSet) (int, error) {
			fmt.Println("Available commands:")
			for _, name := range List() {
				fmt.Printf("- %q\n", name)
			}
			return 0, nil
		},
	})
	RegisterCommand(Command{
		Name:   "version",
		Func:   func(*flag.FlagSet) (int, error) { return 0, nil },
		Header: true,
	})
	RegisterCommand(Command{
		Name:       "run",
		Func:       runFunc,
		Boxes:      true,
		Header:     true,
		LineServer: true,
		SetFlags:   flgRun,
	})
	RegisterCommand(Command{
		Name:   strRunSimple,
		Func:   runFunc,
		Simple: true,
		Boxes:  true,
		Header: true,
		// LineServer: true,
		SetFlags: func(fs *flag.FlagSet) {
			// fs.Uint("a", 0, "port number kernel service (0=disable)")
			fs.String("d", "", "zettel directory")
		},
	})
	RegisterCommand(Command{
		Name: "file",
		Func: cmdFile,
		SetFlags: func(fs *flag.FlagSet) {
			fs.String("t", api.EncoderHTML.String(), "target output encoding")
		},
	})
	RegisterCommand(Command{
		Name: "password",
		Func: cmdPassword,
	})
}

func fetchStartupConfiguration(fs *flag.FlagSet) (cfg *meta.Meta) {
	if configFlag := fs.Lookup("c"); configFlag != nil {
		if filename := configFlag.Value.String(); filename != "" {
			content, err := readConfiguration(filename)
			return createConfiguration(content, err)
		}
	}
	content, err := searchAndReadConfiguration()
	return createConfiguration(content, err)
}

func createConfiguration(content []byte, err error) *meta.Meta {
	if err != nil {
		return meta.New(id.Invalid)
	}
	return meta.NewFromInput(id.Invalid, input.NewInput(content))
}

func readConfiguration(filename string) ([]byte, error) { return os.ReadFile(filename) }

func searchAndReadConfiguration() ([]byte, error) {
	for _, filename := range []string{"zettelstore.cfg", "zsconfig.txt", "zscfg.txt", "_zscfg"} {
		if content, err := readConfiguration(filename); err == nil {
			return content, nil
		}
	}
	return readConfiguration(".zscfg")
}

func getConfig(fs *flag.FlagSet) *meta.Meta {
	cfg := fetchStartupConfiguration(fs)
	fs.Visit(func(flg *flag.Flag) {
		switch flg.Name {
		case "p":
			if portStr, err := parsePort(flg.Value.String()); err == nil {
				cfg.Set(keyListenAddr, net.JoinHostPort("127.0.0.1", portStr))
			}
		case "a":
			if portStr, err := parsePort(flg.Value.String()); err == nil {
				cfg.Set(keyAdminPort, portStr)
			}
		case "d":
			val := flg.Value.String()
			if strings.HasPrefix(val, "/") {
				val = "dir://" + val
			} else {
				val = "dir:" + val
			}
			deleteConfiguredBoxes(cfg)
			cfg.Set(keyBoxOneURI, val)
		case "l":
			cfg.Set(keyLogLevel, flg.Value.String())
		case "debug":
			cfg.Set(keyDebug, flg.Value.String())
		case "r":
			cfg.Set(keyReadOnly, flg.Value.String())
		case "v":
			cfg.Set(keyVerbose, flg.Value.String())
		}
	})
	return cfg
}

func parsePort(s string) (string, error) {
	port, err := net.LookupPort("tcp", s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Wrong port specification: %q", s)
		return "", err
	}
	return strconv.Itoa(port), nil
}

func deleteConfiguredBoxes(cfg *meta.Meta) {
	for _, p := range cfg.PairsRest() {
		if key := p.Key; strings.HasPrefix(key, kernel.BoxURIs) {
			cfg.Delete(key)
		}
	}
}

const (
	keyAdminPort         = "admin-port"
	keyDebug             = "debug-mode"
	keyDefaultDirBoxType = "default-dir-box-type"
	keyInsecureCookie    = "insecure-cookie"
	keyListenAddr        = "listen-addr"
	keyLogLevel          = "log-level"
	keyMaxRequestSize    = "max-request-size"
	keyOwner             = "owner"
	keyPersistentCookie  = "persistent-cookie"
	keyBoxOneURI         = kernel.BoxURIs + "1"
	keyReadOnly          = "read-only-mode"
	keyTokenLifetimeHTML = "token-lifetime-html"
	keyTokenLifetimeAPI  = "token-lifetime-api"
	keyURLPrefix         = "url-prefix"
	keyVerbose           = "verbose-mode"
)

func setServiceConfig(cfg *meta.Meta) error {
	debugMode := cfg.GetBool(keyDebug)
	if debugMode && kernel.Main.GetKernelLogger().Level() > logger.DebugLevel {
		kernel.Main.SetGlobalLogLevel(logger.DebugLevel)
	}
	if strLevel, found := cfg.Get(keyLogLevel); found {
		if level := logger.ParseLevel(strLevel); level.IsValid() {
			kernel.Main.SetGlobalLogLevel(level)
		}
	}
	ok := setConfigValue(true, kernel.CoreService, kernel.CoreDebug, debugMode)
	ok = setConfigValue(ok, kernel.CoreService, kernel.CoreVerbose, cfg.GetBool(keyVerbose))
	if val, found := cfg.Get(keyAdminPort); found {
		ok = setConfigValue(ok, kernel.CoreService, kernel.CorePort, val)
	}

	ok = setConfigValue(ok, kernel.AuthService, kernel.AuthOwner, cfg.GetDefault(keyOwner, ""))
	ok = setConfigValue(ok, kernel.AuthService, kernel.AuthReadonly, cfg.GetBool(keyReadOnly))

	ok = setConfigValue(
		ok, kernel.BoxService, kernel.BoxDefaultDirType,
		cfg.GetDefault(keyDefaultDirBoxType, kernel.BoxDirTypeNotify))
	ok = setConfigValue(ok, kernel.BoxService, kernel.BoxURIs+"1", "dir:./zettel")
	format := kernel.BoxURIs + "%v"
	for i := 1; ; i++ {
		key := fmt.Sprintf(format, i)
		val, found := cfg.Get(key)
		if !found {
			break
		}
		ok = setConfigValue(ok, kernel.BoxService, key, val)
	}

	ok = setConfigValue(
		ok, kernel.WebService, kernel.WebListenAddress,
		cfg.GetDefault(keyListenAddr, "127.0.0.1:23123"))
	ok = setConfigValue(ok, kernel.WebService, kernel.WebURLPrefix, cfg.GetDefault(keyURLPrefix, "/"))
	ok = setConfigValue(ok, kernel.WebService, kernel.WebSecureCookie, !cfg.GetBool(keyInsecureCookie))
	ok = setConfigValue(ok, kernel.WebService, kernel.WebPersistentCookie, cfg.GetBool(keyPersistentCookie))
	if val, found := cfg.Get(keyMaxRequestSize); found {
		ok = setConfigValue(ok, kernel.WebService, kernel.WebMaxRequestSize, val)
	}
	ok = setConfigValue(
		ok, kernel.WebService, kernel.WebTokenLifetimeAPI, cfg.GetDefault(keyTokenLifetimeAPI, ""))
	ok = setConfigValue(
		ok, kernel.WebService, kernel.WebTokenLifetimeHTML, cfg.GetDefault(keyTokenLifetimeHTML, ""))

	if !ok {
		return errors.New("unable to set configuration")
	}
	return nil
}

func setConfigValue(ok bool, subsys kernel.Service, key string, val interface{}) bool {
	done := kernel.Main.SetConfig(subsys, key, fmt.Sprintf("%v", val))
	if !done {
		kernel.Main.GetKernelLogger().Error().Str(key, fmt.Sprint(val)).Msg("Unable to set configuration")
	}
	return ok && done
}

func executeCommand(name string, args ...string) int {
	command, ok := Get(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", name)
		return 1
	}
	fs := command.GetFlags()
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to parse flags: %v %v\n", name, args, err)
		return 1
	}
	cfg := getConfig(fs)
	if err := setServiceConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		return 2
	}

	kern := kernel.Main
	var createManager kernel.CreateBoxManagerFunc
	if command.Boxes {
		err := raiseFdLimit()
		if err != nil {
			logger := kern.GetKernelLogger()
			logger.IfErr(err).Msg("Raising some limitions did not work")
			logger.Error().Msg("Prepare to encounter errors. Most of them can be mitigated. See the manual for details")
			kern.SetConfig(kernel.BoxService, kernel.BoxDefaultDirType, kernel.BoxDirTypeSimple)
		}
		createManager = func(boxURIs []*url.URL, authManager auth.Manager, rtConfig config.Config) (box.Manager, error) {
			compbox.Setup(cfg)
			return manager.New(boxURIs, authManager, rtConfig)
		}
	} else {
		createManager = func([]*url.URL, auth.Manager, config.Config) (box.Manager, error) { return nil, nil }
	}

	kern.SetCreators(
		func(readonly bool, owner id.Zid) (auth.Manager, error) {
			return impl.New(readonly, owner, cfg.GetDefault("secret", "")), nil
		},
		createManager,
		func(srv server.Server, plMgr box.Manager, authMgr auth.Manager, rtConfig config.Config) error {
			setupRouting(srv, plMgr, authMgr, rtConfig)
			return nil
		},
	)

	if command.Simple {
		kern.SetConfig(kernel.ConfigService, kernel.ConfigSimpleMode, "true")
	}
	kern.Start(command.Header, command.LineServer)
	exitCode, err := command.Func(fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}
	kern.Shutdown(true)
	return exitCode
}

// runSimple is called, when the user just starts the software via a double click
// or via a simple call ``./zettelstore`` on the command line.
func runSimple() int {
	if _, err := searchAndReadConfiguration(); err == nil {
		return executeCommand(strRunSimple)
	}
	dir := "./zettel"
	if err := os.MkdirAll(dir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create zettel directory %q (%s)\n", dir, err)
		return 1
	}
	return executeCommand(strRunSimple, "-d", dir)
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

// Main is the real entrypoint of the zettelstore.
func Main(progName, buildVersion string) int {
	fullVersion := retrieveFullVersion(buildVersion)
	kernel.Main.SetConfig(kernel.CoreService, kernel.CoreProgname, progName)
	kernel.Main.SetConfig(kernel.CoreService, kernel.CoreVersion, fullVersion)
	flag.Parse()
	if *cpuprofile != "" || *memprofile != "" {
		if *cpuprofile != "" {
			kernel.Main.StartProfiling(kernel.ProfileCPU, *cpuprofile)
		} else {
			kernel.Main.StartProfiling(kernel.ProfileHead, *memprofile)
		}
		defer kernel.Main.StopProfiling()
	}
	args := flag.Args()
	if len(args) == 0 {
		return runSimple()
	}
	return executeCommand(args[0], args[1:]...)
}

func retrieveFullVersion(version string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	var revision, dirty string
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			revision = "+" + kv.Value
			if len(revision) > 11 {
				revision = revision[:11]
			}
		case "vcs.modified":
			if kv.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	return version + revision + dirty
}

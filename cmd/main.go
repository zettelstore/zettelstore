//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
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
	"zettelstore.de/z/web/server"
)

const (
	defConfigfile = ".zscfg"
)

func init() {
	RegisterCommand(Command{
		Name: "help",
		Func: func(*flag.FlagSet, *meta.Meta) (int, error) {
			fmt.Println("Available commands:")
			for _, name := range List() {
				fmt.Printf("- %q\n", name)
			}
			return 0, nil
		},
	})
	RegisterCommand(Command{
		Name:   "version",
		Func:   func(*flag.FlagSet, *meta.Meta) (int, error) { return 0, nil },
		Header: true,
	})
	RegisterCommand(Command{
		Name:       "run",
		Func:       runFunc,
		Boxes:      true,
		Header:     true,
		LineServer: true,
		Flags:      flgRun,
	})
	RegisterCommand(Command{
		Name:   "run-simple",
		Func:   runSimpleFunc,
		Boxes:  true,
		Header: true,
		Flags:  flgSimpleRun,
	})
	RegisterCommand(Command{
		Name: "file",
		Func: cmdFile,
		Flags: func(fs *flag.FlagSet) {
			fs.String("t", api.EncoderHTML.String(), "target output encoding")
		},
	})
	RegisterCommand(Command{
		Name: "password",
		Func: cmdPassword,
	})
}

func readConfig(fs *flag.FlagSet) (cfg *meta.Meta) {
	var configFile string
	if configFlag := fs.Lookup("c"); configFlag != nil {
		configFile = configFlag.Value.String()
	} else {
		configFile = defConfigfile
	}
	content, err := os.ReadFile(configFile)
	if err != nil {
		return meta.New(id.Invalid)
	}
	return meta.NewFromInput(id.Invalid, input.NewInput(string(content)))
}

func getConfig(fs *flag.FlagSet) *meta.Meta {
	cfg := readConfig(fs)
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
	for _, p := range cfg.PairsRest(false) {
		if key := p.Key; strings.HasPrefix(key, kernel.BoxURIs) {
			cfg.Delete(key)
		}
	}
}

const (
	keyAdminPort         = "admin-port"
	keyDefaultDirBoxType = "default-dir-box-type"
	keyInsecureCookie    = "insecure-cookie"
	keyListenAddr        = "listen-addr"
	keyOwner             = "owner"
	keyPersistentCookie  = "persistent-cookie"
	keyBoxOneURI         = kernel.BoxURIs + "1"
	keyReadOnly          = "read-only-mode"
	keyTokenLifetimeHTML = "token-lifetime-html"
	keyTokenLifetimeAPI  = "token-lifetime-api"
	keyURLPrefix         = "url-prefix"
	keyVerbose           = "verbose"
)

func setServiceConfig(cfg *meta.Meta) error {
	ok := setConfigValue(true, kernel.CoreService, kernel.CoreVerbose, cfg.GetBool(keyVerbose))
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
		kernel.Main.Log("unable to set configuration:", key, val)
	}
	return ok && done
}

func setupOperations(cfg *meta.Meta, withBoxes bool) {
	var createManager kernel.CreateBoxManagerFunc
	if withBoxes {
		err := raiseFdLimit()
		if err != nil {
			srvm := kernel.Main
			srvm.Log("Raising some limitions did not work:", err)
			srvm.Log("Prepare to encounter errors. Most of them can be mitigated. See the manual for details")
			srvm.SetConfig(kernel.BoxService, kernel.BoxDefaultDirType, kernel.BoxDirTypeSimple)
		}
		createManager = func(boxURIs []*url.URL, authManager auth.Manager, rtConfig config.Config) (box.Manager, error) {
			compbox.Setup(cfg)
			return manager.New(boxURIs, authManager, rtConfig)
		}
	} else {
		createManager = func([]*url.URL, auth.Manager, config.Config) (box.Manager, error) { return nil, nil }
	}

	kernel.Main.SetCreators(
		func(readonly bool, owner id.Zid) (auth.Manager, error) {
			return impl.New(readonly, owner, cfg.GetDefault("secret", "")), nil
		},
		createManager,
		func(srv server.Server, plMgr box.Manager, authMgr auth.Manager, rtConfig config.Config) error {
			setupRouting(srv, plMgr, authMgr, rtConfig)
			return nil
		},
	)
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
	setupOperations(cfg, command.Boxes)
	kernel.Main.Start(command.Header, command.LineServer)
	exitCode, err := command.Func(fs, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}
	kernel.Main.Shutdown(true)
	return exitCode
}

// Main is the real entrypoint of the zettelstore.
func Main(progName, buildVersion string) {
	kernel.Main.SetConfig(kernel.CoreService, kernel.CoreProgname, progName)
	kernel.Main.SetConfig(kernel.CoreService, kernel.CoreVersion, buildVersion)
	var exitCode int
	if len(os.Args) <= 1 {
		exitCode = runSimple()
	} else {
		exitCode = executeCommand(os.Args[1], os.Args[2:]...)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

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
	"net/http"
	"os"
	"strconv"
	"strings"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/auth/impl"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
	"zettelstore.de/z/service"
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
		Name:   "run",
		Func:   runFunc,
		Places: true,
		Header: true,
		Flags:  flgRun,
	})
	RegisterCommand(Command{
		Name:   "run-simple",
		Func:   runSimpleFunc,
		Places: true,
		Header: true,
		Flags:  flgSimpleRun,
	})
	RegisterCommand(Command{
		Name: "file",
		Func: cmdFile,
		Flags: func(fs *flag.FlagSet) {
			fs.String("t", "html", "target output format")
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
			cfg.Set(keyPlaceOneURI, val)
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

const (
	keyAdminPort           = "admin-port"
	keyDefaultDirPlaceType = "default-dir-place-type"
	keyListenAddr          = "listen-addr"
	keyOwner               = "owner"
	keyPlaceOneURI         = "place-1-uri"
	keyReadOnly            = "read-only-mode"
	keyVerbose             = "verbose"
	keyURLPrefix           = "url-prefix"
)

func setServiceConfig(cfg *meta.Meta) error {
	ok := setConfigValue(true, service.SubCore, service.CoreVerbose, cfg.GetBool(keyVerbose))
	if val, found := cfg.Get(keyAdminPort); found {
		ok = setConfigValue(ok, service.SubCore, service.CorePort, val)
	}

	ok = setConfigValue(ok, service.SubAuth, service.AuthOwner, cfg.GetDefault(keyOwner, ""))
	ok = setConfigValue(ok, service.SubAuth, service.AuthReadonly, cfg.GetBool(keyReadOnly))

	ok = setConfigValue(
		ok, service.SubPlace, service.PlaceDefaultDirType,
		cfg.GetDefault(keyDefaultDirPlaceType, service.PlaceDirTypeNotify))

	ok = setConfigValue(
		ok, service.SubWeb, service.WebListenAddress,
		cfg.GetDefault(keyListenAddr, "127.0.0.1:23123"))
	ok = setConfigValue(ok, service.SubWeb, service.WebURLPrefix, cfg.GetDefault(keyURLPrefix, "/"))

	if !ok {
		return errors.New("unable to set configuration")
	}
	return nil
}

func setConfigValue(ok bool, subsys service.Subservice, key string, val interface{}) bool {
	done := service.Main.SetConfig(subsys, key, fmt.Sprintf("%v", val))
	if !done {
		service.Main.Log("unable to set configuration:", key, val)
	}
	return ok && done
}

func setupOperations(cfg *meta.Meta, withPlaces bool) error {
	var createManager service.CreatePlaceManagerFunc
	if withPlaces {
		err := raiseFdLimit()
		if err != nil {
			srvm := service.Main
			srvm.Log("Raising some limitions did not work:", err)
			srvm.Log("Prepare to encounter errors. Most of them can be mitigated. See the manual for details")
			srvm.SetConfig(service.SubPlace, service.PlaceDefaultDirType, service.PlaceDirTypeSimple)
		}
		createManager = func(authManager auth.Manager) (place.Manager, error) {
			return manager.New(getPlaces(cfg), cfg, authManager)
		}
	} else {
		createManager = func(authManager auth.Manager) (place.Manager, error) { return nil, nil }
	}
	startup.SetupStartupConfig(cfg)

	service.Main.SetCreators(
		func(readonly bool, owner id.Zid) (auth.Manager, error) {
			return impl.New(readonly, owner), nil
		},
		createManager,
		func(urlPrefix string, plMgr place.Manager, authMgr auth.Manager) (http.Handler, error) {
			return setupRouting(urlPrefix, plMgr, authMgr), nil
		},
	)
	return nil
}

func getPlaces(cfg *meta.Meta) []string {
	var result []string = nil
	for cnt := 1; ; cnt++ {
		key := fmt.Sprintf("place-%v-uri", cnt)
		uri, ok := cfg.Get(key)
		if !ok || uri == "" {
			if cnt > 1 {
				break
			}
			uri = "dir:./zettel"
		}
		result = append(result, uri)
	}
	return result
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
	if err := setupOperations(cfg, command.Places); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		return 2
	}

	service.Main.Start(command.Header)

	exitCode, err := command.Func(fs, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}
	if err := service.Main.StopSub(service.SubPlace); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}
	return exitCode
}

// Main is the real entrypoint of the zettelstore.
func Main(progName, buildVersion string) {
	service.Main.SetConfig(service.SubCore, service.CoreProgname, progName)
	service.Main.SetConfig(service.SubCore, service.CoreVersion, buildVersion)
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

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
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/index/indexer"
	"zettelstore.de/z/input"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
	"zettelstore.de/z/place/progplace"
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
		Simple: true,
		Header: true,
		Flags:  flgSimpleRun,
	})
	RegisterCommand(Command{
		Name:   "config",
		Func:   cmdConfig,
		Flags:  flgRun,
		Header: true,
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

func getConfig(fs *flag.FlagSet) (cfg *meta.Meta) {
	var configFile string
	if configFlag := fs.Lookup("c"); configFlag != nil {
		configFile = configFlag.Value.String()
	} else {
		configFile = defConfigfile
	}
	content, err := os.ReadFile(configFile)
	if err != nil {
		cfg = meta.New(id.Invalid)
	} else {
		cfg = meta.NewFromInput(id.Invalid, input.NewInput(string(content)))
	}
	fs.Visit(func(flg *flag.Flag) {
		switch flg.Name {
		case "p":
			cfg.Set(keyListenAddr, "127.0.0.1:"+flg.Value.String())
		case "d":
			val := flg.Value.String()
			if strings.HasPrefix(val, "/") {
				val = "dir://" + val
			} else {
				val = "dir:" + val
			}
			cfg.Set(startup.KeyPlaceOneURI, val)
		case "r":
			cfg.Set(keyReadOnly, flg.Value.String())
		case "v":
			cfg.Set(keyVerbose, flg.Value.String())
		}
	})
	return cfg
}

const (
	keyDefaultDirPlaceType = "default-dir-place-type"
	keyListenAddr          = "listen-addr"
	keyReadOnly            = "read-only-mode"
	keyVerbose             = "verbose"
	keyURLPrefix           = "url-prefix"
)

func setServiceConfig(cfg *meta.Meta, simple bool) error {
	ok := setConfigValue(true, service.SubMain, service.MainVerbose, cfg.GetBool(keyVerbose))

	ok = setConfigValue(ok, service.SubAuth, service.AuthReadonly, cfg.GetBool(keyReadOnly))
	// AuthSimple must be set last, when it is known to have authentication or not.
	// Previous code: 	config.simple = simple && !config.withAuth
	ok = setConfigValue(ok, service.SubAuth, service.AuthSimple, simple)

	ok = setConfigValue(
		ok, service.SubPlace, service.PlaceDefaultDirType,
		cfg.GetDefault(keyDefaultDirPlaceType, service.PlaceDirTypeNotify))

	ok = setConfigValue(ok, service.SubWeb, service.WebListenAddress, cfg.GetDefault(keyListenAddr, "127.0.0.1:23123"))
	ok = setConfigValue(ok, service.SubWeb, service.WebURLPrefix, cfg.GetDefault(keyURLPrefix, "/"))

	if !ok {
		return errors.New("unable to set configuration")
	}
	return nil
}

func setConfigValue(ok bool, subsys service.Subservice, key string, val interface{}) bool {
	if !ok {
		return false
	}
	return service.Main.SetConfig(subsys, key, fmt.Sprintf("%v", val))
}

func setupOperations(cfg *meta.Meta, withPlaces bool) error {
	var mgr place.Manager
	var idx index.Indexer
	if withPlaces {
		err := raiseFdLimit()
		if err != nil {
			srvm := service.Main
			srvm.Log("Raising some limitions did not work:", err)
			srvm.Log("Prepare to encounter errors. Most of them can be mitigated. See the manual for details")
			srvm.SetConfig(service.SubPlace, service.PlaceDefaultDirType, service.PlaceDirTypeSimple)
		}
		startup.SetupStartupConfig(cfg)
		readonlyMode := service.Main.GetConfig(service.SubAuth, service.AuthReadonly).(bool)
		idx = indexer.New()
		filter := index.NewMetaFilter(idx)
		mgr, err = manager.New(getPlaces(cfg), readonlyMode, filter)
		if err != nil {
			return err
		}
	} else {
		startup.SetupStartupConfig(cfg)
	}

	startup.SetupStartupService(mgr, idx)
	if withPlaces {
		if err := mgr.Start(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "Unable to start zettel place")
			return err
		}
		runtime.SetupConfiguration(mgr)
		progplace.Setup(cfg, mgr, idx)
		idx.Start(mgr)
	}
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

func cleanupOperations(withPlaces bool) error {
	if withPlaces {
		startup.Indexer().Stop()
		if err := startup.PlaceManager().Stop(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "Unable to stop zettel place")
			return err
		}
	}
	return nil
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
	if err := setServiceConfig(cfg, command.Simple); err != nil {
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
	if err := cleanupOperations(command.Places); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}
	return exitCode
}

// Main is the real entrypoint of the zettelstore.
func Main(progName, buildVersion string) {
	service.Main.SetConfig(service.SubMain, service.MainProgname, progName)
	service.Main.SetConfig(service.SubMain, service.MainVersion, buildVersion)
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

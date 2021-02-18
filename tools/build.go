//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package main provides a command to build and run the software.
package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func executeCommand(env []string, name string, arg ...string) (string, error) {
	if verbose {
		if len(env) > 0 {
			for i, e := range env {
				fmt.Fprintf(os.Stderr, "ENV%d %v\n", i+1, e)
			}
		}
		fmt.Fprintln(os.Stderr, "EXEC", name, arg)
	}
	if len(env) > 0 {
		env = append(env, os.Environ()...)
	}
	var out bytes.Buffer
	cmd := exec.Command(name, arg...)
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return out.String(), err
}

func readVersionFile() (string, error) {
	content, err := ioutil.ReadFile("VERSION")
	if err != nil {
		return "", err
	}
	return strings.TrimFunc(string(content), func(r rune) bool {
		return r <= ' '
	}), nil
}

var fossilHash = regexp.MustCompile("\\[[0-9a-fA-F]+\\]")
var dirtyPrefixes = []string{"DELETED", "ADDED", "UPDATED", "CONFLICT", "EDITED", "RENAMED"}

func readFossilVersion() (string, error) {
	s, err := executeCommand(nil, "fossil", "timeline", "--limit", "1")
	if err != nil {
		return "", err
	}
	hash := fossilHash.FindString(s)
	if len(hash) < 3 {
		return "", errors.New("no fossil hash found")
	}
	hash = hash[1 : len(hash)-1]

	s, err = executeCommand(nil, "fossil", "status")
	if err != nil {
		return "", err
	}
	for _, line := range splitLines(s) {
		for _, prefix := range dirtyPrefixes {
			if strings.HasPrefix(line, prefix) {
				return hash + "-dirty", nil
			}
		}
	}
	return hash, nil
}

func splitLines(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}

func getVersionData() (string, string) {
	base, err := readVersionFile()
	if err != nil {
		base = "dev"
	}
	fossil, err := readFossilVersion()
	if err != nil {
		return base, ""
	}
	return base, fossil
}

func calcVersion(base, vcs string) string { return base + "+" + vcs }

func getVersion() string {
	base, vcs := getVersionData()
	return calcVersion(base, vcs)
}
func cmdCheck() error {
	out, err := executeCommand(nil, "go", "test", "./...")
	if err != nil {
		for _, line := range splitLines(out) {
			if strings.HasPrefix(line, "ok") || strings.HasPrefix(line, "?") {
				continue
			}
			fmt.Fprintln(os.Stderr, line)
		}
		return err
	}
	out, err = executeCommand(nil, "go", "vet", "./...")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some checks failed")
		if len(out) > 0 {
			fmt.Fprintln(os.Stderr, out)
		}
		return err
	}
	out, err = executeCommand(nil, "golint", "./...")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some lints failed")
		if len(out) > 0 {
			fmt.Fprintln(os.Stderr, out)
		}
		return err
	}
	if out, err = executeCommand(nil, "which", "shadow"); err == nil && len(out) > 0 {
		out, err = executeCommand(nil, "go", "vet", "-vettool", strings.TrimSpace(out), "./...")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Some shadowed variables found")
			if len(out) > 0 {
				fmt.Fprintln(os.Stderr, out)
			}
			return err
		}
	}
	out, err = executeCommand(nil, "fossil", "extra")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to execute 'fossil extra'")
	} else if len(out) > 0 {
		fmt.Fprint(os.Stderr, "Warning: unversioned file(s):")
		for i, extra := range splitLines(out) {
			if i > 0 {
				fmt.Fprint(os.Stderr, ",")
			}
			fmt.Fprintf(os.Stderr, " %q", extra)
		}
		fmt.Fprintln(os.Stderr)
	}
	return nil
}

func cmdBuild() error {
	return doBuild(nil, getVersion(), "bin/zettelstore")
}

func doBuild(env []string, version string, target string) error {
	out, err := executeCommand(
		env,
		"go", "build",
		"-tags", "osusergo,netgo",
		"-ldflags", fmt.Sprintf("-X main.version=%v -w", version),
		"-o", target,
		"zettelstore.de/z/cmd/zettelstore",
	)
	if err != nil {
		return err
	}
	if len(out) > 0 {
		fmt.Println(out)
	}
	return nil
}

func cmdRelease() error {
	base, fossil := getVersionData()
	if strings.HasSuffix(base, "dev") {
		base = base[:len(base)-3] + "preview-" + time.Now().Format("20060102")
	}
	if strings.HasSuffix(fossil, "-dirty") {
		fmt.Fprintf(os.Stderr, "Warning: releasing a dirty version %v\n", fossil)
		base = base + "-dirty"
	}
	if err := cmdCheck(); err != nil {
		return err
	}
	releases := []struct {
		arch string
		os   string
		env  []string
		name string
	}{
		{"amd64", "linux", nil, "zettelstore"},
		{"arm", "linux", []string{"GOARM=6"}, "zettelstore-arm6"},
		{"amd64", "darwin", nil, "iZettelstore"},
		{"amd64", "windows", nil, "zettelstore.exe"},
	}
	for _, rel := range releases {
		env := append(rel.env, "GOARCH="+rel.arch, "GOOS="+rel.os)
		zsName := filepath.Join("releases", rel.name)
		if err := doBuild(env, calcVersion(base, fossil), zsName); err != nil {
			return err
		}
		zipName := fmt.Sprintf("zettelstore-%v-%v-%v.zip", base, rel.os, rel.arch)
		if err := createZip(zsName, zipName, rel.name); err != nil {
			return err
		}
		if err := os.Remove(zsName); err != nil {
			return err
		}
	}
	return nil
}

func createZip(zsName string, zipName string, fileName string) error {
	zsFile, err := os.Open(zsName)
	if err != nil {
		return err
	}
	defer zsFile.Close()
	zipFile, err := os.OpenFile(filepath.Join("releases", zipName), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	hash := crc32.NewIEEE()
	if _, err = io.Copy(hash, zsFile); err != nil {
		return err
	}
	if _, err = zsFile.Seek(0, io.SeekStart); err != nil {
		return nil
	}
	stat, err := zsFile.Stat()
	if err != nil {
		return err
	}
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:               fileName,
		Method:             zip.Deflate,
		Modified:           stat.ModTime(),
		CRC32:              hash.Sum32(),
		UncompressedSize64: uint64(stat.Size()),
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, zsFile)
	return err
}

func cmdClean() error {
	for _, dir := range []string{"bin", "releases"} {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func cmdHelp() {
	fmt.Println(`Usage: go run tools/build.go [-v] COMMAND

Options:
  -v       Verbose output.

Commands:
  build    Build the software for local computer.
  check    Check current working state: execute tests, static analysis tools,
           extra files, ...
           Is automatically done when releasing the software.
  clean    Remove all build and release directories.
  help     Outputs this text.
  release  Create the software for various platforms and put them in
           appropriate named ZIP files.
  version  Print the current version of the software.

All commands can be abbreviated as long as they remain unique.`)
}

var (
	verbose bool
)

func main() {
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()
	var err error
	args := flag.Args()
	if len(args) < 1 {
		cmdHelp()
	} else {
		switch args[0] {
		case "b", "bu", "bui", "buil", "build":
			err = cmdBuild()
		case "r", "re", "rel", "rele", "relea", "releas", "release":
			err = cmdRelease()
		case "cl", "cle", "clea", "clean":
			err = cmdClean()
		case "v", "ve", "ver", "vers", "versi", "versio", "version":
			fmt.Print(getVersion())
		case "ch", "che", "chec", "check":
			err = cmdCheck()
		case "h", "he", "hel", "help":
			cmdHelp()
		default:
			fmt.Fprintf(os.Stderr, "Unknown command %q\n", args[0])
			cmdHelp()
			os.Exit(1)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

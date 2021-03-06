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
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"zettelstore.de/z/strfun"
)

func executeCommand(env []string, name string, arg ...string) (string, error) {
	logCommand("EXEC", env, name, arg)
	var out bytes.Buffer
	cmd := prepareCommand(env, name, arg, &out)
	err := cmd.Run()
	return out.String(), err
}

func prepareCommand(env []string, name string, arg []string, out io.Writer) *exec.Cmd {
	if len(env) > 0 {
		env = append(env, os.Environ()...)
	}
	cmd := exec.Command(name, arg...)
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	return cmd
}

func logCommand(exec string, env []string, name string, arg []string) {
	if verbose {
		if len(env) > 0 {
			for i, e := range env {
				fmt.Fprintf(os.Stderr, "ENV%d %v\n", i+1, e)
			}
		}
		fmt.Fprintln(os.Stderr, exec, name, arg)
	}
}

func readVersionFile() (string, error) {
	content, err := os.ReadFile("VERSION")
	if err != nil {
		return "", err
	}
	return strings.TrimFunc(string(content), func(r rune) bool {
		return r <= ' '
	}), nil
}

var fossilCheckout = regexp.MustCompile(`^checkout:\s+([0-9a-f]+)\s`)
var dirtyPrefixes = []string{
	"DELETED ", "ADDED ", "UPDATED ", "CONFLICT ", "EDITED ", "RENAMED ", "EXTRA "}

const dirtySuffix = "-dirty"

func readFossilVersion() (string, error) {
	s, err := executeCommand(nil, "fossil", "status", "--differ")
	if err != nil {
		return "", err
	}
	var hash, suffix string
	for _, line := range strfun.SplitLines(s) {
		if hash == "" {
			if m := fossilCheckout.FindStringSubmatch(line); len(m) > 0 {
				hash = m[1][:10]
				if suffix != "" {
					return hash + suffix, nil
				}
				continue
			}
		}
		if suffix == "" {
			for _, prefix := range dirtyPrefixes {
				if strings.HasPrefix(line, prefix) {
					suffix = dirtySuffix
					if hash != "" {
						return hash + suffix, nil
					}
					break
				}
			}
		}
	}
	return hash, nil
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

func findExec(cmd string) string {
	if path, err := executeCommand(nil, "which", "shadow"); err == nil && path != "" {
		return path
	}
	return ""
}

func cmdCheck() error {
	if err := checkGoTest("./..."); err != nil {
		return err
	}
	if err := checkGoVet(); err != nil {
		return err
	}
	if err := checkGoLint(); err != nil {
		return err
	}
	if err := checkGoVetShadow(); err != nil {
		return err
	}
	if err := checkStaticcheck(); err != nil {
		return err
	}
	return checkFossilExtra()
}

func checkGoTest(pkg string, testParams ...string) error {
	args := []string{"test", pkg}
	args = append(args, testParams...)
	out, err := executeCommand(nil, "go", args...)
	if err != nil {
		for _, line := range strfun.SplitLines(out) {
			if strings.HasPrefix(line, "ok") || strings.HasPrefix(line, "?") {
				continue
			}
			fmt.Fprintln(os.Stderr, line)
		}
	}
	return err
}

func checkGoVet() error {
	out, err := executeCommand(nil, "go", "vet", "./...")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some checks failed")
		if len(out) > 0 {
			fmt.Fprintln(os.Stderr, out)
		}
	}
	return err
}

func checkGoLint() error {
	out, err := executeCommand(nil, "golint", "./...")
	if out != "" {
		fmt.Fprintln(os.Stderr, "Some lints failed")
		fmt.Fprint(os.Stderr, out)
	}
	return err
}

func checkGoVetShadow() error {
	path := findExec("shadow")
	if path == "" {
		return nil
	}
	out, err := executeCommand(nil, "go", "vet", "-vettool", strings.TrimSpace(path), "./...")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some shadowed variables found")
		if len(out) > 0 {
			fmt.Fprintln(os.Stderr, out)
		}
	}
	return err
}
func checkStaticcheck() error {
	out, err := executeCommand(nil, "staticcheck", "./...")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some staticcheck problems found")
		if len(out) > 0 {
			fmt.Fprintln(os.Stderr, out)
		}
	}
	return err
}

func checkFossilExtra() error {
	out, err := executeCommand(nil, "fossil", "extra")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to execute 'fossil extra'")
		return err
	}
	if len(out) > 0 {
		fmt.Fprint(os.Stderr, "Warning: unversioned file(s):")
		for i, extra := range strfun.SplitLines(out) {
			if i > 0 {
				fmt.Fprint(os.Stderr, ",")
			}
			fmt.Fprintf(os.Stderr, " %q", extra)
		}
		fmt.Fprintln(os.Stderr)
	}
	return nil
}

type zsInfo struct {
	cmd          *exec.Cmd
	out          bytes.Buffer
	adminAddress string
}

func cmdTestAPI() error {
	var err error
	var info zsInfo
	needServer := !addressInUse(":23123")
	if needServer {
		err = startZettelstore(&info)
	}
	if err != nil {
		return err
	}
	err = checkGoTest("zettelstore.de/z/client", "-base-url", "http://127.0.0.1:23123")
	if needServer {
		err1 := stopZettelstore(&info)
		if err == nil {
			err = err1
		}
	}
	return err
}

func startZettelstore(info *zsInfo) error {
	info.adminAddress = ":2323"
	name, arg := "go", []string{
		"run", "cmd/zettelstore/main.go", "run",
		"-c", "./testdata/testbox/19700101000000.zettel", "-a", info.adminAddress[1:]}
	logCommand("FORK", nil, name, arg)
	cmd := prepareCommand(nil, name, arg, &info.out)
	if !verbose {
		cmd.Stderr = nil
	}
	err := cmd.Start()
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 100)
		if addressInUse(info.adminAddress) {
			info.cmd = cmd
			return err
		}
	}
	return errors.New("zettelstore did not start")
}

func stopZettelstore(i *zsInfo) error {
	conn, err := net.Dial("tcp", i.adminAddress)
	if err != nil {
		fmt.Println("Unable to stop Zettelstore")
		return err
	}
	io.WriteString(conn, "shutdown\n")
	conn.Close()
	err = i.cmd.Wait()
	return err
}

func addressInUse(address string) bool {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func cmdBuild() error {
	return doBuild(nil, getVersion(), "bin/zettelstore")
}

func doBuild(env []string, version, target string) error {
	out, err := executeCommand(
		env,
		"go", "build",
		"-tags", "osusergo,netgo",
		"-trimpath",
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

func cmdManual() error {
	base, _ := getReleaseVersionData()
	return createManualZip(".", base)
}

func createManualZip(path, base string) error {
	manualPath := filepath.Join("docs", "manual")
	entries, err := os.ReadDir(manualPath)
	if err != nil {
		return err
	}
	zipName := filepath.Join(path, "manual-"+base+".zip")
	zipFile, err := os.OpenFile(zipName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, entry := range entries {
		if err = createManualZipEntry(manualPath, entry, zipWriter); err != nil {
			return err
		}
	}
	return nil
}

func createManualZipEntry(path string, entry fs.DirEntry, zipWriter *zip.Writer) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	fh, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	fh.Name = entry.Name()
	fh.Method = zip.Deflate
	w, err := zipWriter.CreateHeader(fh)
	if err != nil {
		return err
	}
	manualFile, err := os.Open(filepath.Join(path, entry.Name()))
	if err != nil {
		return err
	}
	defer manualFile.Close()
	_, err = io.Copy(w, manualFile)
	return err
}

func getReleaseVersionData() (string, string) {
	base, fossil := getVersionData()
	if strings.HasSuffix(base, "dev") {
		base = base[:len(base)-3] + "preview-" + time.Now().Format("20060102")
	}
	if strings.HasSuffix(fossil, dirtySuffix) {
		fmt.Fprintf(os.Stderr, "Warning: releasing a dirty version %v\n", fossil)
		base = base + dirtySuffix
	}
	return base, fossil
}

func cmdRelease() error {
	if err := cmdCheck(); err != nil {
		return err
	}
	base, fossil := getReleaseVersionData()
	releases := []struct {
		arch string
		os   string
		env  []string
		name string
	}{
		{"amd64", "linux", nil, "zettelstore"},
		{"arm", "linux", []string{"GOARM=6"}, "zettelstore"},
		{"amd64", "darwin", nil, "iZettelstore"},
		{"arm64", "darwin", nil, "iZettelstore"},
		{"amd64", "windows", nil, "zettelstore.exe"},
	}
	for _, rel := range releases {
		env := append(rel.env, "GOARCH="+rel.arch, "GOOS="+rel.os)
		zsName := filepath.Join("releases", rel.name)
		if err := doBuild(env, calcVersion(base, fossil), zsName); err != nil {
			return err
		}
		zipName := fmt.Sprintf("zettelstore-%v-%v-%v.zip", base, rel.os, rel.arch)
		if err := createReleaseZip(zsName, zipName, rel.name); err != nil {
			return err
		}
		if err := os.Remove(zsName); err != nil {
			return err
		}
	}
	return createManualZip("releases", base)
}

func createReleaseZip(zsName, zipName, fileName string) error {
	zipFile, err := os.OpenFile(filepath.Join("releases", zipName), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zw := zip.NewWriter(zipFile)
	defer zw.Close()
	err = addFileToZip(zw, zsName, fileName)
	if err != nil {
		return err
	}
	err = addFileToZip(zw, "LICENSE.txt", "LICENSE.txt")
	if err != nil {
		return err
	}
	err = addFileToZip(zw, "docs/readmezip.txt", "README.txt")
	return err
}

func addFileToZip(zipFile *zip.Writer, filepath, filename string) error {
	zsFile, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer zsFile.Close()
	stat, err := zsFile.Stat()
	if err != nil {
		return err
	}
	fh, err := zip.FileInfoHeader(stat)
	if err != nil {
		return err
	}
	fh.Name = filename
	fh.Method = zip.Deflate
	w, err := zipFile.CreateHeader(fh)
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
  manual   Create a ZIP file with all manual zettel
  release  Create the software for various platforms and put them in
           appropriate named ZIP files.
  testapi  Starts a Zettelstore and execute API tests.
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
		case "m", "ma", "man", "manu", "manua", "manual":
			err = cmdManual()
		case "r", "re", "rel", "rele", "relea", "releas", "release":
			err = cmdRelease()
		case "cl", "cle", "clea", "clean":
			err = cmdClean()
		case "v", "ve", "ver", "vers", "versi", "versio", "version":
			fmt.Print(getVersion())
		case "ch", "che", "chec", "check":
			err = cmdCheck()
		case "t", "te", "tes", "test", "testa", "testap", "testapi":
			cmdTestAPI()
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

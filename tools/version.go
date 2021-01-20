package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

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
	var out bytes.Buffer
	cmd := exec.Command("fossil", "timeline", "--limit", "1")
	cmd.Stdin = nil
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	hash := fossilHash.FindString(out.String())
	if len(hash) < 3 {
		return "", errors.New("No fossil hash found")
	}
	hash = hash[1 : len(hash)-1]

	out.Reset()
	cmd = exec.Command("fossil", "status")
	cmd.Stdin = nil
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	lines := strings.FieldsFunc(out.String(), func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	for _, line := range lines {
		for _, prefix := range dirtyPrefixes {
			if strings.HasPrefix(line, prefix) {
				return hash + "-dirty", nil
			}
		}
	}
	return hash, nil
}

func main() {
	base, err := readVersionFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "No VERSION found: %v\n", err)
		base = "dev"
	}
	fossil, err := readFossilVersion()
	if err != nil {
		fmt.Print(base)
	}
	fmt.Printf("%v+%v", base, fossil)
}

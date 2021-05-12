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
	"io"
	"os"
	"runtime/metrics"
	"sort"
	"strings"

	"zettelstore.de/z/service"
	"zettelstore.de/z/strfun"
)

type cmdSession struct {
	w        io.Writer
	srv      *myService
	echo     bool
	header   bool
	colwidth int
}

func (sess *cmdSession) initialize(w io.Writer, srv *myService) {
	sess.w = w
	sess.srv = srv
	sess.header = true
	sess.colwidth = 80
}

func (sess *cmdSession) executeLine(line string) bool {
	if sess.echo {
		sess.println(line)
	}
	cmd, args := splitLine(line)
	if c, ok := commands[cmd]; ok {
		return c.Func(sess, cmd, args)
	}
	if cmd == "help" {
		return cmdHelp(sess, cmd, args)
	}
	sess.println("Unknown command:", cmd, strings.Join(args, " "))
	return true
}

func (sess *cmdSession) println(args ...string) {
	if len(args) > 0 {
		io.WriteString(sess.w, args[0])
		for _, arg := range args[1:] {
			io.WriteString(sess.w, " ")
			io.WriteString(sess.w, arg)
		}
	}
	io.WriteString(sess.w, "\n")
}

func (sess *cmdSession) printTable(table [][]string) {
	maxLen := sess.calcMaxLen(table)
	if len(maxLen) == 0 {
		return
	}
	if sess.header {
		sess.printRow(table[0], maxLen, " | ", ' ')
		hLine := make([]string, len(table[0]))
		sess.printRow(hLine, maxLen, "-+-", '-')
	}

	for _, row := range table[1:] {
		sess.printRow(row, maxLen, " | ", ' ')
	}
}

func (sess *cmdSession) calcMaxLen(table [][]string) []int {
	maxLen := make([]int, 0)
	for _, row := range table {
		for colno, column := range row {
			if colno >= len(maxLen) {
				maxLen = append(maxLen, 0)
			}
			colLen := strfun.Length(column)
			if colLen <= maxLen[colno] {
				continue
			}
			if colLen < sess.colwidth {
				maxLen[colno] = colLen
			} else {
				maxLen[colno] = sess.colwidth
			}
		}
	}
	return maxLen
}

func (sess *cmdSession) printRow(row []string, maxLen []int, delim string, pad rune) {
	for colno, column := range row {
		if colno > 0 {
			io.WriteString(sess.w, delim)
		}
		io.WriteString(sess.w, strfun.JustifyLeft(column, maxLen[colno], pad))
	}
	io.WriteString(sess.w, "\n")
}

func splitLine(line string) (string, []string) {
	s := strings.Fields(line)
	if len(s) == 0 {
		return "", nil
	}
	return strings.ToLower(s[0]), s[1:]
}

type command struct {
	Text string
	Func func(sess *cmdSession, cmd string, args []string) bool
}

var commands = map[string]command{
	"": {"", func(*cmdSession, string, []string) bool { return true }},
	"bye": {
		"end this session",
		func(*cmdSession, string, []string) bool { return false },
	},
	"shutdown": {
		"shutdown Zettelstore",
		func(sess *cmdSession, cmd string, args []string) bool { sess.srv.Shutdown(); return false },
	},
	"echo": {
		"toggle echo mode",
		func(sess *cmdSession, cmd string, args []string) bool {
			sess.echo = !sess.echo
			if sess.echo {
				sess.println("echo is on")
			} else {
				sess.println("echo is off")
			}
			return true
		},
	},
	"header": {
		"toggle table header",
		func(sess *cmdSession, cmd string, args []string) bool {
			sess.header = !sess.header
			if sess.header {
				sess.println("header are on")
			} else {
				sess.println("header are off")
			}
			return true
		},
	},
	"env":        {"show environment values", cmdEnvironment},
	"get-config": {"show configuration data", cmdGetConfig},
	"subsystems": {"show available subsystems", cmdSubsystems},
	"metrics":    {"show Go runtime metrics", cmdMetrics},
}

func cmdHelp(sess *cmdSession, cmd string, args []string) bool {
	cmds := make([]string, 0, len(commands))
	maxLen := 0
	for key := range commands {
		if key == "" {
			continue
		}
		if keyLen := strfun.Length(key); keyLen > maxLen {
			maxLen = keyLen
		}
		cmds = append(cmds, key)
	}
	sort.Strings(cmds)
	sess.println("Available commands:")
	for _, cmd := range cmds {
		sess.println("-", strfun.JustifyLeft(cmd, maxLen, ' '), commands[cmd].Text)
	}
	return true
}

func cmdGetConfig(sess *cmdSession, cmd string, args []string) bool {
	if len(args) == 0 {
		for subsrv := service.SubCore; subsrv <= service.SubWeb; subsrv++ {
			if subsrv > service.SubCore {
				sess.println()
			}
			sub := sess.srv.getSubservice(subsrv)
			listSubConfig(sess, sub)
		}
		return true
	}
	sub := sess.srv.getSubserviceByName(args[0])
	if sub == nil {
		sess.println("Unknown sub-system:", args[0])
		return true
	}
	if len(args) == 1 {
		listSubConfig(sess, sub)
		return true
	}
	val := sub.GetConfig(args[1])
	if val == nil {
		sess.println("Unknown key", args[1], "for sub-system", args[0])
		return true
	}
	sess.println(fmt.Sprintf("%v", val))
	return true
}
func listSubConfig(sess *cmdSession, sub subService) {
	l := sub.GetConfigList(true)
	table := [][]string{{"Key", "Value", "Description"}}
	for _, kdv := range l {
		table = append(table, []string{kdv.Key, kdv.Value, kdv.Descr})
	}
	sess.printTable(table)
}

func cmdSubsystems(sess *cmdSession, cmd string, args []string) bool {
	names := make([]string, 0, len(sess.srv.subNames))
	for name := range sess.srv.subNames {
		names = append(names, name)
	}
	sort.Strings(names)
	sess.println("Available sub-systems:")
	for _, name := range names {
		sess.println("-", name)
	}
	return true
}

func cmdMetrics(sess *cmdSession, cmd string, args []string) bool {
	var samples []metrics.Sample
	all := metrics.All()
	for _, d := range all {
		if d.Kind == metrics.KindFloat64Histogram {
			continue
		}
		samples = append(samples, metrics.Sample{Name: d.Name})
	}
	metrics.Read(samples)

	table := [][]string{{"Value", "Description"}}
	i := 0
	for _, d := range all {
		if d.Kind == metrics.KindFloat64Histogram {
			continue
		}
		descr := d.Description
		if pos := strings.IndexByte(descr, '.'); pos > 0 {
			descr = descr[:pos]
		}
		value := samples[i].Value
		i++
		var sVal string
		switch value.Kind() {
		case metrics.KindUint64:
			sVal = fmt.Sprintf("%v", value.Uint64())
		case metrics.KindFloat64:
			sVal = fmt.Sprintf("%v", value.Float64())
		case metrics.KindFloat64Histogram:
			sVal = "(Histogramm)"
		case metrics.KindBad:
			sVal = "BAD"
		default:
			sVal = fmt.Sprintf("(unexpected metric kind: %v)", value.Kind())
		}
		table = append(table, []string{sVal, descr})
	}
	sess.printTable(table)
	return true
}

func cmdEnvironment(sess *cmdSession, cmd string, args []string) bool {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = err.Error()
	}
	execName, err := os.Executable()
	if err != nil {
		execName = err.Error()
	}
	envs := os.Environ()
	sort.Strings(envs)

	table := [][]string{
		{"Key", "Value"},
		{"workdir", workDir},
		{"executable", execName},
	}
	for _, env := range envs {
		if pos := strings.IndexByte(env, '='); pos >= 0 && pos < len(env) {
			table = append(table, []string{env[:pos], env[pos+1:]})
		}
	}
	sess.printTable(table)
	return true
}

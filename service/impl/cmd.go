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
		sess.printRow(table[0], maxLen, "|=", " | ", ' ')
		hLine := make([]string, len(table[0]))
		sess.printRow(hLine, maxLen, "%%", "-+-", '-')
	}

	for _, row := range table[1:] {
		sess.printRow(row, maxLen, "| ", " | ", ' ')
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

func (sess *cmdSession) printRow(row []string, maxLen []int, prefix, delim string, pad rune) {
	for colno, column := range row {
		io.WriteString(sess.w, prefix)
		prefix = delim
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
	"dump-index": {"Writes the content of the index", cmdDumpIndex},
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
	"env":        {"show environment values", cmdEnvironment},
	"get-config": {"show current configuration data", cmdGetConfig},
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
	"metrics":     {"show Go runtime metrics", cmdMetrics},
	"next-config": {"show next configuration data", cmdNextConfig},
	"restart":     {"restart subservice", cmdRestart},
	"set-config":  {"set next configuration data", cmdSetConfig},
	"shutdown": {
		"shutdown Zettelstore",
		func(sess *cmdSession, cmd string, args []string) bool { sess.srv.Shutdown(false); return false },
	},
	"start":       {"start subservice", cmdStart},
	"stat":        {"show subservice statistics", cmdStat},
	"stop":        {"stop subservice", cmdStop},
	"subservices": {"show available subservices", cmdServices},
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
	showConfig(sess, args,
		listSubConfig, func(sub subService, key string) interface{} { return sub.GetConfig(key) })
	return true
}
func cmdNextConfig(sess *cmdSession, cmd string, args []string) bool {
	showConfig(sess, args,
		listNextConfig, func(sub subService, key string) interface{} { return sub.GetNextConfig(key) })
	return true
}
func showConfig(sess *cmdSession, args []string,
	listConfig func(*cmdSession, subService), getConfig func(subService, string) interface{}) {

	if len(args) == 0 {
		for subsrv := service.SubCore; subsrv <= service.SubWeb; subsrv++ {
			if subsrv > service.SubCore {
				sess.println()
			}
			sub := sess.srv.subs[subsrv].sub
			listConfig(sess, sub)
		}
		return
	}
	subD, ok := sess.srv.subNames[args[0]]
	if !ok {
		sess.println("Unknown subservice:", args[0])
		return
	}
	if len(args) == 1 {
		listConfig(sess, subD.sub)
		return
	}
	val := getConfig(subD.sub, args[1])
	if val == nil {
		sess.println("Unknown key", args[1], "for subservice", args[0])
		return
	}
	sess.println(fmt.Sprintf("%v", val))
}
func listSubConfig(sess *cmdSession, sub subService) {
	listConfig(sess, func() []service.KeyDescrValue { return sub.GetConfigList(true) })
}
func listNextConfig(sess *cmdSession, sub subService) {
	listConfig(sess, sub.GetNextConfigList)
}
func listConfig(sess *cmdSession, getConfigList func() []service.KeyDescrValue) {
	l := getConfigList()
	table := [][]string{{"Key", "Value", "Description"}}
	for _, kdv := range l {
		table = append(table, []string{kdv.Key, kdv.Value, kdv.Descr})
	}
	sess.printTable(table)
}

func cmdSetConfig(sess *cmdSession, cmd string, args []string) bool {
	if len(args) < 3 {
		sess.println("Usage:", cmd, "SUBSERIVCE KEY VALUE")
		return true
	}
	subD, ok := sess.srv.subNames[args[0]]
	if !ok {
		sess.println("Unknown subservice:", args[0])
		return true
	}
	if !subD.sub.SetConfig(args[1], args[2]) {
		sess.println("Unable to set key", args[1], "to value", args[2])
	}
	return true
}

func cmdServices(sess *cmdSession, cmd string, args []string) bool {
	names := make([]string, 0, len(sess.srv.subNames))
	for name := range sess.srv.subNames {
		names = append(names, name)
	}
	sort.Strings(names)

	table := [][]string{{"Subservice", "Status"}}
	for _, name := range names {
		if sess.srv.subNames[name].sub.IsStarted() {
			table = append(table, []string{name, "started"})
		} else {
			table = append(table, []string{name, "stopped"})
		}
	}
	sess.printTable(table)
	return true
}

func cmdStart(sess *cmdSession, cmd string, args []string) bool {
	subsrv, ok := lookupSubsrv(sess, cmd, args)
	if !ok {
		return true
	}
	err := sess.srv.doStartSub(subsrv)
	if err != nil {
		sess.println(err.Error())
	}
	return true
}

func cmdRestart(sess *cmdSession, cmd string, args []string) bool {
	subsrv, ok := lookupSubsrv(sess, cmd, args)
	if !ok {
		return true
	}
	err := sess.srv.doRestartSub(subsrv)
	if err != nil {
		sess.println(err.Error())
	}
	return true
}

func cmdStop(sess *cmdSession, cmd string, args []string) bool {
	subsrv, ok := lookupSubsrv(sess, cmd, args)
	if !ok {
		return true
	}
	err := sess.srv.doStopSub(subsrv)
	if err != nil {
		sess.println(err.Error())
	}
	return true
}

func cmdStat(sess *cmdSession, cmd string, args []string) bool {
	if len(args) == 0 {
		sess.println("Usage:", cmd, "SUBSERVICE")
		return true
	}
	subD, ok := sess.srv.subNames[args[0]]
	if !ok {
		sess.println("Unknown subservice", args[0])
		return true
	}
	kvl := subD.sub.GetStatistics()
	if len(kvl) == 0 {
		return true
	}
	table := [][]string{{"Key", "Value"}}
	for _, kv := range kvl {
		table = append(table, []string{kv.Key, kv.Value})
	}
	sess.printTable(table)
	return true
}

func lookupSubsrv(sess *cmdSession, cmd string, args []string) (service.Subservice, bool) {
	if len(args) == 0 {
		sess.println("Usage:", cmd, "SUBSERVICE")
		return 0, false
	}
	subD, ok := sess.srv.subNames[args[0]]
	if !ok {
		sess.println("Unknown subservice", args[0])
		return 0, false
	}
	return subD.subsrv, true
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

func cmdDumpIndex(sess *cmdSession, cmd string, args []string) bool {
	sess.srv.place.manager.Dump(sess.w)
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

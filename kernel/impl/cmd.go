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
	"fmt"
	"io"
	"os"
	"runtime/metrics"
	"sort"
	"strings"

	"zettelstore.de/z/kernel"
	"zettelstore.de/z/strfun"
)

type cmdSession struct {
	w        io.Writer
	kern     *myKernel
	echo     bool
	header   bool
	colwidth int
	eol      []byte
}

func (sess *cmdSession) initialize(w io.Writer, kern *myKernel) {
	sess.w = w
	sess.kern = kern
	sess.header = true
	sess.colwidth = 80
	sess.eol = []byte{'\n'}
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
	sess.println("-- Enter 'help' go get a list of valid commands.")
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
	sess.w.Write(sess.eol)
}

func (sess *cmdSession) usage(cmd, val string) {
	sess.println("Usage:", cmd, val)
}

func (sess *cmdSession) printTable(table [][]string) {
	maxLen := sess.calcMaxLen(table)
	if len(maxLen) == 0 {
		return
	}
	if sess.header {
		sess.printRow(table[0], maxLen, "|=", " | ", ' ')
		hLine := make([]string, len(table[0]))
		sess.printRow(hLine, maxLen, "|%", "-+-", '-')
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
	sess.w.Write(sess.eol)
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
	"config": {"show configuration keys", cmdConfig},
	"crlf": {
		"toggle crlf mode",
		func(sess *cmdSession, cmd string, args []string) bool {
			if len(sess.eol) == 1 {
				sess.eol = []byte{'\r', '\n'}
				sess.println("crlf is on")
			} else {
				sess.eol = []byte{'\n'}
				sess.println("crlf is off")
			}
			return true
		},
	},
	"dump-index":   {"writes the content of the index", cmdDumpIndex},
	"dump-recover": {"show data of last recovery", cmdDumpRecover},
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
	"restart":     {"restart service", cmdRestart},
	"services":    {"show available services", cmdServices},
	"set-config":  {"set next configuration data", cmdSetConfig},
	"shutdown": {
		"shutdown Zettelstore",
		func(sess *cmdSession, cmd string, args []string) bool { sess.kern.Shutdown(false); return false },
	},
	"start": {"start service", cmdStart},
	"stat":  {"show service statistics", cmdStat},
	"stop":  {"stop service", cmdStop},
}

func cmdHelp(sess *cmdSession, _ string, _ []string) bool {
	cmds := make([]string, 0, len(commands))
	for key := range commands {
		if key == "" {
			continue
		}
		cmds = append(cmds, key)
	}
	sort.Strings(cmds)
	table := [][]string{{"Command", "Description"}}
	for _, cmd := range cmds {
		table = append(table, []string{cmd, commands[cmd].Text})
	}
	sess.printTable(table)
	return true
}

func cmdConfig(sess *cmdSession, cmd string, args []string) bool {
	srvnum, ok := lookupService(sess, cmd, args)
	if !ok {
		return true
	}
	srv := sess.kern.srvs[srvnum].srv
	table := [][]string{{"Key", "Description"}}
	for _, kd := range srv.ConfigDescriptions() {
		table = append(table, []string{kd.Key, kd.Descr})
	}
	sess.printTable(table)
	return true
}
func cmdGetConfig(sess *cmdSession, _ string, args []string) bool {
	showConfig(sess, args,
		listCurConfig, func(srv service, key string) interface{} { return srv.GetConfig(key) })
	return true
}
func cmdNextConfig(sess *cmdSession, _ string, args []string) bool {
	showConfig(sess, args,
		listNextConfig, func(srv service, key string) interface{} { return srv.GetNextConfig(key) })
	return true
}
func showConfig(sess *cmdSession, args []string,
	listConfig func(*cmdSession, service), getConfig func(service, string) interface{}) {

	if len(args) == 0 {
		keys := make([]int, 0, len(sess.kern.srvs))
		for k := range sess.kern.srvs {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		for i, k := range keys {
			if i > 0 {
				sess.println()
			}
			srvD := sess.kern.srvs[kernel.Service(k)]
			sess.println("%% Service", srvD.name)
			listConfig(sess, srvD.srv)

		}
		return
	}
	srvD, ok := sess.kern.srvNames[args[0]]
	if !ok {
		sess.println("Unknown service:", args[0])
		return
	}
	if len(args) == 1 {
		listConfig(sess, srvD.srv)
		return
	}
	val := getConfig(srvD.srv, args[1])
	if val == nil {
		sess.println("Unknown key", args[1], "for service", args[0])
		return
	}
	sess.println(fmt.Sprintf("%v", val))
}
func listCurConfig(sess *cmdSession, srv service) {
	listConfig(sess, func() []kernel.KeyDescrValue { return srv.GetConfigList(true) })
}
func listNextConfig(sess *cmdSession, srv service) {
	listConfig(sess, srv.GetNextConfigList)
}
func listConfig(sess *cmdSession, getConfigList func() []kernel.KeyDescrValue) {
	l := getConfigList()
	table := [][]string{{"Key", "Value", "Description"}}
	for _, kdv := range l {
		table = append(table, []string{kdv.Key, kdv.Value, kdv.Descr})
	}
	sess.printTable(table)
}

func cmdSetConfig(sess *cmdSession, cmd string, args []string) bool {
	if len(args) < 3 {
		sess.usage(cmd, "SERVICE KEY VALUE")
		return true
	}
	srvD, ok := sess.kern.srvNames[args[0]]
	if !ok {
		sess.println("Unknown service:", args[0])
		return true
	}
	newValue := strings.Join(args[2:], " ")
	if !srvD.srv.SetConfig(args[1], newValue) {
		sess.println("Unable to set key", args[1], "to value", newValue)
	}
	return true
}

func cmdServices(sess *cmdSession, _ string, _ []string) bool {
	names := make([]string, 0, len(sess.kern.srvNames))
	for name := range sess.kern.srvNames {
		names = append(names, name)
	}
	sort.Strings(names)

	table := [][]string{{"Service", "Status"}}
	for _, name := range names {
		if sess.kern.srvNames[name].srv.IsStarted() {
			table = append(table, []string{name, "started"})
		} else {
			table = append(table, []string{name, "stopped"})
		}
	}
	sess.printTable(table)
	return true
}

func cmdStart(sess *cmdSession, cmd string, args []string) bool {
	srvnum, ok := lookupService(sess, cmd, args)
	if !ok {
		return true
	}
	err := sess.kern.doStartService(srvnum)
	if err != nil {
		sess.println(err.Error())
	}
	return true
}

func cmdRestart(sess *cmdSession, cmd string, args []string) bool {
	srvnum, ok := lookupService(sess, cmd, args)
	if !ok {
		return true
	}
	err := sess.kern.doRestartService(srvnum)
	if err != nil {
		sess.println(err.Error())
	}
	return true
}

func cmdStop(sess *cmdSession, cmd string, args []string) bool {
	srvnum, ok := lookupService(sess, cmd, args)
	if !ok {
		return true
	}
	err := sess.kern.doStopService(srvnum)
	if err != nil {
		sess.println(err.Error())
	}
	return true
}

func cmdStat(sess *cmdSession, cmd string, args []string) bool {
	if len(args) == 0 {
		sess.usage(cmd, "SERVICE")
		return true
	}
	srvD, ok := sess.kern.srvNames[args[0]]
	if !ok {
		sess.println("Unknown service", args[0])
		return true
	}
	kvl := srvD.srv.GetStatistics()
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

func lookupService(sess *cmdSession, cmd string, args []string) (kernel.Service, bool) {
	if len(args) == 0 {
		sess.usage(cmd, "SERVICE")
		return 0, false
	}
	srvD, ok := sess.kern.srvNames[args[0]]
	if !ok {
		sess.println("Unknown service", args[0])
		return 0, false
	}
	return srvD.srvnum, true
}

func cmdMetrics(sess *cmdSession, _ string, _ []string) bool {
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

func cmdDumpIndex(sess *cmdSession, _ string, _ []string) bool {
	sess.kern.DumpIndex(sess.w)
	return true
}
func cmdDumpRecover(sess *cmdSession, cmd string, args []string) bool {
	if len(args) == 0 {
		sess.usage(cmd, "RECOVER")
		sess.println("-- A valid value for RECOVER can be obtained via 'stat core'.")
		return true
	}
	lines := sess.kern.core.RecoverLines(args[0])
	if len(lines) == 0 {
		return true
	}
	for _, line := range lines {
		sess.println(line)
	}
	return true
}

func cmdEnvironment(sess *cmdSession, _ string, _ []string) bool {
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

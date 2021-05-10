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
	"io"
	"sort"
	"strings"

	"zettelstore.de/z/service"
)

type cmdSession struct {
	w    io.Writer
	srv  *myService
	echo bool
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

func (sess *cmdSession) printTable(table [][]string, withHeader bool) {
	maxLen := make([]int, 0)
	for _, row := range table {
		for colno, column := range row {
			if colno >= len(maxLen) {
				maxLen = append(maxLen, 0)
			}
			if len(column) > maxLen[colno] {
				maxLen[colno] = len(column)
			}
		}
	}
	if len(maxLen) == 0 {
		return
	}
	if withHeader {
		sess.printRow(table[0], maxLen)
		for colno := range table[0] {
			if colno > 0 {
				io.WriteString(sess.w, "-+-")
			}
			io.WriteString(sess.w, ljust("", maxLen[colno], '-'))
		}
		io.WriteString(sess.w, "\n")
		table = table[1:]
	}

	for _, row := range table {
		sess.printRow(row, maxLen)
	}
}
func (sess *cmdSession) printRow(row []string, maxLen []int) {
	for colno, column := range row {
		if colno > 0 {
			io.WriteString(sess.w, " | ")
		}
		io.WriteString(sess.w, ljust(column, maxLen[colno], ' '))
	}
	io.WriteString(sess.w, "\n")
}

func ljust(s string, l int, fill byte) string {
	if l <= len(s) {
		return s
	}
	var sb strings.Builder
	sb.Grow(l)
	sb.WriteString(s)
	for i := 0; i < l-len(s); i++ {
		sb.WriteByte(fill)
	}
	return sb.String()
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
		}},
	"list-config": {"list configuration data", cmdListConfig},
}

func cmdHelp(sess *cmdSession, cmd string, args []string) bool {
	cmds := make([]string, 0, len(commands))
	maxLen := 0
	for key := range commands {
		if key == "" {
			continue
		}
		if len(key) > maxLen {
			maxLen = len(key)
		}
		cmds = append(cmds, key)
	}
	sort.Strings(cmds)
	sess.println("Available commands:")
	for _, cmd := range cmds {
		cmdName := cmd
		for len(cmdName) < maxLen {
			cmdName += " "
		}
		sess.println("-", cmdName, commands[cmd].Text)
	}
	return true
}

func cmdListConfig(sess *cmdSession, cmd string, args []string) bool {
	for subsrv := service.SubCore; subsrv <= service.SubWeb; subsrv++ {
		if subsrv > service.SubCore {
			sess.println()
		}
		listSubConfig(sess, subsrv)
	}
	return true
}
func listSubConfig(sess *cmdSession, subsrv service.Subservice) {
	sub := sess.srv.getSubservice(subsrv)
	if sub == nil {
		return
	}
	l := sub.GetConfigList(true)
	table := [][]string{{"Key", "Value", "Description"}}
	for _, kdv := range l {
		table = append(table, []string{kdv.Key, kdv.Value, kdv.Descr})
	}
	sess.printTable(table, true)
}

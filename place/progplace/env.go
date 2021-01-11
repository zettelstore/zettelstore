//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package progplace provides zettel that inform the user about the internal Zettelstore state.
package progplace

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func genEnvironmentM(zid id.Zid) *meta.Meta {
	if myPlace.startConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Environment Values")
	return m
}

func genEnvironmentC(*meta.Meta) string {
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

	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	fmt.Fprintf(&sb, "|Working directory| %v\n", workDir)
	fmt.Fprintf(&sb, "|Executable| %v\n", execName)
	fmt.Fprintf(&sb, "|Build with| %v\n", startup.GetVersion().GoVersion)

	sb.WriteString("=== Environment\n")
	sb.WriteString("|=Key>|=Value<\n")
	for _, env := range envs {
		if pos := strings.IndexByte(env, '='); pos >= 0 && pos < len(env) {
			fmt.Fprintf(&sb, "| %v| %v\n", env[:pos], env[pos+1:])
		} else {
			fmt.Fprintf(&sb, "| %v\n", env)
		}
	}
	return sb.String()
}

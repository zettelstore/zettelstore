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
	"runtime"
	"strings"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func genRuntimeM(zid id.Zid) *meta.Meta {
	if myPlace.startConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Runtime Values")
	return m
}

func genRuntimeC(*meta.Meta) string {
	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	fmt.Fprintf(&sb, "|Number of CPUs|%v\n", runtime.NumCPU())
	fmt.Fprintf(&sb, "|Number of goroutines|%v\n", runtime.NumGoroutine())
	fmt.Fprintf(&sb, "|Number of Cgo calls|%v\n", runtime.NumCgoCall())
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(&sb, "|Memory from OS|%v\n", m.Sys)
	fmt.Fprintf(&sb, "|Objects active|%v\n", m.Mallocs-m.Frees)
	fmt.Fprintf(&sb, "|Heap alloc|%v\n", m.HeapAlloc)
	fmt.Fprintf(&sb, "|Heap sys|%v\n", m.HeapSys)
	fmt.Fprintf(&sb, "|Heap idle|%v\n", m.HeapIdle)
	fmt.Fprintf(&sb, "|Heap in use|%v\n", m.HeapInuse)
	fmt.Fprintf(&sb, "|Heap released|%v\n", m.HeapReleased)
	fmt.Fprintf(&sb, "|Heap objects|%v\n", m.HeapObjects)
	fmt.Fprintf(&sb, "|Stack in use|%v\n", m.StackInuse)
	fmt.Fprintf(&sb, "|Stack sys|%v\n", m.StackSys)
	fmt.Fprintf(&sb, "|Garbage collection metadata|%v\n", m.GCSys)
	fmt.Fprintf(&sb, "|Last garbage collection|%v\n", time.Unix((int64)(m.LastGC/1000000000), 0))
	fmt.Fprintf(&sb, "|Garbage collection goal|%v\n", m.NextGC)
	fmt.Fprintf(&sb, "|Garbage collections|%v\n", m.NumGC)
	fmt.Fprintf(&sb, "|Forced garbage collections|%v\n", m.NumForcedGC)
	fmt.Fprintf(&sb, "|Garbage collection fraction|%.3f%%\n", m.GCCPUFraction*100.0)
	return sb.String()
}

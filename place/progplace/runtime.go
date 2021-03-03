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
	"runtime/metrics"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func genRuntimeM(zid id.Zid) *meta.Meta {
	if myPlace.startConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Runtime Metrics")
	return m
}

func genRuntimeC(*meta.Meta) string {
	var samples []metrics.Sample
	all := metrics.All()
	for _, d := range all {
		if d.Kind == metrics.KindFloat64Histogram {
			continue
		}
		samples = append(samples, metrics.Sample{Name: d.Name})
	}
	metrics.Read(samples)

	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	i := 0
	for _, d := range all {
		if d.Kind == metrics.KindFloat64Histogram {
			continue
		}
		descr := d.Description
		if pos := strings.IndexByte(descr, '.'); pos > 0 {
			descr = descr[:pos]
		}
		fmt.Fprintf(&sb, "|%s|", descr)
		value := samples[i].Value
		i++
		switch value.Kind() {
		case metrics.KindUint64:
			fmt.Fprintf(&sb, "%v", value.Uint64())
		case metrics.KindFloat64:
			fmt.Fprintf(&sb, "%v", value.Float64())
		case metrics.KindFloat64Histogram:
			sb.WriteString("???")
		case metrics.KindBad:
			sb.WriteString("BAD")
		default:
			fmt.Fprintf(&sb, "(unexpected metric kind: %v)", value.Kind())
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

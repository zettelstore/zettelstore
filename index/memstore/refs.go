//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package memstore stored the index in main memory.
package memstore

import (
	"bytes"

	"zettelstore.de/z/domain/id"
)

func refsToString(refs []id.Zid) string {
	var buf bytes.Buffer
	for i, dref := range refs {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.Write(dref.Bytes())
	}
	return buf.String()
}

func refsDiff(refsN, refsO []id.Zid) (newRefs, remRefs []id.Zid) {
	npos, opos := 0, 0
	for npos < len(refsN) && opos < len(refsO) {
		rn, ro := refsN[npos], refsO[opos]
		if rn == ro {
			npos++
			opos++
			continue
		}
		if rn < ro {
			newRefs = append(newRefs, rn)
			npos++
			continue
		}
		remRefs = append(remRefs, ro)
		opos++
	}
	if npos < len(refsN) {
		newRefs = append(newRefs, refsN[npos:]...)
	}
	if opos < len(refsO) {
		remRefs = append(remRefs, refsO[opos:]...)
	}
	return newRefs, remRefs
}

func addRefs(refs []id.Zid, add []id.Zid) []id.Zid {
	result := make([]id.Zid, 0, len(refs)+len(add))
	rpos, apos := 0, 0
	for rpos < len(refs) && apos < len(add) {
		rr, ra := refs[rpos], add[apos]
		if rr < ra {
			result = append(result, rr)
			rpos++
			continue
		}
		if ra < rr {
			result = append(result, ra)
			apos++
			continue
		}
		result = append(result, rr)
		rpos++
		apos++
	}
	if rpos < len(refs) {
		result = append(result, refs[rpos:]...)
	}
	if apos < len(add) {
		result = append(result, add[apos:]...)
	}
	return result
}

func addRef(refs []id.Zid, ref id.Zid) []id.Zid {
	// Too simple
	return addRefs(refs, []id.Zid{ref})
}

func remRefs(refs []id.Zid, rem []id.Zid) []id.Zid {
	result := make([]id.Zid, 0, len(refs)-len(rem))
	rpos, dpos := 0, 0
	for rpos < len(refs) && dpos < len(rem) {
		rr, dr := refs[rpos], rem[dpos]
		if rr < dr {
			result = append(result, rr)
			rpos++
			continue
		}
		if dr < rr {
			dpos++
			continue
		}
		rpos++
		dpos++
	}
	if rpos < len(refs) {
		result = append(result, refs[rpos:]...)
	}
	return result
}

func remRef(refs []id.Zid, ref id.Zid) []id.Zid {
	// Too simple
	return remRefs(refs, []id.Zid{ref})
}

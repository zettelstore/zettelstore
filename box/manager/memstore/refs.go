//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package memstore

import "zettelstore.de/z/domain/id"

func refsDiff(refsN, refsO id.Slice) (newRefs, remRefs id.Slice) {
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

func addRef(refs id.Slice, ref id.Zid) id.Slice {
	hi := len(refs)
	for lo := 0; lo < hi; {
		m := lo + (hi-lo)/2
		if r := refs[m]; r == ref {
			return refs
		} else if r < ref {
			lo = m + 1
		} else {
			hi = m
		}
	}
	refs = append(refs, id.Invalid)
	copy(refs[hi+1:], refs[hi:])
	refs[hi] = ref
	return refs
}

func remRefs(refs, rem id.Slice) id.Slice {
	if len(refs) == 0 || len(rem) == 0 {
		return refs
	}
	result := make(id.Slice, 0, len(refs))
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

func remRef(refs id.Slice, ref id.Zid) id.Slice {
	hi := len(refs)
	for lo := 0; lo < hi; {
		m := lo + (hi-lo)/2
		if r := refs[m]; r == ref {
			copy(refs[m:], refs[m+1:])
			refs = refs[:len(refs)-1]
			return refs
		} else if r < ref {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return refs
}

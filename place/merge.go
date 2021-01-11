//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package place provides a generic interface to zettel places.
package place

import "zettelstore.de/z/domain/meta"

// MergeSorted returns a merged sequence of meta data, sorted by a given Sorter.
// The lists first and second must be sorted descending by Zid.
func MergeSorted(first, second []*meta.Meta) []*meta.Meta {
	lenFirst := len(first)
	lenSecond := len(second)
	result := make([]*meta.Meta, 0, lenFirst+lenSecond)
	iFirst := 0
	iSecond := 0
	for iFirst < lenFirst && iSecond < lenSecond {
		zidFirst := first[iFirst].Zid
		zidSecond := second[iSecond].Zid
		if zidFirst > zidSecond {
			result = append(result, first[iFirst])
			iFirst++
		} else if zidFirst < zidSecond {
			result = append(result, second[iSecond])
			iSecond++
		} else { // zidFirst == zidSecond
			result = append(result, first[iFirst])
			iFirst++
			iSecond++
		}
	}
	if iFirst < lenFirst {
		result = append(result, first[iFirst:]...)
	} else {
		result = append(result, second[iSecond:]...)
	}

	return result
}

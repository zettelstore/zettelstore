//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package index allows to search for metadata and content.
package index

// WordSet contains the set of all words, with the count of their occurrences.
type WordSet map[string]int

// Words gives the slice of all words in the set.
func (ws WordSet) Words() []string {
	if len(ws) == 0 {
		return nil
	}
	words := make([]string, 0, len(ws))
	for w := range ws {
		words = append(words, w)
	}
	return words
}

// Diff calculates the word slice to be added and to be removed from oldWords
// to get the given word set.
func (ws WordSet) Diff(oldWords WordSet) (newWords, removeWords []string) {
	if len(ws) == 0 {
		return nil, oldWords.Words()
	}
	if len(oldWords) == 0 {
		return ws.Words(), nil
	}
	for w := range ws {
		if _, ok := oldWords[w]; ok {
			continue
		}
		newWords = append(newWords, w)
	}
	for ow := range oldWords {
		if _, ok := ws[ow]; ok {
			continue
		}
		removeWords = append(removeWords, ow)
	}
	return newWords, removeWords
}

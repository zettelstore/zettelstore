//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package encoding provides helper functions for encodings.
package encoding

import (
	"bytes"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/parser"
)

// LastUpdated returns the formated time of the zettel which was updated at the latest time.
func LastUpdated(ml []*meta.Meta, timeFormat string) string {
	maxPublished := time.Date(1, time.January, 1, 0, 0, 0, 0, time.Local)
	for _, m := range ml {
		if val, found := m.Get(api.KeyPublished); found {
			if published, err := time.ParseInLocation(id.ZidLayout, val, time.Local); err == nil {
				if maxPublished.Before(published) {
					maxPublished = published
				}
			}
		}
	}
	if maxPublished.Year() > 1 {
		return maxPublished.UTC().Format(timeFormat)
	}
	return ""
}

var textEnc = textenc.Create()

// TitleAsText returns the title of a zettel as plain text
func TitleAsText(m *meta.Meta) string {
	var title bytes.Buffer
	titleIns := parser.ParseMetadata(m.GetTitle())
	if _, err := textEnc.WriteInlines(&title, &titleIns); err != nil {
		return m.GetTitle()
	}
	return title.String()
}
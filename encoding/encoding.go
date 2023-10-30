//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
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
	"time"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// LastUpdated returns the formated time of the zettel which was updated at the latest time.
func LastUpdated(ml []*meta.Meta, timeFormat string) string {
	maxPublished := time.Date(1, time.January, 1, 0, 0, 0, 0, time.Local)
	for _, m := range ml {
		if val, found := m.Get(api.KeyPublished); found {
			if published, err := time.ParseInLocation(id.TimestampLayout, val, time.Local); err == nil {
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

// TitleAsText returns the title of a zettel as plain text
func TitleAsText(m *meta.Meta) string { return parser.NormalizedSpacedText(m.GetTitle()) }

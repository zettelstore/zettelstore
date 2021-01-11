//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/jsonenc"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListTagsHandler creates a new HTTP handler for the use case "list some zettel".
func MakeListTagsHandler(listTags usecase.ListTags) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		iMinCount, _ := strconv.Atoi(r.URL.Query().Get("min"))
		tagData, err := listTags.Run(ctx, iMinCount)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		format := adapter.GetFormat(r, r.URL.Query(), encoder.GetDefaultFormat())
		switch format {
		case "json":
			w.Header().Set("Content-Type", format2ContentType(format))
			renderListTagsJSON(w, tagData)
		default:
			adapter.BadRequest(w, fmt.Sprintf("Tags list not available in format %q", format))
		}
	}
}

func renderListTagsJSON(w http.ResponseWriter, tagData usecase.TagData) {
	buf := encoder.NewBufWriter(w)

	tagList := make([]string, 0, len(tagData))
	for tag := range tagData {
		tagList = append(tagList, tag)
	}
	sort.Strings(tagList)

	buf.WriteString("{\"tags\":{")
	first := true
	for _, tag := range tagList {
		if first {
			buf.WriteByte('"')
			first = false
		} else {
			buf.WriteString(",\"")
		}
		buf.Write(jsonenc.Escape(tag))
		buf.WriteString("\":[")
		for i, meta := range tagData[tag] {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('"')
			buf.WriteString(meta.Zid.String())
			buf.WriteByte('"')
		}
		buf.WriteString("]")

	}
	buf.WriteString("}}")
	buf.Flush()
}

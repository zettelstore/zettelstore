//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"encoding/json"
	"io"
	"net/http"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func encodeJSONData(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

type zettelDataJSON struct {
	Meta     api.ZettelMeta `json:"meta"`
	Encoding string         `json:"encoding"`
	Content  string         `json:"content"`
}

func buildZettelFromJSONData(r *http.Request, zid id.Zid) (zettel.Zettel, error) {
	var zettel zettel.Zettel
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	var zettelData zettelDataJSON
	if err := dec.Decode(&zettelData); err != nil {
		return zettel, err
	}
	m := meta.New(zid)
	for k, v := range zettelData.Meta {
		m.Set(meta.RemoveNonGraphic(k), meta.RemoveNonGraphic(v))
	}
	zettel.Meta = m
	if err := zettel.Content.SetDecoded(zettelData.Content, zettelData.Encoding); err != nil {
		return zettel, err
	}
	return zettel, nil
}

//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func encodeJSONData(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

func (a *API) writeMetaList(ctx context.Context, w http.ResponseWriter, m *meta.Meta, metaList []*meta.Meta) error {
	outList := make([]api.ZidMetaJSON, len(metaList))
	for i, m := range metaList {
		outList[i].ID = api.ZettelID(m.Zid.String())
		outList[i].Meta = m.Map()
		outList[i].Rights = a.getRights(ctx, m)
	}

	var buf bytes.Buffer
	err := encodeJSONData(&buf, api.ZidMetaRelatedList{
		ID:   api.ZettelID(m.Zid.String()),
		Meta: m.Map(),
		List: outList,
	})
	if err != nil {
		a.log.Fatal().Err(err).Zid(m.Zid).Msg("Unable to store meta list in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil
	}

	return writeBuffer(w, &buf, ctJSON)
}

func buildZettelFromJSONData(r *http.Request, zid id.Zid) (domain.Zettel, error) {
	var zettel domain.Zettel
	dec := json.NewDecoder(r.Body)
	var zettelData api.ZettelDataJSON
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

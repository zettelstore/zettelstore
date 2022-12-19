//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/query"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/content"
)

// MakeQueryHandler creates a new HTTP handler to perform a query.
func (a *API) MakeQueryHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		query := adapter.GetQuery(q)

		metaList, err := listMeta.Run(ctx, query)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		var buf bytes.Buffer
		var contentType string
		switch enc, _ := getEncoding(r, q, api.EncoderPlain); enc {
		case api.EncoderPlain:
			for _, m := range metaList {
				_, err = fmt.Fprintln(&buf, m.Zid.String(), m.GetTitle())
				if err != nil {
					a.log.Fatal().Err(err).Msg("Unable to store plain list in buffer")
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
			}
			contentType = content.PlainText

		case api.EncoderJson:
			contentType, err = a.queryAction(ctx, &buf, query, metaList)
			if err != nil {
				a.reportUsecaseError(w, err)
				return
			}

		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		err = writeBuffer(w, &buf, contentType)
		a.log.IfErr(err).Msg("write result buffer")
	}
}

func (a *API) queryAction(ctx context.Context, w io.Writer, q *query.Query, ml []*meta.Meta) (string, error) {
	ap := actionPara{
		w:   w,
		q:   q,
		ml:  ml,
		min: -1,
		max: -1,
	}
	if actions := q.Actions(); len(actions) > 0 {
		acts := make([]string, 0, len(actions))
		for _, act := range actions {
			if strings.HasPrefix(act, "MIN") {
				if num, err := strconv.Atoi(act[3:]); err == nil && num > 0 {
					ap.min = num
					continue
				}
			}
			if strings.HasPrefix(act, "MAX") {
				if num, err := strconv.Atoi(act[3:]); err == nil && num > 0 {
					ap.max = num
					continue
				}
			}
			acts = append(acts, act)
		}
		for _, act := range acts {
			key := strings.ToLower(act)
			switch meta.Type(key) {
			case meta.TypeWord, meta.TypeTagSet:
				return ap.createMapMeta(key)
			}
		}
	}
	err := a.writeQueryMetaList(ctx, w, q, ml)
	return content.JSON, err
}

type actionPara struct {
	w   io.Writer
	q   *query.Query
	ml  []*meta.Meta
	min int
	max int
}

func (ap *actionPara) createMapMeta(key string) (string, error) {
	if len(ap.ml) == 0 {
		return "", nil
	}
	arr := meta.CreateArrangement(ap.ml, key)
	if len(arr) == 0 {
		return "", nil
	}
	min, max := ap.min, ap.max
	mm := make(api.MapMeta, len(arr))
	for tag, metaList := range arr {
		if len(metaList) < min || (max > 0 && len(metaList) > max) {
			continue
		}
		zidList := make([]api.ZettelID, 0, len(metaList))
		for _, m := range metaList {
			zidList = append(zidList, api.ZettelID(m.Zid.String()))
		}
		mm[tag] = zidList
	}

	err := encodeJSONData(ap.w, api.MapListJSON{Map: mm})
	return content.JSON, err
}

func (a *API) writeQueryMetaList(ctx context.Context, w io.Writer, q *query.Query, ml []*meta.Meta) error {
	result := make([]api.ZidMetaJSON, 0, len(ml))
	for _, m := range ml {
		result = append(result, api.ZidMetaJSON{
			ID:     api.ZettelID(m.Zid.String()),
			Meta:   m.Map(),
			Rights: a.getRights(ctx, m),
		})
	}

	err := encodeJSONData(w, api.ZettelListJSON{
		Query: q.String(),
		Human: q.Human(),
		List:  result,
	})
	return err
}

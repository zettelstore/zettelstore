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
	"io"
	"net/http"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListMapMetaHandler creates a new HTTP handler to retrieve mappings of
// metadata values of a specific key to the list of zettel IDs, which contain
// this value.
func (a *API) MakeListMapMetaHandler(listMeta usecase.ListMeta, listRole usecase.ListRoles, listTags usecase.ListTags) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var buf bytes.Buffer
		query := r.URL.Query()
		sea := adapter.GetSearch(r.URL.Query())
		if sea != nil {
			if !sea.EnrichNeeded() {
				ctx = box.NoEnrichContext(ctx)
			}
			metaList, err := listMeta.Run(ctx, sea)
			if err != nil {
				a.reportUsecaseError(w, err)
				return
			}
			contentType, err := actionSearch(&buf, sea, metaList)
			if err != nil {
				a.reportUsecaseError(w, err)
				return
			}
			if buf.Len() > 0 {
				err = writeBuffer(w, &buf, contentType)
				a.log.IfErr(err).Msg("write action")
				return
			}
		}

		minVal := query.Get(api.QueryKeyMin)
		if minVal == "" {
			minVal = query.Get("_min")
		}
		iMinCount, err := strconv.Atoi(minVal)
		if err != nil || iMinCount < 0 {
			iMinCount = 0
		}
		key := query.Get(api.QueryKeyKey)
		if key == "" {
			key = query.Get("_key")
		}

		var ar meta.Arrangement
		switch key {
		case api.KeyRole:
			ar, err = listRole.Run(ctx)
		case api.KeyTags:
			ar, err = listTags.Run(ctx, iMinCount)
		default:
			a.log.Info().Str("key", key).Msg("illegal key for retrieving meta map")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		mm := make(api.MapMeta, len(ar))
		for tag, metaList := range ar {
			zidList := make([]api.ZettelID, 0, len(metaList))
			for _, m := range metaList {
				zidList = append(zidList, api.ZettelID(m.Zid.String()))
			}
			mm[tag] = zidList
		}

		buf.Reset()
		err = encodeJSONData(&buf, api.MapListJSON{Map: mm})
		if err != nil {
			a.log.Fatal().Err(err).Msg("Unable to store map list in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, ctJSON)
		a.log.IfErr(err).Str("key", key).Msg("write meta map")
	}
}

func actionSearch(w io.Writer, sea *search.Search, ml []*meta.Meta) (string, error) {
	ap := actionPara{
		w:   w,
		sea: sea,
		ml:  ml,
		min: -1,
		max: -1,
	}
	if actions := sea.Actions(); len(actions) > 0 {
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
	return "", nil
}

type actionPara struct {
	w   io.Writer
	sea *search.Search
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
	return ctJSON, err
}

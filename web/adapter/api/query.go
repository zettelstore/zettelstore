//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
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
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/z/query"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/content"
	"zettelstore.de/z/zettel/meta"
)

// MakeQueryHandler creates a new HTTP handler to perform a query.
func (a *API) MakeQueryHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		sq := adapter.GetQuery(q)

		metaList, err := listMeta.Run(ctx, sq)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		var encoder zettelEncoder
		var contentType string
		switch enc, _ := getEncoding(r, q); enc {
		case api.EncoderPlain:
			encoder = &plainZettelEncoder{}
			contentType = content.PlainText

		case api.EncoderData:
			encoder = &dataZettelEncoder{
				sf:        sxpf.MakeMappedFactory(),
				sq:        sq,
				getRights: func(m *meta.Meta) api.ZettelRights { return a.getRights(ctx, m) },
			}
			contentType = content.SXPF

		case api.EncoderJson: // DEPRECATED
			encoder = &jsonZettelEncoder{
				sq:        sq,
				getRights: func(m *meta.Meta) api.ZettelRights { return a.getRights(ctx, m) },
			}
			contentType = content.JSON

		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var buf bytes.Buffer
		err = queryAction(&buf, encoder, metaList, sq)
		if err != nil {
			a.log.Error().Err(err).Str("query", sq.String()).Msg("execute query action")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		err = writeBuffer(w, &buf, contentType)
		a.log.IfErr(err).Msg("write result buffer")
	}
}
func queryAction(w io.Writer, enc zettelEncoder, ml []*meta.Meta, sq *query.Query) error {
	min, max := -1, -1
	if actions := sq.Actions(); len(actions) > 0 {
		acts := make([]string, 0, len(actions))
		for _, act := range actions {
			if strings.HasPrefix(act, "MIN") {
				if num, err := strconv.Atoi(act[3:]); err == nil && num > 0 {
					min = num
					continue
				}
			}
			if strings.HasPrefix(act, "MAX") {
				if num, err := strconv.Atoi(act[3:]); err == nil && num > 0 {
					max = num
					continue
				}
			}
			acts = append(acts, act)
		}
		for _, act := range acts {
			switch act {
			case "KEYS":
				return encodeKeysArrangement(w, enc, ml, act)
			}
			switch key := strings.ToLower(act); meta.Type(key) {
			case meta.TypeWord, meta.TypeTagSet:
				return encodeMetaKeyArrangement(w, enc, ml, key, min, max)
			}
		}
	}
	return enc.writeMetaList(w, ml)
}

func encodeKeysArrangement(w io.Writer, enc zettelEncoder, ml []*meta.Meta, act string) error {
	arr := make(meta.Arrangement, 128)
	for _, m := range ml {
		for k := range m.Map() {
			arr[k] = append(arr[k], m)
		}
	}
	return enc.writeArrangement(w, act, arr)
}

func encodeMetaKeyArrangement(w io.Writer, enc zettelEncoder, ml []*meta.Meta, key string, min, max int) error {
	arr0 := meta.CreateArrangement(ml, key)
	arr := make(meta.Arrangement, len(arr0))
	for k0, ml0 := range arr0 {
		if len(ml0) < min || (max > 0 && len(ml0) > max) {
			continue
		}
		arr[k0] = ml0
	}
	return enc.writeArrangement(w, key, arr)
}

type zettelEncoder interface {
	writeMetaList(w io.Writer, ml []*meta.Meta) error
	writeArrangement(w io.Writer, act string, arr meta.Arrangement) error
}

type plainZettelEncoder struct{}

func (*plainZettelEncoder) writeMetaList(w io.Writer, ml []*meta.Meta) error {
	for _, m := range ml {
		_, err := fmt.Fprintln(w, m.Zid.String(), m.GetTitle())
		if err != nil {
			return err
		}
	}
	return nil
}
func (*plainZettelEncoder) writeArrangement(w io.Writer, _ string, arr meta.Arrangement) error {
	for key, ml := range arr {
		_, err := io.WriteString(w, key)
		if err != nil {
			return err
		}
		for i, m := range ml {
			if i == 0 {
				_, err = io.WriteString(w, "\t")
			} else {
				_, err = io.WriteString(w, " ")
			}
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, m.Zid.String())
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

type dataZettelEncoder struct {
	sf        sxpf.SymbolFactory
	sq        *query.Query
	getRights func(*meta.Meta) api.ZettelRights
}

func (dze *dataZettelEncoder) writeMetaList(w io.Writer, ml []*meta.Meta) error {
	sf := dze.sf
	result := make([]sxpf.Object, len(ml)+1)
	result[0] = sf.MustMake("list")
	symID, symZettel := sf.MustMake("id"), sf.MustMake("zettel")
	for i, m := range ml {
		msz := metaRights2sz(m, dze.getRights(m))
		msz = sxpf.Cons(sxpf.MakeList(symID, sxpf.Int64(m.Zid)), msz.Cdr()).Cons(symZettel)
		result[i+1] = msz
	}

	_, err := sxpf.Print(w, sxpf.MakeList(
		sf.MustMake("meta-list"),
		sxpf.MakeList(sf.MustMake("query"), sxpf.MakeString(dze.sq.String())),
		sxpf.MakeList(sf.MustMake("human"), sxpf.MakeString(dze.sq.Human())),
		sxpf.MakeList(result...),
	))
	return err
}
func (dze *dataZettelEncoder) writeArrangement(w io.Writer, act string, arr meta.Arrangement) error {
	sf := dze.sf
	result := sxpf.Nil()
	for aggKey, metaList := range arr {
		sxMeta := sxpf.Nil()
		for i := len(metaList) - 1; i >= 0; i-- {
			sxMeta = sxMeta.Cons(sxpf.Int64(metaList[i].Zid))
		}
		sxMeta = sxMeta.Cons(sxpf.MakeString(aggKey))
		result = result.Cons(sxMeta)
	}
	_, err := sxpf.Print(w, sxpf.MakeList(
		sf.MustMake("aggregate"),
		sxpf.MakeString(act),
		sxpf.MakeList(sf.MustMake("query"), sxpf.MakeString(dze.sq.String())),
		sxpf.MakeList(sf.MustMake("human"), sxpf.MakeString(dze.sq.Human())),
		result.Cons(sf.MustMake("list")),
	))
	return err
}

// jsonZettelEncoder is DEPRECATED
type jsonZettelEncoder struct {
	sq        *query.Query
	getRights func(*meta.Meta) api.ZettelRights
}

func (jze *jsonZettelEncoder) writeMetaList(w io.Writer, ml []*meta.Meta) error {
	result := make([]api.ZidMetaJSON, 0, len(ml))
	for _, m := range ml {
		result = append(result, api.ZidMetaJSON{
			ID:     api.ZettelID(m.Zid.String()),
			Meta:   m.Map(),
			Rights: jze.getRights(m),
		})
	}

	err := encodeJSONData(w, api.ZettelListJSON{
		Query: jze.sq.String(),
		Human: jze.sq.Human(),
		List:  result,
	})
	return err
}
func (*jsonZettelEncoder) writeArrangement(w io.Writer, _ string, arr meta.Arrangement) error {
	mm := make(api.MapMeta, len(arr))
	for key, metaList := range arr {
		zidList := make([]api.ZettelID, 0, len(metaList))
		for _, m := range metaList {
			zidList = append(zidList, api.ZettelID(m.Zid.String()))
		}
		mm[key] = zidList
	}
	return encodeJSONData(w, api.MapListJSON{Map: mm})
}

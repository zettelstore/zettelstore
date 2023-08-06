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
	"bytes"
	"context"
	"fmt"
	"net/http"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/content"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// MakeGetZettelHandler creates a new HTTP handler to return a zettel in various encodings.
func (a *API) MakeGetZettelHandler(getZettel usecase.GetZettel, parseZettel usecase.ParseZettel, evaluate usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		part := getPart(q, partContent)
		ctx := r.Context()
		switch enc, encStr := getEncoding(r, q); enc {
		case api.EncoderPlain:
			a.writePlainData(w, ctx, zid, part, getZettel)

		case api.EncoderData:
			a.writeSzData(w, ctx, zid, part, getZettel)

		case api.EncoderJson:
			a.writeJSONData(w, ctx, zid, part, getZettel)

		default:
			var zn *ast.ZettelNode
			var em func(value string) ast.InlineSlice
			if q.Has(api.QueryKeyParseOnly) {
				zn, err = parseZettel.Run(ctx, zid, q.Get(api.KeySyntax))
				em = parser.ParseMetadata
			} else {
				zn, err = evaluate.Run(ctx, zid, q.Get(api.KeySyntax))
				em = func(value string) ast.InlineSlice {
					return evaluate.RunMetadata(ctx, value)
				}
			}
			if err != nil {
				a.reportUsecaseError(w, err)
				return
			}
			a.writeEncodedZettelPart(w, zn, em, enc, encStr, part)
		}
	}
}

func (a *API) writePlainData(w http.ResponseWriter, ctx context.Context, zid id.Zid, part partType, getZettel usecase.GetZettel) {
	var buf bytes.Buffer
	var contentType string
	var err error

	z, err := getZettel.Run(box.NoEnrichContext(ctx), zid)
	if err != nil {
		a.reportUsecaseError(w, err)
		return
	}

	switch part {
	case partZettel:
		_, err = z.Meta.Write(&buf)
		if err == nil {
			err = buf.WriteByte('\n')
		}
		if err == nil {
			_, err = z.Content.Write(&buf)
		}

	case partMeta:
		contentType = content.PlainText
		_, err = z.Meta.Write(&buf)

	case partContent:
		contentType = content.MIMEFromSyntax(z.Meta.GetDefault(api.KeySyntax, ""))
		_, err = z.Content.Write(&buf)
	}

	if err != nil {
		a.log.Fatal().Err(err).Zid(zid).Msg("Unable to store plain zettel/part in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = writeBuffer(w, &buf, contentType)
	a.log.IfErr(err).Zid(zid).Msg("Write Plain data")
}

func (a *API) writeSzData(w http.ResponseWriter, ctx context.Context, zid id.Zid, part partType, getZettel usecase.GetZettel) {
	z, err := getZettel.Run(ctx, zid)
	if err != nil {
		a.reportUsecaseError(w, err)
		return
	}
	var obj sx.Object
	switch part {
	case partZettel:
		obj = zettel2sz(z, a.getRights(ctx, z.Meta))

	case partMeta:
		m := z.Meta
		obj = metaRights2sz(m, a.getRights(ctx, m))
	}
	err = a.writeObject(w, zid, obj)
	a.log.IfErr(err).Zid(zid).Msg("write sx data")
}

func zettel2sz(z zettel.Zettel, rights api.ZettelRights) sx.Object {
	zContent, encoding := z.Content.Encode()
	sf := sx.MakeMappedFactory()
	return sx.MakeList(
		sf.MustMake("zettel"),
		sx.MakeList(sf.MustMake("id"), sx.MakeString(z.Meta.Zid.String())),
		meta2sz(z.Meta, sf),
		sx.MakeList(sf.MustMake("rights"), sx.Int64(int64(rights))),
		sx.MakeList(sf.MustMake("encoding"), sx.MakeString(encoding)),
		sx.MakeList(sf.MustMake("content"), sx.MakeString(zContent)),
	)
}
func metaRights2sz(m *meta.Meta, rights api.ZettelRights) *sx.Pair {
	sf := sx.MakeMappedFactory()
	return sx.MakeList(
		sf.MustMake("list"),
		meta2sz(m, sf),
		sx.MakeList(sf.MustMake("rights"), sx.Int64(int64(rights))),
	)
}
func meta2sz(m *meta.Meta, sf sx.SymbolFactory) sx.Object {
	result := sx.Nil().Cons(sf.MustMake("meta"))
	curr := result
	for _, p := range m.ComputedPairs() {
		val := sx.MakeList(sf.MustMake(p.Key), sx.MakeString(p.Value))
		curr = curr.AppendBang(val)
	}
	return result
}

type zettelJSON struct {
	ID       api.ZettelID     `json:"id"`
	Meta     api.ZettelMeta   `json:"meta"`
	Encoding string           `json:"encoding"`
	Content  string           `json:"content"`
	Rights   api.ZettelRights `json:"rights"`
}
type zettelMetaJSON struct {
	Meta   api.ZettelMeta   `json:"meta"`
	Rights api.ZettelRights `json:"rights"`
}
type zettelContentJSON struct {
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

func (a *API) writeJSONData(w http.ResponseWriter, ctx context.Context, zid id.Zid, part partType, getZettel usecase.GetZettel) {
	z, err := getZettel.Run(ctx, zid)
	if err != nil {
		a.reportUsecaseError(w, err)
		return
	}

	var buf bytes.Buffer
	switch part {
	case partZettel:
		zContent, encoding := z.Content.Encode()
		err = encodeJSONData(&buf, zettelJSON{
			ID:       api.ZettelID(zid.String()),
			Meta:     z.Meta.Map(),
			Encoding: encoding,
			Content:  zContent,
			Rights:   a.getRights(ctx, z.Meta),
		})

	case partMeta:
		m := z.Meta
		err = encodeJSONData(&buf, zettelMetaJSON{
			Meta:   m.Map(),
			Rights: a.getRights(ctx, m),
		})

	case partContent:
		zContent, encoding := z.Content.Encode()
		err = encodeJSONData(&buf, zettelContentJSON{
			Encoding: encoding,
			Content:  zContent,
		})
	}
	if err != nil {
		a.log.Fatal().Err(err).Zid(zid).Msg("Unable to store zettel in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = writeBuffer(w, &buf, content.JSON)
	a.log.IfErr(err).Zid(zid).Msg("Write JSON data")
}

func (a *API) writeEncodedZettelPart(
	w http.ResponseWriter, zn *ast.ZettelNode,
	evalMeta encoder.EvalMetaFunc,
	enc api.EncodingEnum, encStr string, part partType,
) {
	encdr := encoder.Create(enc)
	if encdr == nil {
		adapter.BadRequest(w, fmt.Sprintf("Zettel %q not available in encoding %q", zn.Meta.Zid, encStr))
		return
	}
	var err error
	var buf bytes.Buffer
	switch part {
	case partZettel:
		_, err = encdr.WriteZettel(&buf, zn, evalMeta)
	case partMeta:
		_, err = encdr.WriteMeta(&buf, zn.InhMeta, evalMeta)
	case partContent:
		_, err = encdr.WriteContent(&buf, zn)
	}
	if err != nil {
		a.log.Fatal().Err(err).Zid(zn.Zid).Msg("Unable to store data in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if buf.Len() == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err = writeBuffer(w, &buf, content.MIMEFromEncoding(enc))
	a.log.IfErr(err).Zid(zn.Zid).Msg("Write Encoded Zettel")
}

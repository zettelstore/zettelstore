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
	"io"
	"net/http"
	"net/url"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/client.fossil/sexp"
	"zettelstore.de/sx.fossil/sxreader"
	"zettelstore.de/z/input"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// getEncoding returns the data encoding selected by the caller.
func getEncoding(r *http.Request, q url.Values) (api.EncodingEnum, string) {
	encoding := q.Get(api.QueryKeyEncoding)
	if encoding != "" {
		return api.Encoder(encoding), encoding
	}
	if enc, ok := getOneEncoding(r, api.HeaderAccept); ok {
		return api.Encoder(enc), enc
	}
	if enc, ok := getOneEncoding(r, api.HeaderContentType); ok {
		return api.Encoder(enc), enc
	}
	return api.EncoderPlain, api.EncoderPlain.String()
}

func getOneEncoding(r *http.Request, key string) (string, bool) {
	if values, ok := r.Header[key]; ok {
		for _, value := range values {
			if enc, ok2 := contentType2encoding(value); ok2 {
				return enc, true
			}
		}
	}
	return "", false
}

var mapCT2encoding = map[string]string{
	"application/json": "json",
	"text/html":        api.EncodingHTML,
}

func contentType2encoding(contentType string) (string, bool) {
	// TODO: only check before first ';'
	enc, ok := mapCT2encoding[contentType]
	return enc, ok
}

type partType int

const (
	_ partType = iota
	partMeta
	partContent
	partZettel
)

var partMap = map[string]partType{
	api.PartMeta:    partMeta,
	api.PartContent: partContent,
	api.PartZettel:  partZettel,
}

func getPart(q url.Values, defPart partType) partType {
	if part, ok := partMap[q.Get(api.QueryKeyPart)]; ok {
		return part
	}
	return defPart
}

func (p partType) String() string {
	switch p {
	case partMeta:
		return "meta"
	case partContent:
		return "content"
	case partZettel:
		return "zettel"
	}
	return ""
}

func (p partType) DefString(defPart partType) string {
	if p == defPart {
		return ""
	}
	return p.String()
}

func buildZettelFromPlainData(r *http.Request, zid id.Zid) (zettel.Zettel, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return zettel.Zettel{}, err
	}
	inp := input.NewInput(b)
	m := meta.NewFromInput(zid, inp)
	return zettel.Zettel{
		Meta:    m,
		Content: zettel.NewContent(inp.Src[inp.Pos:]),
	}, nil
}

func buildZettelFromData(r *http.Request, zid id.Zid) (zettel.Zettel, error) {
	defer r.Body.Close()
	rdr := sxreader.MakeReader(r.Body)
	obj, err := rdr.Read()
	if err != nil {
		return zettel.Zettel{}, err
	}
	zd, err := sexp.ParseZettel(obj)
	if err != nil {
		return zettel.Zettel{}, err
	}

	m := meta.New(zid)
	for k, v := range zd.Meta {
		if !meta.IsComputed(k) {
			m.Set(meta.RemoveNonGraphic(k), meta.RemoveNonGraphic(v))
		}
	}

	var content zettel.Content
	if err = content.SetDecoded(zd.Content, zd.Encoding); err != nil {
		return zettel.Zettel{}, err
	}

	return zettel.Zettel{
		Meta:    m,
		Content: content,
	}, nil
}

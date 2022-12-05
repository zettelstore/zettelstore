//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/web/content"
)

type formZettelData struct {
	Heading       string
	MetaTitle     string
	MetaRole      string
	HasRoleData   bool
	RoleData      []string
	HasSyntaxData bool
	SyntaxData    []string
	MetaTags      string
	MetaSyntax    string
	MetaPairsRest []meta.Pair
	IsTextContent bool
	Content       string
}

var (
	bsCRLF = []byte{'\r', '\n'}
	bsLF   = []byte{'\n'}
)

func parseZettelForm(r *http.Request, zid id.Zid) (bool, domain.Zettel, bool, error) {
	maxRequestSize := kernel.Main.GetConfig(kernel.WebService, kernel.WebMaxRequestSize).(int64)
	err := r.ParseMultipartForm(maxRequestSize)
	if err != nil {
		return false, domain.Zettel{}, false, err
	}
	_, doSave := r.Form["save"]

	var m *meta.Meta
	if postMeta, ok := trimmedFormValue(r, "meta"); ok {
		m = meta.NewFromInput(zid, input.NewInput(removeEmptyLines([]byte(postMeta))))
		m.Sanitize()
	} else {
		m = meta.New(zid)
	}
	if postTitle, ok := trimmedFormValue(r, "title"); ok {
		m.Set(api.KeyTitle, meta.RemoveNonGraphic(postTitle))
	}
	if postTags, ok := trimmedFormValue(r, "tags"); ok {
		if tags := strings.Fields(meta.RemoveNonGraphic(postTags)); len(tags) > 0 {
			m.SetList(api.KeyTags, tags)
		}
	}
	if postRole, ok := trimmedFormValue(r, "role"); ok {
		m.Set(api.KeyRole, meta.RemoveNonGraphic(postRole))
	}
	if postSyntax, ok := trimmedFormValue(r, "syntax"); ok {
		m.Set(api.KeySyntax, meta.RemoveNonGraphic(postSyntax))
	}

	if b := bytesContent(r, m); b != nil {
		return doSave, domain.Zettel{Meta: m, Content: domain.NewContent(b)}, true, nil
	}
	file, fh, err := r.FormFile("file")
	if file != nil {
		defer file.Close()
		if err == nil {
			var data []byte
			data, m = fileContent(file, fh, m)
			if data != nil {
				return doSave, domain.Zettel{Meta: m, Content: domain.NewContent(data)}, true, nil
			}
		}
	}
	return doSave, domain.Zettel{Meta: m, Content: domain.NewContent(nil)}, false, nil
}

func trimmedFormValue(r *http.Request, key string) (string, bool) {
	if values, ok := r.PostForm[key]; ok && len(values) > 0 {
		value := strings.TrimSpace(values[0])
		if len(value) > 0 {
			return value, true
		}
	}
	return "", false
}

func bytesContent(r *http.Request, m *meta.Meta) []byte {
	if syntax, found := m.Get(api.KeySyntax); found {
		if pinfo := parser.Get(syntax); pinfo != nil {
			if !pinfo.IsTextFormat {
				return nil
			}
		}
	}
	if values, found := r.PostForm["content"]; found && len(values) > 0 {
		result := bytes.ReplaceAll([]byte(values[0]), bsCRLF, bsLF)
		if len(result) == 0 {
			return []byte("")
		}
		if bytes.IndexFunc(result, func(ch rune) bool { return !unicode.IsSpace(ch) }) < 0 {
			return nil
		}
		return result
	}
	return nil
}

func fileContent(r io.Reader, fh *multipart.FileHeader, m *meta.Meta) ([]byte, *meta.Meta) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, m
	}
	if cts, found := fh.Header["Content-Type"]; found && len(cts) > 0 {
		ct := cts[0]
		if fileSyntax := content.SyntaxFromMIME(ct, data); fileSyntax != "" {
			m = m.Clone()
			m.Set(api.KeySyntax, fileSyntax)
		}
	}

	return data, m
}

var reEmptyLines = regexp.MustCompile(`(\n|\r)+\s*(\n|\r)+`)

func removeEmptyLines(s []byte) []byte {
	b := bytes.TrimSpace(s)
	return reEmptyLines.ReplaceAllLiteral(b, []byte{'\n'})
}

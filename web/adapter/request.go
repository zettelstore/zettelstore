//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package adapter

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/search"
	"zettelstore.de/z/usecase"
)

// GetCredentialsViaForm retrieves the authentication credentions from a form.
func GetCredentialsViaForm(r *http.Request) (ident, cred string, ok bool) {
	err := r.ParseForm()
	if err != nil {
		kernel.Main.GetLogger(kernel.WebService).Info().Err(err).Msg("Unable to parse form")
		return "", "", false
	}

	ident = strings.TrimSpace(r.PostFormValue("username"))
	cred = r.PostFormValue("password")
	if ident == "" {
		return "", "", false
	}
	return ident, cred, true
}

// GetInteger returns the integer value of the named query key.
func GetInteger(q url.Values, key string) (int, bool) {
	s := q.Get(key)
	if s != "" {
		if val, err := strconv.Atoi(s); err == nil {
			return val, true
		}
	}
	return 0, false
}

// GetSearch retrieves the specified search and sorting options from a query.
func GetSearch(q url.Values) *search.Search {
	if exprs, found := q[api.QueryKeySearch]; found {
		return search.Parse(strings.Join(exprs, " "))
	}
	return nil
}

// GetZCDirection returns a direction value for a given string.
func GetZCDirection(s string) usecase.ZettelContextDirection {
	switch s {
	case api.DirBackward:
		return usecase.ZettelContextBackward
	case api.DirForward:
		return usecase.ZettelContextForward
	}
	return usecase.ZettelContextBoth
}

// AddUnlinkedRefsToSearch inspects metadata and enhances the given search to ignore
// some zettel identifier.
func AddUnlinkedRefsToSearch(s *search.Search, m *meta.Meta) *search.Search {
	s = s.AddExpr(api.KeyID, "!="+m.Zid.String())
	for _, pair := range m.ComputedPairsRest() {
		switch meta.Type(pair.Key) {
		case meta.TypeID:
			s = s.AddExpr(api.KeyID, "!="+pair.Value)
		case meta.TypeIDSet:
			for _, value := range meta.ListFromValue(pair.Value) {
				s = s.AddExpr(api.KeyID, "!="+value)
			}
		}
	}
	return s
}

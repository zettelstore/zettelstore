//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
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

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel/meta"
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

// GetQuery retrieves the specified options from a query.
func GetQuery(vals url.Values) (result *query.Query) {
	if exprs, found := vals[api.QueryKeyQuery]; found {
		result = query.Parse(strings.Join(exprs, " "))
	}
	if seeds, found := vals[api.QueryKeySeed]; found {
		for _, seed := range seeds {
			if si, err := strconv.ParseInt(seed, 10, 31); err == nil {
				result = result.SetSeed(int(si))
				break
			}
		}
	}
	return result
}

// AddUnlinkedRefsToQuery inspects metadata and enhances the given query to ignore
// some zettel identifier.
func AddUnlinkedRefsToQuery(q *query.Query, m *meta.Meta) *query.Query {
	var sb strings.Builder
	sb.WriteString(api.KeyID)
	sb.WriteString("!:")
	sb.WriteString(m.Zid.String())
	for _, pair := range m.ComputedPairsRest() {
		switch meta.Type(pair.Key) {
		case meta.TypeID:
			sb.WriteByte(' ')
			sb.WriteString(api.KeyID)
			sb.WriteString("!:")
			sb.WriteString(pair.Value)
		case meta.TypeIDSet:
			for _, value := range meta.ListFromValue(pair.Value) {
				sb.WriteByte(' ')
				sb.WriteString(api.KeyID)
				sb.WriteString("!:")
				sb.WriteString(value)
			}
		}
	}
	return q.Parse(sb.String())
}

//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package query

import (
	"context"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/zettel/meta"
)

type identSpec struct{}

func (spec *identSpec) printToEnv(pe *printEnv) {
	pe.printSpace()
	pe.writeString(api.IdentDirective)
}

func (spec *identSpec) retrieve(_ context.Context, startSeq []*meta.Meta, _ MetaMatchFunc, _ GetMetaFunc, _ SelectMetaFunc) ([]*meta.Meta, error) {
	return startSeq, nil
}

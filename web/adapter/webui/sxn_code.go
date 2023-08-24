//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"context"
	"io"

	"zettelstore.de/sx.fossil/sxeval"
	"zettelstore.de/z/zettel/id"
)

func (wui *WebUI) loadSxnCodeZettel(ctx context.Context, zid id.Zid, env sxeval.Environment) error {
	rdr, err := wui.makeZettelReader(ctx, zid)
	if err != nil {
		return err
	}
	for {
		form, err2 := rdr.Read()
		if err2 != nil {
			if err2 == io.EOF {
				return nil
			}
			return err2
		}
		wui.log.Trace().Zid(zid).Str("form", form.Repr()).Msg("Loaded sxn code")

		if _, err2 = wui.engine.Eval(env, form); err2 != nil {
			return err2
		}
	}
}

func (wui *WebUI) loadAllSxnCodeZettel(ctx context.Context) (sxeval.Environment, error) {
	zettelEnv := sxeval.MakeChildEnvironment(wui.engine.RootEnvironment(), "zettel", 128)
	if err := wui.loadSxnCodeZettel(ctx, id.BaseSxnZid, zettelEnv); err != nil {
		return nil, err
	}
	if err := wui.loadSxnCodeZettel(ctx, id.StartSxnZid, zettelEnv); err != nil {
		return nil, err
	}
	return zettelEnv, nil
}

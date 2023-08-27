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
	"fmt"
	"io"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil/sxeval"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func (wui *WebUI) loadAllSxnCodeZettel(ctx context.Context) (id.Digraph, sxeval.Environment, error) {
	dg := buildSxnCodeDigraph(ctx, id.StartSxnZid, id.BaseSxnZid, wui.box.GetMeta)
	if dg == nil {
		return nil, wui.engine.RootEnvironment(), nil
	}
	if zid, isDAG := dg.IsDAG(); !isDAG {
		return nil, nil, fmt.Errorf("zettel %v is part of a dependency cycle", zid)
	}
	env := sxeval.MakeChildEnvironment(wui.engine.RootEnvironment(), "zettel", 128)
	for _, zid := range dg.SortReverse() {
		if err := wui.loadSxnCodeZettel(ctx, zid, env); err != nil {
			return nil, nil, err
		}
	}
	return dg, env, nil
}

type getMetaFunc func(context.Context, id.Zid) (*meta.Meta, error)

func buildSxnCodeDigraph(ctx context.Context, startZid, baseZid id.Zid, getMeta getMetaFunc) id.Digraph {
	m, err := getMeta(ctx, startZid)
	if err != nil {
		return nil
	}
	var marked id.Set
	stack := []*meta.Meta{m}
	dg := id.Digraph(nil).AddVertex(startZid)
	for pos := len(stack) - 1; pos >= 0; pos = len(stack) - 1 {
		curr := stack[pos]
		stack = stack[:pos]
		if marked.Contains(curr.Zid) {
			continue
		}
		marked.Add(curr.Zid)
		if precursors, hasPrecursor := curr.GetList(api.KeyPrecursor); hasPrecursor && len(precursors) > 0 {
			for _, pre := range precursors {
				if preZid, errParse := id.Parse(pre); errParse == nil {
					m, err = getMeta(ctx, preZid)
					if err != nil {
						continue
					}
					stack = append(stack, m)
					dg.AddVertex(preZid)
					dg.AddEdge(curr.Zid, preZid)
				}
			}
		}
	}
	dg = dg.AddVertex(baseZid)
	dg = dg.AddEdge(startZid, baseZid)
	return dg.TransitiveClosure(startZid)
}

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
		wui.log.Debug().Zid(zid).Str("form", form.Repr()).Msg("Loaded sxn code")

		if _, err2 = wui.engine.Eval(env, form); err2 != nil {
			return err2
		}
	}
}

//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"

	"zettelstore.de/z/api"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

// ---------- Subcommand: file -----------------------------------------------

func cmdFile(fs *flag.FlagSet, cfg *meta.Meta) (int, error) {
	enc := fs.Lookup("t").Value.String()
	m, inp, err := getInput(fs.Args())
	if m == nil {
		return 2, err
	}
	z := parser.ParseZettel(
		domain.Zettel{
			Meta:    m,
			Content: domain.NewContent(inp.Src[inp.Pos:]),
		},
		m.GetDefault(meta.KeySyntax, meta.ValueSyntaxZmk),
		nil,
	)
	encdr := encoder.Create(api.Encoder(enc), &encoder.Environment{Lang: m.GetDefault(meta.KeyLang, meta.ValueLangEN)})
	if encdr == nil {
		fmt.Fprintf(os.Stderr, "Unknown encoder %q\n", enc)
		return 2, nil
	}
	_, err = encdr.WriteZettel(os.Stdout, z, enc != "raw")
	if err != nil {
		return 2, err
	}
	fmt.Println()

	return 0, nil
}

func getInput(args []string) (*meta.Meta, *input.Input, error) {
	if len(args) < 1 {
		src, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, nil, err
		}
		inp := input.NewInput(string(src))
		m := meta.NewFromInput(id.New(true), inp)
		return m, inp, nil
	}

	src, err := os.ReadFile(args[0])
	if err != nil {
		return nil, nil, err
	}
	inp := input.NewInput(string(src))
	m := meta.NewFromInput(id.New(true), inp)

	if len(args) > 1 {
		src, err := os.ReadFile(args[1])
		if err != nil {
			return nil, nil, err
		}
		inp = input.NewInput(string(src))
	}
	return m, inp, nil
}

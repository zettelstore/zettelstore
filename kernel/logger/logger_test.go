//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package logger_test

import (
	"io"
	"os"
	"testing"

	"zettelstore.de/z/kernel/logger"
)

func TestParseLevel(t *testing.T) {
	testcases := []struct {
		text string
		exp  logger.Level
	}{
		{"tra", logger.TraceLevel},
		{"deb", logger.DebugLevel},
		{"info", logger.InfoLevel},
		{"warn", logger.WarnLevel},
		{"err", logger.ErrorLevel},
		{"fata", logger.FatalLevel},
		{"pan", logger.PanicLevel},
		{"manda", logger.MandatoryLevel},
		{"dis", logger.NeverLevel},
		{"d", logger.Level(0)},
	}
	for i, tc := range testcases {
		got := logger.ParseLevel(tc.text)
		if got != tc.exp {
			t.Errorf("%d: ParseLevel(%q) == %q, but got %q", i, tc.text, tc.exp, got)
		}
	}
}

func BenchmarkDisabled(b *testing.B) {
	log := logger.New(os.Stderr).SetLevel(logger.NeverLevel)
	for n := 0; n < b.N; n++ {
		log.Info().Str("key", "val").Msg("Benchmark")
	}
}

func BenchmarkStrMessage(b *testing.B) {
	log := logger.New(io.Discard)
	for n := 0; n < b.N; n++ {
		log.Info().Str("key", "val").Msg("Benchmark")
	}
}

func BenchmarkStrNoMessage(b *testing.B) {
	log := logger.New(io.Discard)
	for n := 0; n < b.N; n++ {
		log.Info().Str("key", "val").Msg("")
	}
}

func BenchmarkMessage(b *testing.B) {
	log := logger.New(io.Discard)
	for n := 0; n < b.N; n++ {
		log.Info().Msg("Benchmark")
	}
}

func BenchmarkNoMessage(b *testing.B) {
	log := logger.New(io.Discard)
	for n := 0; n < b.N; n++ {
		log.Info().Msg("")
	}
}

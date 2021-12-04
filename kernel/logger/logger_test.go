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

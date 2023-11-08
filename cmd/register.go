//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package cmd provides command generic functions.
package cmd

// Mention all needed encoders, parsers and stores to have them registered.
import (
	_ "zettelstore.de/z/box/compbox"       // Allow to use computed box.
	_ "zettelstore.de/z/box/constbox"      // Allow to use global internal box.
	_ "zettelstore.de/z/box/dirbox"        // Allow to use directory box.
	_ "zettelstore.de/z/box/filebox"       // Allow to use file box.
	_ "zettelstore.de/z/box/membox"        // Allow to use in-memory box.
	_ "zettelstore.de/z/encoder/htmlenc"   // Allow to use HTML encoder.
	_ "zettelstore.de/z/encoder/htmlsenc"  // Allow to use HTMLS encoder.
	_ "zettelstore.de/z/encoder/mdenc"     // Allow to use markdown encoder.
	_ "zettelstore.de/z/encoder/shtmlenc"  // Allow to use SHTML encoder.
	_ "zettelstore.de/z/encoder/szenc"     // Allow to use Sz encoder.
	_ "zettelstore.de/z/encoder/textenc"   // Allow to use text encoder.
	_ "zettelstore.de/z/encoder/zmkenc"    // Allow to use zmk encoder.
	_ "zettelstore.de/z/kernel/impl"       // Allow kernel implementation to create itself
	_ "zettelstore.de/z/parser/blob"       // Allow to use BLOB parser.
	_ "zettelstore.de/z/parser/draw"       // Allow to use draw parser.
	_ "zettelstore.de/z/parser/markdown"   // Allow to use markdown parser.
	_ "zettelstore.de/z/parser/none"       // Allow to use none parser.
	_ "zettelstore.de/z/parser/plain"      // Allow to use plain parser.
	_ "zettelstore.de/z/parser/zettelmark" // Allow to use zettelmark parser.
)

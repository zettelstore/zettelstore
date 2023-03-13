//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder/textenc"
)

func encodeEvaluatedTitleHTML(m *meta.Meta, evalMetadata evalMetadataFunc, gen *htmlGenerator) string {
	is := evalMetadata(m.GetTitle())
	return gen.InlinesString(&is)
}

func encodeEvaluatedTitleText(m *meta.Meta, evalMetadata evalMetadataFunc, enc *textenc.Encoder) string {
	is := evalMetadata(m.GetTitle())
	result, err := encodeInlinesText(&is, enc)
	if err != nil {
		return err.Error()
	}
	return result

}

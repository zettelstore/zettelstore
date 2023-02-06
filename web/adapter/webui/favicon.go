//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"zettelstore.de/z/web/adapter"
)

func (wui *WebUI) MakeFaviconHandler(baseDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Join(baseDir, "favicon.ico")
		f, err := os.Open(filename)
		if err != nil {
			wui.log.Sense().Err(err).Msg("Favicon not found")
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			wui.log.Info().Err(err).Msg("Unable to read favicon data")
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		err = adapter.WriteData(w, data, "")
		wui.log.IfErr(err).Msg("Write favicon")
	}
}

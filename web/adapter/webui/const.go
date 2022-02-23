//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

// WebUI related constants.

const queryKeyAction = "action"

// Values for queryKeyAction
const (
	valueActionCopy  = "copy"
	valueActionFolge = "folge"
	valueActionNew   = "new"
)

// Enumeration for queryKeyAction
type createAction uint8

const (
	actionCopy createAction = iota
	actionFolge
	actionNew
)

func getCreateAction(s string) createAction {
	switch s {
	case valueActionCopy:
		return actionCopy
	case valueActionFolge:
		return actionFolge
	case valueActionNew:
		return actionNew
	default:
		return actionCopy
	}
}

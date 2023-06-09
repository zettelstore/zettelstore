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

// WebUI related constants.

const queryKeyAction = "action"

// Values for queryKeyAction
const (
	valueActionChild   = "child"
	valueActionCopy    = "copy"
	valueActionFolge   = "folge"
	valueActionNew     = "new"
	valueActionVersion = "version"
)

// Enumeration for queryKeyAction
type createAction uint8

const (
	actionChild createAction = iota
	actionCopy
	actionFolge
	actionNew
	actionVersion
)

var createActionMap = map[string]createAction{
	valueActionChild:   actionChild,
	valueActionCopy:    actionCopy,
	valueActionFolge:   actionFolge,
	valueActionNew:     actionNew,
	valueActionVersion: actionVersion,
}

func getCreateAction(s string) createAction {
	if action, found := createActionMap[s]; found {
		return action
	}
	return actionCopy
}

//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides services for authentification / authorization.
package impl

import (
	"errors"
	"hash/fnv"
	"io"
	"time"

	"codeberg.org/t73fde/sxpf"

	"zettelstore.de/c/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/auth/policy"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

type myAuth struct {
	readonly bool
	owner    id.Zid
	secret   []byte
}

// New creates a new auth object.
func New(readonly bool, owner id.Zid, extSecret string) auth.Manager {
	return &myAuth{
		readonly: readonly,
		owner:    owner,
		secret:   calcSecret(extSecret),
	}
}

var configKeys = []string{
	kernel.CoreProgname,
	kernel.CoreGoVersion,
	kernel.CoreHostname,
	kernel.CoreGoOS,
	kernel.CoreGoArch,
	kernel.CoreVersion,
}

func calcSecret(extSecret string) []byte {
	h := fnv.New128()
	if extSecret != "" {
		io.WriteString(h, extSecret)
	}
	for _, key := range configKeys {
		io.WriteString(h, kernel.Main.GetConfig(kernel.CoreService, key).(string))
	}
	return h.Sum(nil)
}

// IsReadonly returns true, if the systems is configured to run in read-only-mode.
func (a *myAuth) IsReadonly() bool { return a.readonly }

// ErrMalformedToken signals a broken token.
var ErrMalformedToken = errors.New("auth: malformed token")

// ErrNoIdent signals that the 'ident' key is missing.
var ErrNoIdent = errors.New("auth: missing ident")

// ErrOtherKind signals that the token was defined for another token kind.
var ErrOtherKind = errors.New("auth: wrong token kind")

// ErrNoZid signals that the 'zid' key is missing.
var ErrNoZid = errors.New("auth: missing zettel id")

// GetToken returns a token to be used for authentification.
func (a *myAuth) GetToken(ident *meta.Meta, d time.Duration, kind auth.TokenKind) ([]byte, error) {
	subject, ok := ident.Get(api.KeyUserID)
	if !ok || subject == "" {
		return nil, ErrNoIdent
	}

	now := time.Now().Round(time.Second)
	sClaim := sxpf.MakeList(
		sxpf.Int64(kind),
		sxpf.MakeString(subject),
		sxpf.Int64(now.Unix()),
		sxpf.Int64(now.Add(d).Unix()),
		sxpf.Int64(ident.Zid),
	)
	return sign(sClaim, a.secret)
}

// ErrTokenExpired signals an exired token
var ErrTokenExpired = errors.New("auth: token expired")

// CheckToken checks the validity of the token and returns relevant data.
func (a *myAuth) CheckToken(tok []byte, k auth.TokenKind) (auth.TokenData, error) {
	var tokenData auth.TokenData

	obj, err := check(tok, a.secret)
	if err != nil {
		return tokenData, err
	}

	tokenData.Token = tok
	err = setupTokenData(obj, k, &tokenData)
	return tokenData, err
}

func setupTokenData(obj sxpf.Object, k auth.TokenKind, tokenData *auth.TokenData) error {
	cell, isCell := sxpf.GetCell(obj)
	if !isCell || cell == nil {
		return ErrMalformedToken
	}

	if sKind, isInt64 := cell.Car().(sxpf.Int64); isInt64 {
		if auth.TokenKind(sKind) != k {
			return ErrOtherKind
		}
	} else {
		return ErrMalformedToken
	}

	cell = cell.Tail()
	ident, isString := sxpf.GetString(cell.Car())
	if !isString {
		return ErrMalformedToken
	}
	if ident == "" {
		return ErrNoIdent
	}

	cell = cell.Tail()
	unixIssued, isInt64 := cell.Car().(sxpf.Int64)
	if !isInt64 {
		return ErrMalformedToken
	}

	cell = cell.Tail()
	unixExpires, isInisInt64 := cell.Car().(sxpf.Int64)
	if !isInisInt64 {
		return ErrMalformedToken
	}
	now := time.Now().Round(time.Second)
	expires := time.Unix(int64(unixExpires), 0)
	if expires.Before(now) {
		return ErrTokenExpired
	}

	cell = cell.Tail()
	sZid, isInt64 := cell.Car().(sxpf.Int64)
	if !isInt64 || sZid < 0 {
		return ErrMalformedToken
	}
	zid := id.Zid(sZid)
	if !zid.IsValid() {
		return ErrNoZid
	}

	tokenData.Ident = ident.String()
	tokenData.Issued = time.Unix(int64(unixIssued), 0)
	tokenData.Now = now
	tokenData.Expires = expires
	tokenData.Zid = zid
	return nil
}

func (a *myAuth) Owner() id.Zid { return a.owner }

func (a *myAuth) IsOwner(zid id.Zid) bool {
	return zid.IsValid() && zid == a.owner
}

func (a *myAuth) WithAuth() bool { return a.owner != id.Invalid }

// GetUserRole role returns the user role of the given user zettel.
func (a *myAuth) GetUserRole(user *meta.Meta) meta.UserRole {
	if user == nil {
		if a.WithAuth() {
			return meta.UserRoleUnknown
		}
		return meta.UserRoleOwner
	}
	if a.IsOwner(user.Zid) {
		return meta.UserRoleOwner
	}
	if val, ok := user.Get(api.KeyUserRole); ok {
		if ur := meta.GetUserRole(val); ur != meta.UserRoleUnknown {
			return ur
		}
	}
	return meta.UserRoleReader
}

func (a *myAuth) BoxWithPolicy(unprotectedBox box.Box, rtConfig config.Config) (box.Box, auth.Policy) {
	return policy.BoxWithPolicy(a, unprotectedBox, rtConfig)
}

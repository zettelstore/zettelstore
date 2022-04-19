//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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

	"github.com/pascaldekloe/jwt"

	"zettelstore.de/c/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/auth/policy"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/web/server"
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

const reqHash = jwt.HS512

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
	claims := jwt.Claims{
		Registered: jwt.Registered{
			Subject: subject,
			Expires: jwt.NewNumericTime(now.Add(d)),
			Issued:  jwt.NewNumericTime(now),
		},
		Set: map[string]interface{}{
			"zid": ident.Zid.String(),
			"_tk": int(kind),
		},
	}
	token, err := claims.HMACSign(reqHash, a.secret)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// ErrTokenExpired signals an exired token
var ErrTokenExpired = errors.New("auth: token expired")

// CheckToken checks the validity of the token and returns relevant data.
func (a *myAuth) CheckToken(token []byte, k auth.TokenKind) (auth.TokenData, error) {
	h, err := jwt.NewHMAC(reqHash, a.secret)
	if err != nil {
		return auth.TokenData{}, err
	}
	claims, err := h.Check(token)
	if err != nil {
		return auth.TokenData{}, err
	}
	now := time.Now().Round(time.Second)
	expires := claims.Expires.Time()
	if expires.Before(now) {
		return auth.TokenData{}, ErrTokenExpired
	}
	ident := claims.Subject
	if ident == "" {
		return auth.TokenData{}, ErrNoIdent
	}
	if zidS, ok := claims.Set["zid"].(string); ok {
		if zid, err2 := id.Parse(zidS); err2 == nil {
			if kind, ok2 := claims.Set["_tk"].(float64); ok2 {
				if auth.TokenKind(kind) == k {
					return auth.TokenData{
						Token:   token,
						Now:     now,
						Issued:  claims.Issued.Time(),
						Expires: expires,
						Ident:   ident,
						Zid:     zid,
					}, nil
				}
			}
			return auth.TokenData{}, ErrOtherKind
		}
	}
	return auth.TokenData{}, ErrNoZid
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

func (a *myAuth) BoxWithPolicy(auth server.Auth, unprotectedBox box.Box, rtConfig config.Config) (box.Box, auth.Policy) {
	return policy.BoxWithPolicy(auth, a, unprotectedBox, rtConfig)
}

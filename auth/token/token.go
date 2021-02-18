//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package token provides some function for handling auth token.
package token

import (
	"errors"
	"time"

	"github.com/pascaldekloe/jwt"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

const reqHash = jwt.HS512

// ErrNoUser signals that the meta data has no role value 'user'.
var ErrNoUser = errors.New("auth: meta is no user")

// ErrNoIdent signals that the 'ident' key is missing.
var ErrNoIdent = errors.New("auth: missing ident")

// ErrOtherKind signals that the token was defined for another token kind.
var ErrOtherKind = errors.New("auth: wrong token kind")

// ErrNoZid signals that the 'zid' key is missing.
var ErrNoZid = errors.New("auth: missing zettel id")

// Kind specifies for which application / usage a token is/was requested.
type Kind int

// Allowed values of token kind
const (
	_ Kind = iota
	KindJSON
	KindHTML
)

// GetToken returns a token to be used for authentification.
func GetToken(ident *meta.Meta, d time.Duration, kind Kind) ([]byte, error) {
	if role, ok := ident.Get(meta.KeyRole); !ok || role != meta.ValueRoleUser {
		return nil, ErrNoUser
	}
	subject, ok := ident.Get(meta.KeyUserID)
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
	token, err := claims.HMACSign(reqHash, startup.Secret())
	if err != nil {
		return nil, err
	}
	return token, nil
}

// ErrTokenExpired signals an exired token
var ErrTokenExpired = errors.New("auth: token expired")

// Data contains some important elements from a token.
type Data struct {
	Token   []byte
	Now     time.Time
	Issued  time.Time
	Expires time.Time
	Ident   string
	Zid     id.Zid
}

// CheckToken checks the validity of the token and returns relevant data.
func CheckToken(token []byte, k Kind) (Data, error) {
	h, err := jwt.NewHMAC(reqHash, startup.Secret())
	if err != nil {
		return Data{}, err
	}
	claims, err := h.Check(token)
	if err != nil {
		return Data{}, err
	}
	now := time.Now().Round(time.Second)
	expires := claims.Expires.Time()
	if expires.Before(now) {
		return Data{}, ErrTokenExpired
	}
	ident := claims.Subject
	if ident == "" {
		return Data{}, ErrNoIdent
	}
	if zidS, ok := claims.Set["zid"].(string); ok {
		if zid, err := id.Parse(zidS); err == nil {
			if kind, ok := claims.Set["_tk"].(float64); ok {
				if Kind(kind) == k {
					return Data{
						Token:   token,
						Now:     now,
						Issued:  claims.Issued.Time(),
						Expires: expires,
						Ident:   ident,
						Zid:     zid,
					}, nil
				}
			}
			return Data{}, ErrOtherKind
		}
	}
	return Data{}, ErrNoZid
}

//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package cred provides some function for handling credentials.
package cred

import (
	"bytes"
	"time"

	"golang.org/x/crypto/bcrypt"
	"zettelstore.de/z/domain/id"
)

// HashCredential returns a hashed vesion of the given credential
func HashCredential(zid id.Zid, ident string, credential string) (string, error) {
	fullCredential := createFullCredential(zid, ident, credential)
	res, err := bcrypt.GenerateFromPassword(fullCredential, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

const (
	durMinWait = 500 * time.Millisecond // minimum wait time after login
	minSleep   = 100 * time.Millisecond // minimum sleep time, even for slow credential check
)

// CompareHashAndCredential checks, whether the hashed credential is a possible
// value when hashing the credential.
func CompareHashAndCredential(hashed string, zid id.Zid, ident string, credential string) (bool, error) {
	fullCredential := createFullCredential(zid, ident, credential)
	start := time.Now()
	err := bcrypt.CompareHashAndPassword([]byte(hashed), fullCredential)
	if elapsed := time.Since(start); elapsed+minSleep < durMinWait {
		time.Sleep(durMinWait - elapsed)
	} else {
		time.Sleep(minSleep)
	}
	if err == nil {
		return true, nil
	}
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	return false, err
}

func createFullCredential(zid id.Zid, ident string, credential string) []byte {
	var buf bytes.Buffer
	buf.WriteString(zid.String())
	buf.WriteByte(' ')
	buf.WriteString(ident)
	buf.WriteByte(' ')
	buf.WriteString(credential)
	return buf.Bytes()
}

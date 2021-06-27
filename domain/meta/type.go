//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta provides the domain specific type 'meta'.
package meta

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// DescriptionType is a description of a specific key type.
type DescriptionType struct {
	Name  string
	IsSet bool
}

// String returns the string representation of the given type
func (t DescriptionType) String() string { return t.Name }

var registeredTypes = make(map[string]*DescriptionType)

func registerType(name string, isSet bool) *DescriptionType {
	if _, ok := registeredTypes[name]; ok {
		panic("Type '" + name + "' already registered")
	}
	t := &DescriptionType{name, isSet}
	registeredTypes[name] = t
	return t
}

// Supported key types.
var (
	TypeBool         = registerType("Boolean", false)
	TypeCredential   = registerType("Credential", false)
	TypeEmpty        = registerType("EString", false)
	TypeID           = registerType("Identifier", false)
	TypeIDSet        = registerType("IdentifierSet", true)
	TypeNumber       = registerType("Number", false)
	TypeString       = registerType("String", false)
	TypeTagSet       = registerType("TagSet", true)
	TypeTimestamp    = registerType("Timestamp", false)
	TypeURL          = registerType("URL", false)
	TypeWord         = registerType("Word", false)
	TypeWordSet      = registerType("WordSet", true)
	TypeZettelmarkup = registerType("Zettelmarkup", false)
)

// Type returns a type hint for the given key. If no type hint is specified,
// TypeUnknown is returned.
func (m *Meta) Type(key string) *DescriptionType {
	return Type(key)
}

var (
	cachedTypedKeys = make(map[string]*DescriptionType)
	mxTypedKey      sync.RWMutex
)

func typedKey(key string, t *DescriptionType) *DescriptionType {
	mxTypedKey.Lock()
	defer mxTypedKey.Unlock()
	cachedTypedKeys[key] = t
	return t
}

// Type returns a type hint for the given key. If no type hint is specified,
// TypeUnknown is returned.
func Type(key string) *DescriptionType {
	if k, ok := registeredKeys[key]; ok {
		return k.Type
	}
	mxTypedKey.RLock()
	k, ok := cachedTypedKeys[key]
	mxTypedKey.RUnlock()
	if ok {
		return k
	}
	if strings.HasSuffix(key, "-url") {
		return typedKey(key, TypeURL)
	}
	if strings.HasSuffix(key, "-number") {
		return typedKey(key, TypeNumber)
	}
	if strings.HasSuffix(key, "-zid") {
		return typedKey(key, TypeID)
	}
	return TypeEmpty
}

// SetList stores the given string list value under the given key.
func (m *Meta) SetList(key string, values []string) {
	if key != KeyID {
		for i, val := range values {
			values[i] = trimValue(val)
		}
		m.pairs[key] = strings.Join(values, " ")
	}
}

// CleanTag removes the number charachter ('#') from a tag value.
func CleanTag(tag string) string {
	if len(tag) > 1 && tag[0] == '#' {
		return tag[1:]
	}
	return tag
}

// SetNow stores the current timestamp under the given key.
func (m *Meta) SetNow(key string) {
	m.Set(key, time.Now().Format("20060102150405"))
}

// BoolValue returns the value interpreted as a bool.
func BoolValue(value string) bool {
	if len(value) > 0 {
		switch value[0] {
		case '0', 'f', 'F', 'n', 'N':
			return false
		}
	}
	return true
}

// GetBool returns the boolean value of the given key.
func (m *Meta) GetBool(key string) bool {
	if value, ok := m.Get(key); ok {
		return BoolValue(value)
	}
	return false
}

// TimeValue returns the time value of the given value.
func TimeValue(value string) (time.Time, bool) {
	if t, err := time.Parse("20060102150405", value); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// GetTime returns the time value of the given key.
func (m *Meta) GetTime(key string) (time.Time, bool) {
	if value, ok := m.Get(key); ok {
		return TimeValue(value)
	}
	return time.Time{}, false
}

// ListFromValue transforms a string value into a list value.
func ListFromValue(value string) []string {
	return strings.Fields(value)
}

// GetList retrieves the string list value of a given key. The bool value
// signals, whether there was a value stored or not.
func (m *Meta) GetList(key string) ([]string, bool) {
	value, ok := m.Get(key)
	if !ok {
		return nil, false
	}
	return ListFromValue(value), true
}

// GetTags returns the list of tags as a string list. Each tag does not begin
// with the '#' character, in contrast to `GetList`.
func (m *Meta) GetTags(key string) ([]string, bool) {
	tags, ok := m.GetList(key)
	if !ok {
		return nil, false
	}
	for i, tag := range tags {
		tags[i] = CleanTag(tag)
	}
	return tags, len(tags) > 0
}

// GetListOrNil retrieves the string list value of a given key. If there was
// nothing stores, a nil list is returned.
func (m *Meta) GetListOrNil(key string) []string {
	if value, ok := m.GetList(key); ok {
		return value
	}
	return nil
}

// GetNumber retrieves the numeric value of a given key.
func (m *Meta) GetNumber(key string) (int, bool) {
	if value, ok := m.Get(key); ok {
		if num, err := strconv.Atoi(value); err == nil {
			return num, true
		}
	}
	return 0, false
}

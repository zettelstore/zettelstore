//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package meta

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
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
	TypeCredential   = registerType(api.MetaCredential, false)
	TypeEmpty        = registerType(api.MetaEmpty, false)
	TypeID           = registerType(api.MetaID, false)
	TypeIDSet        = registerType(api.MetaIDSet, true)
	TypeNumber       = registerType(api.MetaNumber, false)
	TypeString       = registerType(api.MetaString, false)
	TypeTagSet       = registerType(api.MetaTagSet, true)
	TypeTimestamp    = registerType(api.MetaTimestamp, false)
	TypeURL          = registerType(api.MetaURL, false)
	TypeWord         = registerType(api.MetaWord, false)
	TypeWordSet      = registerType(api.MetaWordSet, true)
	TypeZettelmarkup = registerType(api.MetaZettelmarkup, false)
)

// Type returns a type hint for the given key. If no type hint is specified,
// TypeUnknown is returned.
func (*Meta) Type(key string) *DescriptionType {
	return Type(key)
}

var (
	cachedTypedKeys = make(map[string]*DescriptionType)
	mxTypedKey      sync.RWMutex
	suffixTypes     = map[string]*DescriptionType{
		"-number": TypeNumber,
		"-role":   TypeWord,
		"-set":    TypeWordSet,
		"-title":  TypeZettelmarkup,
		"-url":    TypeURL,
		"-zettel": TypeID,
		"-zid":    TypeID,
		"-zids":   TypeIDSet,
	}
)

// Type returns a type hint for the given key. If no type hint is specified,
// TypeEmpty is returned.
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
	for suffix, t := range suffixTypes {
		if strings.HasSuffix(key, suffix) {
			mxTypedKey.Lock()
			defer mxTypedKey.Unlock()
			cachedTypedKeys[key] = t
			return t
		}
	}
	return TypeEmpty
}

// SetList stores the given string list value under the given key.
func (m *Meta) SetList(key string, values []string) {
	if key != api.KeyID {
		for i, val := range values {
			values[i] = trimValue(val)
		}
		m.pairs[key] = strings.Join(values, " ")
	}
}

// SetNow stores the current timestamp under the given key.
func (m *Meta) SetNow(key string) {
	m.Set(key, time.Now().Local().Format(id.ZidLayout))
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
	if t, err := time.Parse(id.ZidLayout, value); err == nil {
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
	tagsValue, ok := m.Get(key)
	if !ok {
		return nil, false
	}
	tags := ListFromValue(strings.ToLower(tagsValue))
	for i, tag := range tags {
		tags[i] = CleanTag(tag)
	}
	return tags, len(tags) > 0
}

// CleanTag removes the number character ('#') from a tag value and lowercases it.
func CleanTag(tag string) string {
	if len(tag) > 1 && tag[0] == '#' {
		return tag[1:]
	}
	return tag
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
func (m *Meta) GetNumber(key string, def int64) int64 {
	if value, ok := m.Get(key); ok {
		if num, err := strconv.ParseInt(value, 10, 64); err == nil {
			return num
		}
	}
	return def
}

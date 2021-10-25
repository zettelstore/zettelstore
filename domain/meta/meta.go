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
	"bytes"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/input"
)

type keyUsage int

const (
	_             keyUsage = iota
	usageUser              // Key will be manipulated by the user
	usageComputed          // Key is computed by zettelstore
	usageProperty          // Key is computed and not stored by zettelstore
)

// DescriptionKey formally describes each supported metadata key.
type DescriptionKey struct {
	Name    string
	Type    *DescriptionType
	usage   keyUsage
	Inverse string
}

// IsComputed returns true, if metadata is computed and not set by the user.
func (kd *DescriptionKey) IsComputed() bool { return kd.usage >= usageComputed }

// IsProperty returns true, if metadata is a computed property.
func (kd *DescriptionKey) IsProperty() bool { return kd.usage >= usageProperty }

var registeredKeys = make(map[string]*DescriptionKey)

func registerKey(name string, t *DescriptionType, usage keyUsage, inverse string) {
	if _, ok := registeredKeys[name]; ok {
		panic("Key '" + name + "' already defined")
	}
	if inverse != "" {
		if t != TypeID && t != TypeIDSet {
			panic("Inversable key '" + name + "' is not identifier type, but " + t.String())
		}
		inv, ok := registeredKeys[inverse]
		if !ok {
			panic("Inverse Key '" + inverse + "' not found")
		}
		if !inv.IsComputed() {
			panic("Inverse Key '" + inverse + "' is not computed.")
		}
		if inv.Type != TypeIDSet {
			panic("Inverse Key '" + inverse + "' is not an identifier set, but " + inv.Type.String())
		}
	}
	registeredKeys[name] = &DescriptionKey{name, t, usage, inverse}
}

// IsComputed returns true, if key denotes a computed metadata key.
func IsComputed(name string) bool {
	if kd, ok := registeredKeys[name]; ok {
		return kd.IsComputed()
	}
	return false
}

// Inverse returns the name of the inverse key.
func Inverse(name string) string {
	if kd, ok := registeredKeys[name]; ok {
		return kd.Inverse
	}
	return ""
}

// GetDescription returns the key description object of the given key name.
func GetDescription(name string) DescriptionKey {
	if d, ok := registeredKeys[name]; ok {
		return *d
	}
	return DescriptionKey{Type: Type(name)}
}

// GetSortedKeyDescriptions delivers all metadata key descriptions as a slice, sorted by name.
func GetSortedKeyDescriptions() []*DescriptionKey {
	names := make([]string, 0, len(registeredKeys))
	for n := range registeredKeys {
		names = append(names, n)
	}
	sort.Strings(names)
	result := make([]*DescriptionKey, 0, len(names))
	for _, n := range names {
		result = append(result, registeredKeys[n])
	}
	return result
}

// Supported keys.
func init() {
	registerKey(api.KeyID, TypeID, usageComputed, "")
	registerKey(api.KeyTitle, TypeZettelmarkup, usageUser, "")
	registerKey(api.KeyRole, TypeWord, usageUser, "")
	registerKey(api.KeyTags, TypeTagSet, usageUser, "")
	registerKey(api.KeySyntax, TypeWord, usageUser, "")
	registerKey(api.KeyAllTags, TypeTagSet, usageProperty, "")
	registerKey(api.KeyBack, TypeIDSet, usageProperty, "")
	registerKey(api.KeyBackward, TypeIDSet, usageProperty, "")
	registerKey(api.KeyBoxNumber, TypeNumber, usageComputed, "")
	registerKey(api.KeyContentTags, TypeTagSet, usageProperty, "")
	registerKey(api.KeyCopyright, TypeString, usageUser, "")
	registerKey(api.KeyCredential, TypeCredential, usageUser, "")
	registerKey(api.KeyDead, TypeIDSet, usageProperty, "")
	registerKey(api.KeyDuplicates, TypeBool, usageProperty, "")
	registerKey(api.KeyFolge, TypeIDSet, usageProperty, "")
	registerKey(api.KeyForward, TypeIDSet, usageProperty, "")
	registerKey(api.KeyLang, TypeWord, usageUser, "")
	registerKey(api.KeyLicense, TypeEmpty, usageUser, "")
	registerKey(api.KeyModified, TypeTimestamp, usageComputed, "")
	registerKey(api.KeyNoIndex, TypeBool, usageUser, "")
	registerKey(api.KeyPrecursor, TypeIDSet, usageUser, api.KeyFolge)
	registerKey(api.KeyPublished, TypeTimestamp, usageProperty, "")
	registerKey(api.KeyReadOnly, TypeWord, usageUser, "")
	registerKey(api.KeyURL, TypeURL, usageUser, "")
	registerKey(api.KeyUserID, TypeWord, usageUser, "")
	registerKey(api.KeyUserRole, TypeWord, usageUser, "")
	registerKey(api.KeyVisibility, TypeWord, usageUser, "")

	// Keys for runtime configuration zettel
	// See: https://zettelstore.de/manual/h/00001004020000
	registerKey(api.KeyDefaultCopyright, TypeString, usageUser, "")
	registerKey(api.KeyDefaultLang, TypeWord, usageUser, "")
	registerKey(api.KeyDefaultLicense, TypeEmpty, usageUser, "")
	registerKey(api.KeyDefaultRole, TypeWord, usageUser, "")
	registerKey(api.KeyDefaultSyntax, TypeWord, usageUser, "")
	registerKey(api.KeyDefaultTitle, TypeZettelmarkup, usageUser, "")
	registerKey(api.KeyDefaultVisibility, TypeWord, usageUser, "")
	registerKey(api.KeyExpertMode, TypeBool, usageUser, "")
	registerKey(api.KeyFooterHTML, TypeString, usageUser, "")
	registerKey(api.KeyHomeZettel, TypeID, usageUser, "")
	registerKey(api.KeyMarkerExternal, TypeEmpty, usageUser, "")
	registerKey(api.KeyMaxTransclusions, TypeNumber, usageUser, "")
	registerKey(api.KeySiteName, TypeString, usageUser, "")
	registerKey(api.KeyYAMLHeader, TypeBool, usageUser, "")
	registerKey(api.KeyZettelFileSyntax, TypeWordSet, usageUser, "")
}

// NewPrefix is the prefix for metadata key in template zettel for creating new zettel.
const NewPrefix = "new-"

// Meta contains all meta-data of a zettel.
type Meta struct {
	Zid     id.Zid
	pairs   map[string]string
	YamlSep bool
}

// New creates a new chunk for storing metadata.
func New(zid id.Zid) *Meta {
	return &Meta{Zid: zid, pairs: make(map[string]string, 5)}
}

// NewWithData creates metadata object with given data.
func NewWithData(zid id.Zid, data map[string]string) *Meta {
	pairs := make(map[string]string, len(data))
	for k, v := range data {
		pairs[k] = v
	}
	return &Meta{Zid: zid, pairs: pairs}
}

// Clone returns a new copy of the metadata.
func (m *Meta) Clone() *Meta {
	return &Meta{
		Zid:     m.Zid,
		pairs:   m.Map(),
		YamlSep: m.YamlSep,
	}
}

// Map returns a copy of the meta data as a string map.
func (m *Meta) Map() map[string]string {
	pairs := make(map[string]string, len(m.pairs))
	for k, v := range m.pairs {
		pairs[k] = v
	}
	return pairs
}

var reKey = regexp.MustCompile("^[0-9a-z][-0-9a-z]{0,254}$")

// KeyIsValid returns true, the the key is a valid string.
func KeyIsValid(key string) bool {
	return reKey.MatchString(key)
}

// Pair is one key-value-pair of a Zettel meta.
type Pair struct {
	Key   string
	Value string
}

var firstKeys = []string{api.KeyTitle, api.KeyRole, api.KeyTags, api.KeySyntax}
var firstKeySet map[string]bool

func init() {
	firstKeySet = make(map[string]bool, len(firstKeys))
	for _, k := range firstKeys {
		firstKeySet[k] = true
	}
}

// Set stores the given string value under the given key.
func (m *Meta) Set(key, value string) {
	if key != api.KeyID {
		m.pairs[key] = trimValue(value)
	}
}

func trimValue(value string) string {
	return strings.TrimFunc(value, input.IsSpace)
}

// Get retrieves the string value of a given key. The bool value signals,
// whether there was a value stored or not.
func (m *Meta) Get(key string) (string, bool) {
	if key == api.KeyID {
		return m.Zid.String(), true
	}
	value, ok := m.pairs[key]
	return value, ok
}

// GetDefault retrieves the string value of the given key. If no value was
// stored, the given default value is returned.
func (m *Meta) GetDefault(key, def string) string {
	if value, ok := m.Get(key); ok {
		return value
	}
	return def
}

// Pairs returns all key/values pairs stored, in a specific order. First come
// the pairs with predefined keys: MetaTitleKey, MetaTagsKey, MetaSyntaxKey,
// MetaContextKey. Then all other pairs are append to the list, ordered by key.
func (m *Meta) Pairs(allowComputed bool) []Pair {
	return m.doPairs(true, allowComputed)
}

// PairsRest returns all key/values pairs stored, except the values with
// predefined keys. The pairs are ordered by key.
func (m *Meta) PairsRest(allowComputed bool) []Pair {
	return m.doPairs(false, allowComputed)
}

func (m *Meta) doPairs(first, allowComputed bool) []Pair {
	result := make([]Pair, 0, len(m.pairs))
	if first {
		for _, key := range firstKeys {
			if value, ok := m.pairs[key]; ok {
				result = append(result, Pair{key, value})
			}
		}
	}

	keys := make([]string, 0, len(m.pairs)-len(result))
	for k := range m.pairs {
		if !firstKeySet[k] && (allowComputed || !IsComputed(k)) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		result = append(result, Pair{k, m.pairs[k]})
	}
	return result
}

// Delete removes a key from the data.
func (m *Meta) Delete(key string) {
	if key != api.KeyID {
		delete(m.pairs, key)
	}
}

// Equal compares to metas for equality.
func (m *Meta) Equal(o *Meta, allowComputed bool) bool {
	if m == nil && o == nil {
		return true
	}
	if m == nil || o == nil || m.Zid != o.Zid {
		return false
	}
	tested := make(map[string]bool, len(m.pairs))
	for k, v := range m.pairs {
		tested[k] = true
		if !equalValue(k, v, o, allowComputed) {
			return false
		}
	}
	for k, v := range o.pairs {
		if !tested[k] && !equalValue(k, v, m, allowComputed) {
			return false
		}
	}
	return true
}

func equalValue(key, val string, other *Meta, allowComputed bool) bool {
	if allowComputed || !IsComputed(key) {
		if valO, ok := other.pairs[key]; !ok || val != valO {
			return false
		}
	}
	return true
}

// Sanitize all metadata keys and values, so that they can be written safely into a file.
func (m *Meta) Sanitize() {
	if m == nil {
		return
	}
	for k, v := range m.pairs {
		m.pairs[RemoveNonGraphic(k)] = RemoveNonGraphic(v)
	}
}

// RemoveNonGraphic changes the given string not to include non-graphical characters.
// It is needed to sanitize meta data.
func RemoveNonGraphic(s string) string {
	if s == "" {
		return ""
	}
	pos := 0
	var buf bytes.Buffer
	for pos < len(s) {
		nextPos := strings.IndexFunc(s[pos:], func(r rune) bool { return !unicode.IsGraphic(r) })
		if nextPos < 0 {
			break
		}
		buf.WriteString(s[pos:nextPos])
		buf.WriteByte(' ')
		_, size := utf8.DecodeRuneInString(s[nextPos:])
		pos = nextPos + size
	}
	if pos == 0 {
		return strings.TrimSpace(s)
	}
	buf.WriteString(s[pos:])
	return strings.TrimSpace(buf.String())
}

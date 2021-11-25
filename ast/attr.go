//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package ast

import "strings"

// Attributes store additional information about some node types.
type Attributes struct {
	Attrs map[string]string
}

// IsEmpty returns true if there are no attributes.
func (a *Attributes) IsEmpty() bool { return a == nil || len(a.Attrs) == 0 }

// HasDefault returns true, if the default attribute "-" has been set.
func (a *Attributes) HasDefault() bool {
	if a != nil {
		_, ok := a.Attrs["-"]
		return ok
	}
	return false
}

// RemoveDefault removes the default attribute
func (a *Attributes) RemoveDefault() {
	a.Remove("-")
}

// Get returns the attribute value of the given key and a succes value.
func (a *Attributes) Get(key string) (string, bool) {
	if a != nil {
		value, ok := a.Attrs[key]
		return value, ok
	}
	return "", false
}

// Clone returns a duplicate of the attribute.
func (a *Attributes) Clone() *Attributes {
	if a == nil {
		return nil
	}
	attrs := make(map[string]string, len(a.Attrs))
	for k, v := range a.Attrs {
		attrs[k] = v
	}
	return &Attributes{attrs}
}

// Set changes the attribute that a given key has now a given value.
func (a *Attributes) Set(key, value string) *Attributes {
	if a == nil {
		return &Attributes{map[string]string{key: value}}
	}
	if a.Attrs == nil {
		a.Attrs = make(map[string]string)
	}
	a.Attrs[key] = value
	return a
}

// Remove the key from the attributes.
func (a *Attributes) Remove(key string) {
	if a != nil {
		delete(a.Attrs, key)
	}
}

// AddClass adds a value to the class attribute.
func (a *Attributes) AddClass(class string) *Attributes {
	if a == nil {
		return &Attributes{map[string]string{"class": class}}
	}
	if a.Attrs == nil {
		a.Attrs = make(map[string]string)
	}
	classes := a.GetClasses()
	for _, cls := range classes {
		if cls == class {
			return a
		}
	}
	classes = append(classes, class)
	a.Attrs["class"] = strings.Join(classes, " ")
	return a
}

// GetClasses returns the class values as a string slice
func (a *Attributes) GetClasses() []string {
	if a == nil {
		return nil
	}
	classes, ok := a.Attrs["class"]
	if !ok {
		return nil
	}
	return strings.Fields(classes)
}

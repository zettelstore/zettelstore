//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package ast

import "strings"

// Attributes store additional information about some node types.
type Attributes map[string]string

// IsEmpty returns true if there are no attributes.
func (a Attributes) IsEmpty() bool { return len(a) == 0 }

// HasDefault returns true, if the default attribute "-" has been set.
func (a Attributes) HasDefault() bool {
	if a != nil {
		_, ok := a["-"]
		return ok
	}
	return false
}

// RemoveDefault removes the default attribute
func (a Attributes) RemoveDefault() {
	if a != nil {
		a.Remove("-")
	}
}

// Get returns the attribute value of the given key and a succes value.
func (a Attributes) Get(key string) (string, bool) {
	if a != nil {
		value, ok := a[key]
		return value, ok
	}
	return "", false
}

// Clone returns a duplicate of the attribute.
func (a Attributes) Clone() Attributes {
	if a == nil {
		return nil
	}
	attrs := make(map[string]string, len(a))
	for k, v := range a {
		attrs[k] = v
	}
	return attrs
}

// Set changes the attribute that a given key has now a given value.
func (a Attributes) Set(key, value string) Attributes {
	if a == nil {
		return map[string]string{key: value}
	}
	a[key] = value
	return a
}

// Remove the key from the attributes.
func (a Attributes) Remove(key string) Attributes {
	if a != nil {
		delete(a, key)
	}
	return a
}

// AddClass adds a value to the class attribute.
func (a Attributes) AddClass(class string) Attributes {
	if a == nil {
		return map[string]string{"class": class}
	}
	classes := a.GetClasses()
	for _, cls := range classes {
		if cls == class {
			return a
		}
	}
	classes = append(classes, class)
	a["class"] = strings.Join(classes, " ")
	return a
}

// GetClasses returns the class values as a string slice
func (a Attributes) GetClasses() []string {
	if a == nil {
		return nil
	}
	classes, ok := a["class"]
	if !ok {
		return nil
	}
	return strings.Fields(classes)
}

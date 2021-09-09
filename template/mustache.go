//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// This file was derived from previous work:
// - https://github.com/hoisie/mustache (License: MIT)
//   Copyright (c) 2009 Michael Hoisie
// - https://github.com/cbroglie/mustache (a fork from above code)
//   Starting with commit [f9b4cbf]
//   Does not have an explicit copyright and obviously continues with
//   above MIT license.
// The license text is included in the same directory where this file is
// located. See file LICENSE.
//-----------------------------------------------------------------------------

// Package template implements the Mustache templating language.
package template

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"

	"zettelstore.de/z/strfun"
)

// Node represents a node in the parse tree.
// It is either a Tag or a textNode.
type node interface {
	node()
}

// Tag represents the different mustache tag types.
//
// Not all methods apply to all kinds of tags. Restrictions, if any, are noted
// in the documentation for each method. Use the Type method to find out the
// type of tag before calling type-specific methods. Calling a method
// inappropriate to the type of tag causes a run time panic.
type Tag interface {
	node

	// Type returns the type of the tag.
	Type() TagType
	// Name returns the name of the tag.
	Name() string
	// Tags returns any child tags. It panics for tag types which cannot contain
	// child tags (i.e. variable tags).
	Tags() []Tag
}

// A TagType represents the specific type of mustache tag that a Tag
// represents. The zero TagType is not a valid type.
type TagType uint

// Defines representing the possible Tag types
const (
	Invalid TagType = iota
	Variable
	Section
	InvertedSection
	Partial
)

type varNode struct {
	name string
	raw  bool
}

func (*varNode) node()          {}
func (*varNode) Type() TagType  { return Variable }
func (e *varNode) Name() string { return e.name }
func (*varNode) Tags() []Tag    { panic("mustache: Tags on Variable type") }

type sectionNode struct {
	name      string
	inverted  bool
	startline int
	nodes     []node
}

func (*sectionNode) node() {}
func (e *sectionNode) Type() TagType {
	if e.inverted {
		return InvertedSection
	}
	return Section
}
func (e *sectionNode) Name() string { return e.name }
func (e *sectionNode) Tags() []Tag  { return extractTags(e.nodes) }

type partialNode struct {
	name   string
	indent string
	prov   PartialProvider
}

func (*partialNode) node()          {}
func (*partialNode) Type() TagType  { return Partial }
func (e *partialNode) Name() string { return e.name }
func (*partialNode) Tags() []Tag    { return nil }

type textNode struct {
	text []byte
}

func (e *textNode) node() {}

// Template represents a compiled mustache template
type Template struct {
	data    string
	otag    string
	ctag    string
	p       int
	curline int
	nodes   []node
	partial PartialProvider
	errmiss bool // Error when variable is not found?
}

type parseError struct {
	line    int
	message string
}

func (p parseError) Error() string {
	return fmt.Sprintf("line %d: %s", p.line, p.message)
}

// Tags returns the mustache tags for the given template
func (tmpl *Template) Tags() []Tag {
	return extractTags(tmpl.nodes)
}

func extractTags(nodes []node) []Tag {
	tags := make([]Tag, 0, len(nodes))
	for _, elem := range nodes {
		switch elem := elem.(type) {
		case *varNode:
			tags = append(tags, elem)
		case *sectionNode:
			tags = append(tags, elem)
		case *partialNode:
			tags = append(tags, elem)
		}
	}
	return tags
}

func (tmpl *Template) readString(s string) (string, error) {
	newlines := 0
	for i := tmpl.p; ; i++ {
		//are we at the end of the string?
		if i+len(s) > len(tmpl.data) {
			return tmpl.data[tmpl.p:], io.EOF
		}

		if tmpl.data[i] == '\n' {
			newlines++
		}

		if tmpl.data[i] != s[0] {
			continue
		}

		match := true
		for j := 1; j < len(s); j++ {
			if s[j] != tmpl.data[i+j] {
				match = false
				break
			}
		}
		if match {
			e := i + len(s)
			text := tmpl.data[tmpl.p:e]
			tmpl.p = e

			tmpl.curline += newlines
			return text, nil
		}
	}
}

type textReadingResult struct {
	text          string
	padding       string
	mayStandalone bool
}

func (tmpl *Template) readText() (*textReadingResult, error) {
	pPrev := tmpl.p
	text, err := tmpl.readString(tmpl.otag)
	if err == io.EOF {
		return &textReadingResult{
			text:          text,
			padding:       "",
			mayStandalone: false,
		}, err
	}
	i := tmpl.p - len(tmpl.otag)
	for ; i > pPrev; i-- {
		if tmpl.data[i-1] != ' ' && tmpl.data[i-1] != '\t' {
			break
		}
	}

	if i == 0 || tmpl.data[i-1] == '\n' {
		return &textReadingResult{
			text:          tmpl.data[pPrev:i],
			padding:       tmpl.data[i : tmpl.p-len(tmpl.otag)],
			mayStandalone: true,
		}, nil
	}

	return &textReadingResult{
		text:          tmpl.data[pPrev : tmpl.p-len(tmpl.otag)],
		padding:       "",
		mayStandalone: false,
	}, nil
}

type tagReadingResult struct {
	tag        string
	standalone bool
}

var skipWhitespaceTagTypes = map[byte]bool{
	'#': true, '^': true, '/': true, '<': true, '>': true, '=': true, '!': true,
}

func (tmpl *Template) readTag(mayStandalone bool) (*tagReadingResult, error) {
	var text string
	var err error
	if tmpl.p < len(tmpl.data) && tmpl.data[tmpl.p] == '{' {
		text, err = tmpl.readString("}" + tmpl.ctag)
	} else {
		text, err = tmpl.readString(tmpl.ctag)
	}

	if err == io.EOF {
		//put the remaining text in a block
		return nil, parseError{tmpl.curline, "unmatched open tag"}
	}

	text = text[:len(text)-len(tmpl.ctag)]

	//trim the close tag off the text
	tag := strings.TrimSpace(text)
	if tag == "" {
		return nil, parseError{tmpl.curline, "empty tag"}
	}

	eow := tmpl.p
	for i := tmpl.p; i < len(tmpl.data); i++ {
		if !(tmpl.data[i] == ' ' || tmpl.data[i] == '\t') {
			eow = i
			break
		}
	}

	standalone := tmpl.skipWhitespaceTag(tag, eow, mayStandalone)

	return &tagReadingResult{
		tag:        tag,
		standalone: standalone,
	}, nil
}

func (tmpl *Template) skipWhitespaceTag(tag string, eow int, mayStandalone bool) bool {
	if !mayStandalone {
		return true
	}
	// Skip all whitespaces apeared after these types of tags until end of line if
	// the line only contains a tag and whitespaces.
	if _, ok := skipWhitespaceTagTypes[tag[0]]; !ok {
		return false
	}
	if eow == len(tmpl.data) {
		tmpl.p = eow
		return true
	}
	if eow < len(tmpl.data) && tmpl.data[eow] == '\n' {
		tmpl.p = eow + 1
		tmpl.curline++
		return true
	}
	if eow+1 < len(tmpl.data) && tmpl.data[eow] == '\r' && tmpl.data[eow+1] == '\n' {
		tmpl.p = eow + 2
		tmpl.curline++
		return true
	}
	return false
}

func (tmpl *Template) parsePartial(name, indent string) *partialNode {
	return &partialNode{
		name:   name,
		indent: indent,
		prov:   tmpl.partial,
	}
}

func (tmpl *Template) parseSection(section *sectionNode) error {
	for {
		textResult, err := tmpl.readText()
		text := textResult.text
		padding := textResult.padding
		mayStandalone := textResult.mayStandalone

		if err == io.EOF {
			//put the remaining text in a block
			return parseError{section.startline, "Section " + section.name + " has no closing tag"}
		}

		// put text into an item
		section.nodes = append(section.nodes, &textNode{[]byte(text)})

		tagResult, err := tmpl.readTag(mayStandalone)
		if err != nil {
			return err
		}

		if !tagResult.standalone {
			section.nodes = append(section.nodes, &textNode{[]byte(padding)})
		}

		tag := tagResult.tag
		switch tag[0] {
		case '!':
			//ignore comment
		case '#', '^':
			name := strings.TrimSpace(tag[1:])
			sn := &sectionNode{name, tag[0] == '^', tmpl.curline, []node{}}
			err := tmpl.parseSection(sn)
			if err != nil {
				return err
			}
			section.nodes = append(section.nodes, sn)
		case '/':
			name := strings.TrimSpace(tag[1:])
			if name != section.name {
				return parseError{tmpl.curline, "interleaved closing tag: " + name}
			}
			return nil
		case '>':
			name := strings.TrimSpace(tag[1:])
			partial := tmpl.parsePartial(name, textResult.padding)
			section.nodes = append(section.nodes, partial)
		case '=':
			if tag[len(tag)-1] != '=' {
				return parseError{tmpl.curline, "Invalid meta tag"}
			}
			tag = strings.TrimSpace(tag[1 : len(tag)-1])
			newtags := strings.SplitN(tag, " ", 2)
			if len(newtags) == 2 {
				tmpl.otag = newtags[0]
				tmpl.ctag = newtags[1]
			}
		case '{':
			if tag[len(tag)-1] == '}' {
				//use a raw tag
				name := strings.TrimSpace(tag[1 : len(tag)-1])
				section.nodes = append(section.nodes, &varNode{name, true})
			}
		case '&':
			name := strings.TrimSpace(tag[1:])
			section.nodes = append(section.nodes, &varNode{name, true})
		default:
			section.nodes = append(section.nodes, &varNode{tag, false})
		}
	}
}

func (tmpl *Template) parse() error {
	for {
		textResult, err := tmpl.readText()
		text := textResult.text
		padding := textResult.padding
		mayStandalone := textResult.mayStandalone

		if err == io.EOF {
			//put the remaining text in a block
			tmpl.nodes = append(tmpl.nodes, &textNode{[]byte(text)})
			return nil
		}

		// put text into an item
		tmpl.nodes = append(tmpl.nodes, &textNode{[]byte(text)})

		tagResult, err := tmpl.readTag(mayStandalone)
		if err != nil {
			return err
		}

		if !tagResult.standalone {
			tmpl.nodes = append(tmpl.nodes, &textNode{[]byte(padding)})
		}

		tag := tagResult.tag
		switch tag[0] {
		case '!':
			//ignore comment
		case '#', '^':
			name := strings.TrimSpace(tag[1:])
			sn := &sectionNode{name, tag[0] == '^', tmpl.curline, []node{}}
			err := tmpl.parseSection(sn)
			if err != nil {
				return err
			}
			tmpl.nodes = append(tmpl.nodes, sn)
		case '/':
			return parseError{tmpl.curline, "unmatched close tag"}
		case '>':
			name := strings.TrimSpace(tag[1:])
			partial := tmpl.parsePartial(name, textResult.padding)
			tmpl.nodes = append(tmpl.nodes, partial)
		case '=':
			if tag[len(tag)-1] != '=' {
				return parseError{tmpl.curline, "Invalid meta tag"}
			}
			tag = strings.TrimSpace(tag[1 : len(tag)-1])
			newtags := strings.SplitN(tag, " ", 2)
			if len(newtags) == 2 {
				tmpl.otag = newtags[0]
				tmpl.ctag = newtags[1]
			}
		case '{':
			//use a raw tag
			if tag[len(tag)-1] == '}' {
				name := strings.TrimSpace(tag[1 : len(tag)-1])
				tmpl.nodes = append(tmpl.nodes, &varNode{name, true})
			}
		case '&':
			name := strings.TrimSpace(tag[1:])
			tmpl.nodes = append(tmpl.nodes, &varNode{name, true})
		default:
			tmpl.nodes = append(tmpl.nodes, &varNode{tag, false})
		}
	}
}

// Evaluate interfaces and pointers looking for a value that can look up the
// name, via a struct field, method, or map key, and return the result of the
// lookup.
func lookup(stack []reflect.Value, name string, errMissing bool) (reflect.Value, error) {
	// dot notation
	if pos := strings.IndexByte(name, '.'); pos > 0 && pos < len(name)-1 {
		v, err := lookup(stack, name[:pos], errMissing)
		if err != nil {
			return v, err
		}
		return lookup([]reflect.Value{v}, name[pos+1:], errMissing)
	}

	for i := len(stack) - 1; i >= 0; i-- {
		if val, ok := lookupValue(stack[i], name); ok {
			return val, nil
		}
	}
	if errMissing {
		return reflect.Value{}, fmt.Errorf("missing variable %q", name)
	}
	return reflect.Value{}, nil
}

func lookupValue(v reflect.Value, name string) (reflect.Value, bool) {
	for v.IsValid() {
		typ := v.Type()
		if n := v.Type().NumMethod(); n > 0 {
			for i := 0; i < n; i++ {
				m := typ.Method(i)
				mtyp := m.Type
				if m.Name == name && mtyp.NumIn() == 1 {
					return v.Method(i).Call(nil)[0], true
				}
			}
		}
		if name == "." {
			return v, true
		}
		switch av := v; av.Kind() {
		case reflect.Ptr:
			v = av.Elem()
		case reflect.Interface:
			v = av.Elem()
		case reflect.Struct:
			return sanitizeValue(av.FieldByName(name))
		case reflect.Map:
			return sanitizeValue(av.MapIndex(reflect.ValueOf(name)))
		default:
			return reflect.Value{}, false
		}
	}
	return reflect.Value{}, false
}

func sanitizeValue(v reflect.Value) (reflect.Value, bool) {
	if v.IsValid() {
		return v, true
	}
	return reflect.Value{}, false
}

func isEmpty(v reflect.Value) bool {
	if !v.IsValid() || v.Interface() == nil {
		return true
	}

	valueInd := indirect(v)
	if !valueInd.IsValid() {
		return true
	}
	switch val := valueInd; val.Kind() {
	case reflect.Array, reflect.Slice:
		return val.Len() == 0
	case reflect.String:
		return strings.TrimSpace(val.String()) == ""
	default:
		return valueInd.IsZero()
	}
}

func indirect(v reflect.Value) reflect.Value {
loop:
	for v.IsValid() {
		switch av := v; av.Kind() {
		case reflect.Ptr:
			v = av.Elem()
		case reflect.Interface:
			v = av.Elem()
		default:
			break loop
		}
	}
	return v
}

func (tmpl *Template) renderSection(w io.Writer, section *sectionNode, stack []reflect.Value) error {
	value, err := lookup(stack, section.name, false)
	if err != nil {
		return err
	}

	// if the value is empty, check if it's an inverted section
	if isEmpty(value) != section.inverted {
		return nil
	}

	if !section.inverted {
		switch val := indirect(value); val.Kind() {
		case reflect.Slice, reflect.Array:
			valLen := val.Len()
			enumeration := make([]reflect.Value, valLen)
			for i := 0; i < valLen; i++ {
				enumeration[i] = val.Index(i)
			}
			topStack := len(stack)
			stack = append(stack, enumeration[0])
			for _, elem := range enumeration {
				stack[topStack] = elem
				if err := tmpl.renderNodes(w, section.nodes, stack); err != nil {
					return err
				}
			}
			return nil
		case reflect.Map, reflect.Struct:
			return tmpl.renderNodes(w, section.nodes, append(stack, value))
		}
		return tmpl.renderNodes(w, section.nodes, stack)
	}

	return tmpl.renderNodes(w, section.nodes, stack)
}

func (tmpl *Template) renderNodes(w io.Writer, nodes []node, stack []reflect.Value) error {
	for _, n := range nodes {
		if err := tmpl.renderNode(w, n, stack); err != nil {
			return err
		}
	}
	return nil
}

func (tmpl *Template) renderNode(w io.Writer, node node, stack []reflect.Value) error {
	switch n := node.(type) {
	case *textNode:
		_, err := w.Write(n.text)
		return err
	case *varNode:
		val, err := lookup(stack, n.name, tmpl.errmiss)
		if err != nil {
			return err
		}
		if val.IsValid() {
			if n.raw {
				fmt.Fprint(w, val.Interface())
			} else {
				s := fmt.Sprint(val.Interface())
				strfun.HTMLEscape(w, s, false)
			}
		}
	case *sectionNode:
		if err := tmpl.renderSection(w, n, stack); err != nil {
			return err
		}
	case *partialNode:
		partial, err := getPartials(n.prov, n.name, n.indent)
		if err != nil {
			return err
		}
		if err := partial.renderTemplate(w, stack); err != nil {
			return err
		}
	}
	return nil
}

func (tmpl *Template) renderTemplate(w io.Writer, stack []reflect.Value) error {
	for _, n := range tmpl.nodes {
		if err := tmpl.renderNode(w, n, stack); err != nil {
			return err
		}
	}
	return nil
}

// Render uses the given data source - generally a map or struct - to render
// the compiled template to an io.Writer.
func (tmpl *Template) Render(w io.Writer, data interface{}) error {
	return tmpl.renderTemplate(w, []reflect.Value{reflect.ValueOf(data)})
}

// ParseString compiles a mustache template string, retrieving any
// required partials from the given provider. The resulting output can be used
// to efficiently render the template multiple times with different data
// sources.
func ParseString(data string, partials PartialProvider) (*Template, error) {
	if partials == nil {
		partials = &EmptyProvider
	}
	tmpl := Template{data, "{{", "}}", 0, 1, []node{}, partials, false}
	err := tmpl.parse()
	if err != nil {
		return nil, err
	}
	return &tmpl, err
}

// SetErrorOnMissing will produce an error is a variable is not found.
func (tmpl *Template) SetErrorOnMissing() { tmpl.errmiss = true }

// PartialProvider comprises the behaviors required of a struct to be able to
// provide partials to the mustache rendering engine.
type PartialProvider interface {
	// Get accepts the name of a partial and returns the parsed partial, if it
	// could be found; a valid but empty template, if it could not be found; or
	// nil and error if an error occurred (other than an inability to find the
	// partial).
	Get(name string) (string, error)
}

// ErrPartialNotFound is returned if a partial was not found.
type ErrPartialNotFound struct {
	Name string
}

func (err *ErrPartialNotFound) Error() string {
	return "Partial '" + err.Name + "' not found"
}

// StaticProvider implements the PartialProvider interface by providing
// partials drawn from a map, which maps partial name to template contents.
type StaticProvider struct {
	Partials map[string]string
}

// Get accepts the name of a partial and returns the parsed partial.
func (sp *StaticProvider) Get(name string) (string, error) {
	if sp.Partials != nil {
		if data, ok := sp.Partials[name]; ok {
			return data, nil
		}
	}

	return "", &ErrPartialNotFound{name}
}

// emptyProvider will always returns an empty string.
type emptyProvider struct{}

// Get accepts the name of a partial and returns the parsed partial.
func (*emptyProvider) Get(string) (string, error) { return "", nil }

// EmptyProvider is a partial provider that will always return an empty string.
var EmptyProvider emptyProvider

var nonEmptyLine = regexp.MustCompile(`(?m:^(.+)$)`)

func getPartials(partials PartialProvider, name, indent string) (*Template, error) {
	data, err := partials.Get(name)
	if err != nil {
		return nil, err
	}

	// indent non empty lines
	data = nonEmptyLine.ReplaceAllString(data, indent+"$1")
	return ParseString(data, partials)
}

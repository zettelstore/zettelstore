//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zettelmark provides a parser for zettelmarkup.
package zettelmark

import (
	"unicode"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:         meta.ValueSyntaxZmk,
		AltNames:     nil,
		ParseBlocks:  parseBlocks,
		ParseInlines: parseInlines,
	})
}

func parseBlocks(inp *input.Input, m *meta.Meta, syntax string) ast.BlockSlice {
	parser := &zmkP{inp: inp}
	bs := parser.parseBlockSlice()
	return postProcessBlocks(bs)
}

func parseInlines(inp *input.Input, syntax string) ast.InlineSlice {
	parser := &zmkP{inp: inp}
	is := parser.parseInlineSlice()
	return postProcessInlines(is)
}

type zmkP struct {
	inp          *input.Input             // Input stream
	lists        []*ast.NestedListNode    // Stack of lists
	table        *ast.TableNode           // Current table
	descrl       *ast.DescriptionListNode // Current description list
	nestingLevel int                      // Count nesting of block and inline elements
}

// runeModGrave is Unicode code point U+02CB (715) called "MODIFIER LETTER
// GRAVE ACCENT". On the iPad it is much more easier to type in this code point
// than U+0060 (96) "Grave accent" (aka backtick). Therefore, U+02CB will be
// considered equivalent to U+0060.
const runeModGrave = 'ˋ' // This is NOT '`'!

const maxNestingLevel = 50

// clearStacked removes all multi-line nodes from parser.
func (cp *zmkP) clearStacked() {
	cp.lists = nil
	cp.table = nil
	cp.descrl = nil
}

func (cp *zmkP) parseNormalAttribute(attrs map[string]string, sameLine bool) bool {
	inp := cp.inp
	posK := inp.Pos
	for isNameRune(inp.Ch) {
		inp.Next()
	}
	if posK == inp.Pos {
		return false
	}
	key := string(inp.Src[posK:inp.Pos])
	if inp.Ch != '=' {
		attrs[key] = ""
		return true
	}
	if sameLine {
		switch inp.Ch {
		case input.EOS, '\n', '\r':
			return false
		}
	}
	return cp.parseAttributeValue(key, attrs, sameLine)
}

func (cp *zmkP) parseAttributeValue(
	key string, attrs map[string]string, sameLine bool) bool {
	inp := cp.inp
	inp.Next()
	if inp.Ch == '"' {
		inp.Next()
		var val string
		for {
			switch inp.Ch {
			case input.EOS:
				return false
			case '"':
				updateAttrs(attrs, key, val)
				inp.Next()
				return true
			case '\n', '\r':
				if sameLine {
					return false
				}
				inp.EatEOL()
				val += " "
			case '\\':
				inp.Next()
				switch inp.Ch {
				case input.EOS, '\n', '\r':
					return false
				}
				fallthrough
			default:
				val += string(inp.Ch)
				inp.Next()
			}
		}
	}
	posV := inp.Pos
	for {
		switch inp.Ch {
		case input.EOS:
			return false
		case '\n', '\r':
			if sameLine {
				return false
			}
			fallthrough
		case ' ', '}':
			updateAttrs(attrs, key, inp.Src[posV:inp.Pos])
			return true
		}
		inp.Next()
	}
}

func updateAttrs(attrs map[string]string, key string, val string) {
	if prevVal := attrs[key]; len(prevVal) > 0 {
		attrs[key] = prevVal + " " + val
	} else {
		attrs[key] = val
	}
}

// parseAttributes reads optional attributes.
// If sameLine is True, it is called from block nodes. In this case, a single
// name is allowed. It will parse as {name}. Attributes are not allowed to be
// continued on next line.
// If sameLine is False, it is called from inline nodes. In this case, the next
// rune must be '{'. A continuation on next lines is allowed.
func (cp *zmkP) parseAttributes(sameLine bool) *ast.Attributes {
	inp := cp.inp
	if sameLine {
		pos := inp.Pos
		for isNameRune(inp.Ch) {
			inp.Next()
		}
		if pos < inp.Pos {
			return &ast.Attributes{Attrs: map[string]string{"": inp.Src[pos:inp.Pos]}}
		}

		// No immediate name: skip spaces
		cp.skipSpace(!sameLine)
	}

	pos := inp.Pos
	attrs, success := cp.doParseAttributes(sameLine)
	if sameLine || success {
		return attrs
	}
	inp.SetPos(pos)
	return nil
}

func (cp *zmkP) doParseAttributes(sameLine bool) (res *ast.Attributes, success bool) {
	inp := cp.inp
	if inp.Ch != '{' {
		return nil, false
	}
	inp.Next()
	attrs := map[string]string{}
loop:
	for {
		cp.skipSpace(!sameLine)
		switch inp.Ch {
		case input.EOS:
			return nil, false
		case '}':
			break loop
		case '.':
			inp.Next()
			posC := inp.Pos
			for isNameRune(inp.Ch) {
				inp.Next()
			}
			if posC == inp.Pos {
				return nil, false
			}
			updateAttrs(attrs, "class", inp.Src[posC:inp.Pos])
		case '=':
			delete(attrs, "")
			if !cp.parseAttributeValue("", attrs, sameLine) {
				return nil, false
			}
		default:
			if !cp.parseNormalAttribute(attrs, sameLine) {
				return nil, false
			}
		}
		switch inp.Ch {
		case '}':
			break loop
		case '\n', '\r':
			if sameLine {
				return nil, false
			}
		case ' ', ',':
			inp.Next()
		default:
			return nil, false
		}
	}
	inp.Next()
	return &ast.Attributes{Attrs: attrs}, true
}

func (cp *zmkP) skipSpace(eolIsSpace bool) {
	inp := cp.inp
	if eolIsSpace {
		for {
			switch inp.Ch {
			case ' ':
				inp.Next()
			case '\n', '\r':
				inp.EatEOL()
			default:
				return
			}
		}
	}
	for inp.Ch == ' ' {
		inp.Next()
	}
}

func isNameRune(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '-' || ch == '_'
}

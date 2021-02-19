//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlenc encodes the abstract syntax tree into HTML5.
package htmlenc

import (
	"fmt"
	"strconv"
	"strings"

	"zettelstore.de/z/ast"
)

// VisitPara emits HTML code for a paragraph: <p>...</p>
func (v *visitor) VisitPara(pn *ast.ParaNode) {
	v.b.WriteString("<p>")
	v.acceptInlineSlice(pn.Inlines)
	v.writeEndPara()
}

// VisitVerbatim emits HTML code for verbatim lines.
func (v *visitor) VisitVerbatim(vn *ast.VerbatimNode) {
	switch vn.Code {
	case ast.VerbatimProg:
		oldVisible := v.visibleSpace
		if vn.Attrs != nil {
			v.visibleSpace = vn.Attrs.HasDefault()
		}
		v.b.WriteString("<pre><code")
		v.visitAttributes(vn.Attrs)
		v.b.WriteByte('>')
		for _, line := range vn.Lines {
			v.writeHTMLEscaped(line)
			v.b.WriteByte('\n')
		}
		v.b.WriteString("</code></pre>\n")
		v.visibleSpace = oldVisible

	case ast.VerbatimComment:
		if vn.Attrs.HasDefault() {
			v.b.WriteString("<!--\n")
			for _, line := range vn.Lines {
				v.writeHTMLEscaped(line)
				v.b.WriteByte('\n')
			}
			v.b.WriteString("-->\n")
		}

	case ast.VerbatimHTML:
		for _, line := range vn.Lines {
			v.b.WriteStrings(line, "\n")
		}
	default:
		panic(fmt.Sprintf("Unknown verbatim code %v", vn.Code))
	}
}

var specialSpanAttr = map[string]bool{
	"example":   true,
	"note":      true,
	"tip":       true,
	"important": true,
	"caution":   true,
	"warning":   true,
}

func processSpanAttributes(attrs *ast.Attributes) *ast.Attributes {
	if attrVal, ok := attrs.Get(""); ok {
		attrVal = strings.ToLower(attrVal)
		if specialSpanAttr[attrVal] {
			attrs = attrs.Clone()
			attrs.Remove("")
			attrs = attrs.AddClass("zs-indication").AddClass("zs-" + attrVal)
		}
	}
	return attrs
}

// VisitRegion writes HTML code for block regions.
func (v *visitor) VisitRegion(rn *ast.RegionNode) {
	var code string
	attrs := rn.Attrs
	oldVerse := v.inVerse
	switch rn.Code {
	case ast.RegionSpan:
		code = "div"
		attrs = processSpanAttributes(attrs)
	case ast.RegionVerse:
		v.inVerse = true
		code = "div"
	case ast.RegionQuote:
		code = "blockquote"
	default:
		panic(fmt.Sprintf("Unknown region code %v", rn.Code))
	}

	v.lang.push(attrs)
	defer v.lang.pop()

	v.b.WriteStrings("<", code)
	v.visitAttributes(attrs)
	v.b.WriteString(">\n")
	v.acceptBlockSlice(rn.Blocks)
	if len(rn.Inlines) > 0 {
		v.b.WriteString("<cite>")
		v.acceptInlineSlice(rn.Inlines)
		v.b.WriteString("</cite>\n")
	}
	v.b.WriteStrings("</", code, ">\n")
	v.inVerse = oldVerse
}

// VisitHeading writes the HTML code for a heading.
func (v *visitor) VisitHeading(hn *ast.HeadingNode) {
	v.lang.push(hn.Attrs)
	defer v.lang.pop()

	lvl := hn.Level
	if lvl > 6 {
		lvl = 6 // HTML has H1..H6
	}
	strLvl := strconv.Itoa(lvl)
	v.b.WriteStrings("<h", strLvl)
	v.visitAttributes(hn.Attrs)
	if slug := hn.Slug; len(slug) > 0 {
		v.b.WriteStrings(" id=\"", slug, "\"")
	}
	v.b.WriteByte('>')
	v.acceptInlineSlice(hn.Inlines)
	v.b.WriteStrings("</h", strLvl, ">\n")
}

// VisitHRule writes HTML code for a horizontal rule: <hr>.
func (v *visitor) VisitHRule(hn *ast.HRuleNode) {
	v.b.WriteString("<hr")
	v.visitAttributes(hn.Attrs)
	if v.xhtml {
		v.b.WriteString(" />\n")
	} else {
		v.b.WriteString(">\n")
	}
}

var listCode = map[ast.NestedListCode]string{
	ast.NestedListOrdered:   "ol",
	ast.NestedListUnordered: "ul",
}

// VisitNestedList writes HTML code for lists and blockquotes.
func (v *visitor) VisitNestedList(ln *ast.NestedListNode) {
	v.lang.push(ln.Attrs)
	defer v.lang.pop()

	if ln.Code == ast.NestedListQuote {
		// NestedListQuote -> HTML <blockquote> doesn't use <li>...</li>
		v.writeQuotationList(ln)
		return
	}

	code, ok := listCode[ln.Code]
	if !ok {
		panic(fmt.Sprintf("Invalid list code %v", ln.Code))
	}

	compact := isCompactList(ln.Items)
	v.b.WriteStrings("<", code)
	v.visitAttributes(ln.Attrs)
	v.b.WriteString(">\n")
	for _, item := range ln.Items {
		v.b.WriteString("<li>")
		v.writeItemSliceOrPara(item, compact)
		v.b.WriteString("</li>\n")
	}
	v.b.WriteStrings("</", code, ">\n")
}

func (v *visitor) writeQuotationList(ln *ast.NestedListNode) {
	v.b.WriteString("<blockquote>\n")
	inPara := false
	for _, item := range ln.Items {
		if pn := getParaItem(item); pn != nil {
			if inPara {
				v.b.WriteByte('\n')
			} else {
				v.b.WriteString("<p>")
				inPara = true
			}
			v.acceptInlineSlice(pn.Inlines)
		} else {
			if inPara {
				v.writeEndPara()
				inPara = false
			}
			v.acceptItemSlice(item)
		}
	}
	if inPara {
		v.writeEndPara()
	}
	v.b.WriteString("</blockquote>\n")
}

func getParaItem(its ast.ItemSlice) *ast.ParaNode {
	if len(its) != 1 {
		return nil
	}
	if pn, ok := its[0].(*ast.ParaNode); ok {
		return pn
	}
	return nil
}

func isCompactList(insl []ast.ItemSlice) bool {
	for _, ins := range insl {
		if !isCompactSlice(ins) {
			return false
		}
	}
	return true
}

func isCompactSlice(ins ast.ItemSlice) bool {
	if len(ins) < 1 {
		return true
	}
	if len(ins) == 1 {
		switch ins[0].(type) {
		case *ast.ParaNode, *ast.VerbatimNode, *ast.HRuleNode:
			return true
		case *ast.NestedListNode:
			return false
		}
	}
	return false
}

// writeItemSliceOrPara emits the content of a paragraph if the paragraph is
// the only element of the block slice and if compact mode is true. Otherwise,
// the item slice is emitted normally.
func (v *visitor) writeItemSliceOrPara(ins ast.ItemSlice, compact bool) {
	if compact && len(ins) == 1 {
		if para, ok := ins[0].(*ast.ParaNode); ok {
			v.acceptInlineSlice(para.Inlines)
			return
		}
	}
	v.acceptItemSlice(ins)
}

func (v *visitor) writeDescriptionsSlice(ds ast.DescriptionSlice) {
	if len(ds) == 1 {
		if para, ok := ds[0].(*ast.ParaNode); ok {
			v.acceptInlineSlice(para.Inlines)
			return
		}
	}
	for _, dn := range ds {
		dn.Accept(v)
	}
}

// VisitDescriptionList emits a HTML description list.
func (v *visitor) VisitDescriptionList(dn *ast.DescriptionListNode) {
	v.b.WriteString("<dl>\n")
	for _, descr := range dn.Descriptions {
		v.b.WriteString("<dt>")
		v.acceptInlineSlice(descr.Term)
		v.b.WriteString("</dt>\n")

		for _, b := range descr.Descriptions {
			v.b.WriteString("<dd>")
			v.writeDescriptionsSlice(b)
			v.b.WriteString("</dd>\n")
		}
	}
	v.b.WriteString("</dl>\n")
}

// VisitTable emits a HTML table.
func (v *visitor) VisitTable(tn *ast.TableNode) {
	v.b.WriteString("<table>\n")
	if len(tn.Header) > 0 {
		v.b.WriteString("<thead>\n")
		v.writeRow(tn.Header, "<th", "</th>")
		v.b.WriteString("</thead>\n")
	}
	if len(tn.Rows) > 0 {
		v.b.WriteString("<tbody>\n")
		for _, row := range tn.Rows {
			v.writeRow(row, "<td", "</td>")
		}
		v.b.WriteString("</tbody>\n")
	}
	v.b.WriteString("</table>\n")
}

var alignStyle = map[ast.Alignment]string{
	ast.AlignDefault: ">",
	ast.AlignLeft:    " style=\"text-align:left\">",
	ast.AlignCenter:  " style=\"text-align:center\">",
	ast.AlignRight:   " style=\"text-align:right\">",
}

func (v *visitor) writeRow(row ast.TableRow, cellStart, cellEnd string) {
	v.b.WriteString("<tr>")
	for _, cell := range row {
		v.b.WriteString(cellStart)
		if len(cell.Inlines) == 0 {
			v.b.WriteByte('>')
		} else {
			v.b.WriteString(alignStyle[cell.Align])
			v.acceptInlineSlice(cell.Inlines)
		}
		v.b.WriteString(cellEnd)
	}
	v.b.WriteString("</tr>\n")
}

// VisitBLOB writes the binary object as a value.
func (v *visitor) VisitBLOB(bn *ast.BLOBNode) {
	switch bn.Syntax {
	case "gif", "jpeg", "png":
		v.b.WriteStrings("<img src=\"data:image/", bn.Syntax, ";base64,")
		v.b.WriteBase64(bn.Blob)
		v.b.WriteString("\" title=\"")
		v.writeQuotedEscaped(bn.Title)
		v.b.WriteString("\">\n")
	default:
		v.b.WriteStrings("<p class=\"error\">Unable to display BLOB with syntax '", bn.Syntax, "'.</p>\n")
	}
}

func (v *visitor) writeEndPara() {
	v.b.WriteString("</p>\n")
}

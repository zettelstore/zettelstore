//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlenc encodes the abstract syntax tree into HTML5.
package htmlenc

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/strfun"
)

func (v *visitor) visitVerbatim(vn *ast.VerbatimNode) {
	switch vn.Kind {
	case ast.VerbatimZettel:
		v.b.WriteString("<!--\n")
		v.visitAttributes(vn.Attrs)
		v.writeHTMLEscaped(string(vn.Content))
		v.b.WriteString("\n-->")
	case ast.VerbatimProg:
		oldVisible := v.visibleSpace
		if vn.Attrs != nil {
			v.visibleSpace = vn.Attrs.HasDefault()
		}
		v.b.WriteString("<pre><code")
		v.visitAttributes(vn.Attrs)
		v.b.WriteByte('>')
		v.writeHTMLEscaped(string(vn.Content))
		v.b.WriteString("</code></pre>")
		v.visibleSpace = oldVisible

	case ast.VerbatimComment:
		if vn.Attrs.HasDefault() {
			v.b.WriteString("<!--\n")
			v.writeHTMLEscaped(string(vn.Content))
			v.b.WriteString("\n-->")
		}

	case ast.VerbatimHTML:
		lines := bytes.Split(vn.Content, []byte{'\n'})
		for _, line := range lines {
			sLine := string(line)
			if !ignoreHTMLText(sLine) {
				v.b.WriteStrings(sLine, "\n")
			}
		}
	default:
		panic(fmt.Sprintf("Unknown verbatim kind %v", vn.Kind))
	}
}

var htmlSnippetsIgnore = []string{
	"<script",
	"</script",
	"<iframe",
	"</iframe",
}

func ignoreHTMLText(s string) bool {
	lower := strings.ToLower(s)
	for _, snippet := range htmlSnippetsIgnore {
		if strings.Contains(lower, snippet) {
			return true
		}
	}
	return false
}

var specialSpanAttr = strfun.NewSet("example", "note", "tip", "important", "caution", "warning")

func processSpanAttributes(attrs ast.Attributes) ast.Attributes {
	if attrVal, ok := attrs.Get(""); ok {
		attrVal = strings.ToLower(attrVal)
		if specialSpanAttr.Has(attrVal) {
			attrs = attrs.Clone()
			attrs.Remove("")
			attrs = attrs.AddClass("zs-indication").AddClass("zs-" + attrVal)
		}
	}
	return attrs
}

func (v *visitor) visitRegion(rn *ast.RegionNode) {
	var code string
	attrs := rn.Attrs
	oldVerse := v.inVerse
	switch rn.Kind {
	case ast.RegionSpan:
		code = "div"
		attrs = processSpanAttributes(attrs)
	case ast.RegionVerse:
		v.inVerse = true
		code = "div"
	case ast.RegionQuote:
		code = "blockquote"
	default:
		panic(fmt.Sprintf("Unknown region kind %v", rn.Kind))
	}

	v.lang.push(attrs)
	defer v.lang.pop()

	v.b.WriteStrings("<", code)
	v.visitAttributes(attrs)
	v.b.WriteString(">\n")
	ast.Walk(v, &rn.Blocks)
	if len(rn.Inlines) > 0 {
		v.b.WriteString("\n<cite>")
		ast.Walk(v, &rn.Inlines)
		v.b.WriteString("</cite>")
	}
	v.b.WriteStrings("\n</", code, ">")
	v.inVerse = oldVerse
}

func (v *visitor) visitHeading(hn *ast.HeadingNode) {
	v.lang.push(hn.Attrs)
	defer v.lang.pop()

	lvl := hn.Level + 1
	if lvl > 6 {
		lvl = 6 // HTML has H1..H6
	}
	strLvl := strconv.Itoa(lvl)
	v.b.WriteStrings("<h", strLvl)
	v.visitAttributes(hn.Attrs)
	if _, ok := hn.Attrs.Get("id"); !ok {
		if fragment := hn.Fragment; fragment != "" {
			v.b.WriteStrings(" id=\"", fragment, "\"")
		}
	}
	v.b.WriteByte('>')
	ast.Walk(v, &hn.Inlines)
	v.b.WriteStrings("</h", strLvl, ">")
}

var mapNestedListKind = map[ast.NestedListKind]string{
	ast.NestedListOrdered:   "ol",
	ast.NestedListUnordered: "ul",
}

func (v *visitor) visitNestedList(ln *ast.NestedListNode) {
	v.lang.push(ln.Attrs)
	defer v.lang.pop()

	if ln.Kind == ast.NestedListQuote {
		// NestedListQuote -> HTML <blockquote> doesn't use <li>...</li>
		v.writeQuotationList(ln)
		return
	}

	code, ok := mapNestedListKind[ln.Kind]
	if !ok {
		panic(fmt.Sprintf("Invalid list kind %v", ln.Kind))
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
	v.b.WriteStrings("</", code, ">")
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
			ast.Walk(v, &pn.Inlines)
		} else {
			if inPara {
				v.writeEndPara()
				inPara = false
			}
			ast.WalkItemSlice(v, item)
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
			ast.Walk(v, &para.Inlines)
			return
		}
	}
	for i, in := range ins {
		if i >= 0 {
			v.b.WriteByte('\n')
		}
		ast.Walk(v, in)
	}
	v.b.WriteByte('\n')
}

func (v *visitor) writeDescriptionsSlice(ds ast.DescriptionSlice) {
	if len(ds) == 1 {
		if para, ok := ds[0].(*ast.ParaNode); ok {
			ast.Walk(v, &para.Inlines)
			return
		}
	}
	ast.WalkDescriptionSlice(v, ds)
}

func (v *visitor) visitDescriptionList(dn *ast.DescriptionListNode) {
	v.b.WriteString("<dl>\n")
	for _, descr := range dn.Descriptions {
		v.b.WriteString("<dt>")
		ast.Walk(v, &descr.Term)
		v.b.WriteString("</dt>\n")

		for _, b := range descr.Descriptions {
			v.b.WriteString("<dd>")
			v.writeDescriptionsSlice(b)
			v.b.WriteString("</dd>\n")
		}
	}
	v.b.WriteString("</dl>")
}

func (v *visitor) visitTable(tn *ast.TableNode) {
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
	v.b.WriteString("</table>")
}

var alignStyle = map[ast.Alignment]string{
	ast.AlignDefault: ">",
	ast.AlignLeft:    " class=\"zs-ta-left\">",
	ast.AlignCenter:  " class=\"zs-ta-center\">",
	ast.AlignRight:   " class=\"zs-ta-right\">",
}

func (v *visitor) writeRow(row ast.TableRow, cellStart, cellEnd string) {
	v.b.WriteString("<tr>")
	for _, cell := range row {
		v.b.WriteString(cellStart)
		if len(cell.Inlines) == 0 {
			v.b.WriteByte('>')
		} else {
			v.b.WriteString(alignStyle[cell.Align])
			ast.Walk(v, &cell.Inlines)
		}
		v.b.WriteString(cellEnd)
	}
	v.b.WriteString("</tr>\n")
}

func (v *visitor) visitBLOB(bn *ast.BLOBNode) {
	switch bn.Syntax {
	case api.ValueSyntaxSVG:
		v.b.Write(bn.Blob)
	case api.ValueSyntaxGif, "jpeg", "png":
		v.b.WriteStrings("<img src=\"data:image/", bn.Syntax, ";base64,")
		v.b.WriteBase64(bn.Blob)
		v.b.WriteString("\" title=\"")
		v.writeQuotedEscaped(bn.Title)
		v.b.WriteString("\">")
	default:
		v.b.WriteStrings("<p class=\"zs-error\">Unable to display BLOB with syntax '", bn.Syntax, "'.</p>")
	}
}

func (v *visitor) writeEndPara() {
	v.b.WriteString("</p>")
}

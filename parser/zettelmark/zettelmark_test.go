//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zettelmark_test provides some tests for the zettelmarkup parser.
package zettelmark_test

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"

	// Ensure that the text encoder is available.
	// Needed by parser/cleanup.go
	_ "zettelstore.de/z/encoder/textenc"
)

type TestCase struct{ source, want string }
type TestCases []TestCase

func replace(s string, tcs TestCases) TestCases {
	var testCases TestCases

	for _, tc := range tcs {
		source := strings.ReplaceAll(tc.source, "$", s)
		want := strings.ReplaceAll(tc.want, "$", s)
		testCases = append(testCases, TestCase{source, want})
	}
	return testCases
}

func checkTcs(t *testing.T, tcs TestCases) {
	t.Helper()

	for tcn, tc := range tcs {
		t.Run(fmt.Sprintf("TC=%02d,src=%q", tcn, tc.source), func(st *testing.T) {
			st.Helper()
			inp := input.NewInput([]byte(tc.source))
			bns := parser.ParseBlocks(inp, nil, api.ValueSyntaxZmk)
			var tv TestVisitor
			ast.Walk(&tv, bns)
			got := tv.String()
			if tc.want != got {
				st.Errorf("\nwant=%q\n got=%q", tc.want, got)
			}
		})
	}
}

func TestEOL(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"", ""},
		{"\n", ""},
		{"\r", ""},
		{"\r\n", ""},
		{"\n\n", ""},
	})
}

func TestText(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"abcd", "(PARA abcd)"},
		{"ab cd", "(PARA ab SP cd)"},
		{"abcd ", "(PARA abcd)"},
		{" abcd", "(PARA abcd)"},
		{"\\", "(PARA \\)"},
		{"\\\n", ""},
		{"\\\ndef", "(PARA HB def)"},
		{"\\\r", ""},
		{"\\\rdef", "(PARA HB def)"},
		{"\\\r\n", ""},
		{"\\\r\ndef", "(PARA HB def)"},
		{"\\a", "(PARA a)"},
		{"\\aa", "(PARA aa)"},
		{"a\\a", "(PARA aa)"},
		{"\\+", "(PARA +)"},
		{"\\ ", "(PARA \u00a0)"},
		{"...", "(PARA \u2026)"},
		{"...,", "(PARA \u2026,)"},
		{"...;", "(PARA \u2026;)"},
		{"...:", "(PARA \u2026:)"},
		{"...!", "(PARA \u2026!)"},
		{"...?", "(PARA \u2026?)"},
		{"...-", "(PARA ...-)"},
		{"a...b", "(PARA a...b)"},
		// {"http://a, http://b", "(PARA http://a, SP http://b)"},
	})
}

func TestSpace(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{" ", ""},
		{"\t", ""},
		{"  ", ""},
	})
}

func TestSoftBreak(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"x\ny", "(PARA x SB y)"},
		{"z\n", "(PARA z)"},
		{" \n ", ""},
		{" \n", ""},
	})
}

func TestHardBreak(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"x  \ny", "(PARA x HB y)"},
		{"z  \n", "(PARA z)"},
		{"   \n ", ""},
		{"   \n", ""},
	})
}

func TestLink(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"[", "(PARA [)"},
		{"[[", "(PARA [[)"},
		{"[[|", "(PARA [[|)"},
		{"[[]", "(PARA [[])"},
		{"[[|]", "(PARA [[|])"},
		{"[[]]", "(PARA [[]])"},
		{"[[|]]", "(PARA [[|]])"},
		{"[[ ]]", "(PARA [[ SP ]])"},
		{"[[\n]]", "(PARA [[ SB ]])"},
		{"[[ a]]", "(PARA (LINK a a))"},
		{"[[a ]]", "(PARA [[a SP ]])"},
		{"[[a\n]]", "(PARA [[a SB ]])"},
		{"[[a]]", "(PARA (LINK a a))"},
		{"[[12345678901234]]", "(PARA (LINK 12345678901234 12345678901234))"},
		{"[[a]", "(PARA [[a])"},
		{"[[|a]]", "(PARA [[|a]])"},
		{"[[b|]]", "(PARA [[b|]])"},
		{"[[b|a]]", "(PARA (LINK a b))"},
		{"[[b| a]]", "(PARA (LINK a b))"},
		{"[[b%c|a]]", "(PARA (LINK a b%c))"},
		{"[[b%%c|a]]", "(PARA [[b {% c|a]]})"},
		{"[[b|a]", "(PARA [[b|a])"},
		{"[[b\nc|a]]", "(PARA (LINK a b SB c))"},
		{"[[b c|a#n]]", "(PARA (LINK a#n b SP c))"},
		{"[[a]]go", "(PARA (LINK a a) go)"},
		{"[[a]]{go}", "(PARA (LINK a a)[ATTR go])"},
		{"[[[[a]]|b]]", "(PARA (LINK [[a [[a) |b]])"},
		{"[[a[b]c|d]]", "(PARA (LINK d a[b]c))"},
		{"[[[b]c|d]]", "(PARA (LINK d [b]c))"},
		{"[[a[]c|d]]", "(PARA (LINK d a[]c))"},
		{"[[a[b]|d]]", "(PARA (LINK d a[b]))"},
		{"[[\\|]]", "(PARA (LINK %5C%7C \\|))"},
		{"[[\\||a]]", "(PARA (LINK a |))"},
		{"[[b\\||a]]", "(PARA (LINK a b|))"},
		{"[[b\\|c|a]]", "(PARA (LINK a b|c))"},
		{"[[\\]]]", "(PARA (LINK %5C%5D \\]))"},
		{"[[\\]|a]]", "(PARA (LINK a ]))"},
		{"[[b\\]|a]]", "(PARA (LINK a b]))"},
		{"[[\\]\\||a]]", "(PARA (LINK a ]|))"},
		{"[[http://a|http://a]]", "(PARA (LINK http://a http://a))"},
	})
}

func TestCite(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"[@", "(PARA [@)"},
		{"[@]", "(PARA [@])"},
		{"[@a]", "(PARA (CITE a))"},
		{"[@ a]", "(PARA [@ SP a])"},
		{"[@a ]", "(PARA (CITE a))"},
		{"[@a\n]", "(PARA (CITE a))"},
		{"[@a\nx]", "(PARA (CITE a SB x))"},
		{"[@a\n\n]", "(PARA [@a)(PARA ])"},
		{"[@a,\n]", "(PARA (CITE a))"},
		{"[@a,n]", "(PARA (CITE a n))"},
		{"[@a| n]", "(PARA (CITE a n))"},
		{"[@a|n ]", "(PARA (CITE a n))"},
		{"[@a,[@b]]", "(PARA (CITE a (CITE b)))"},
		{"[@a]{color=green}", "(PARA (CITE a)[ATTR color=green])"},
	})
}

func TestFootnote(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"[^", "(PARA [^)"},
		{"[^]", "(PARA (FN))"},
		{"[^abc]", "(PARA (FN abc))"},
		{"[^abc ]", "(PARA (FN abc))"},
		{"[^abc\ndef]", "(PARA (FN abc SB def))"},
		{"[^abc\n\ndef]", "(PARA [^abc)(PARA def])"},
		{"[^abc[^def]]", "(PARA (FN abc (FN def)))"},
		{"[^abc]{-}", "(PARA (FN abc)[ATTR -])"},
	})
}

func TestEmbed(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"{", "(PARA {)"},
		{"{{", "(PARA {{)"},
		{"{{|", "(PARA {{|)"},
		{"{{}", "(PARA {{})"},
		{"{{|}", "(PARA {{|})"},
		{"{{}}", "(PARA {{}})"},
		{"{{|}}", "(PARA {{|}})"},
		{"{{ }}", "(PARA {{ SP }})"},
		{"{{\n}}", "(PARA {{ SB }})"},
		{"{{a }}", "(PARA {{a SP }})"},
		{"{{a\n}}", "(PARA {{a SB }})"},
		{"{{a}}", "(PARA (EMBED a))"},
		{"{{12345678901234}}", "(PARA (EMBED 12345678901234))"},
		{"{{ a}}", "(PARA (EMBED a))"},
		{"{{a}", "(PARA {{a})"},
		{"{{|a}}", "(PARA {{|a}})"},
		{"{{b|}}", "(PARA {{b|}})"},
		{"{{b|a}}", "(PARA (EMBED a b))"},
		{"{{b| a}}", "(PARA (EMBED a b))"},
		{"{{b|a}", "(PARA {{b|a})"},
		{"{{b\nc|a}}", "(PARA (EMBED a b SB c))"},
		{"{{b c|a#n}}", "(PARA (EMBED a#n b SP c))"},
		{"{{a}}{go}", "(PARA (EMBED a)[ATTR go])"},
		{"{{{{a}}|b}}", "(PARA (EMBED %7B%7Ba) |b}})"},
		{"{{\\|}}", "(PARA (EMBED %5C%7C))"},
		{"{{\\||a}}", "(PARA (EMBED a |))"},
		{"{{b\\||a}}", "(PARA (EMBED a b|))"},
		{"{{b\\|c|a}}", "(PARA (EMBED a b|c))"},
		{"{{\\}}}", "(PARA (EMBED %5C%7D))"},
		{"{{\\}|a}}", "(PARA (EMBED a }))"},
		{"{{b\\}|a}}", "(PARA (EMBED a b}))"},
		{"{{\\}\\||a}}", "(PARA (EMBED a }|))"},
		{"{{http://a|http://a}}", "(PARA (EMBED http://a http://a))"},
	})
}

func TestTag(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"#", "(PARA #)"},
		{"##", "(PARA ##)"},
		{"###", "(PARA ###)"},
		{"#tag", "(PARA #tag#)"},
		{"#tag,", "(PARA #tag# ,)"},
		{"#t-g ", "(PARA #t-g#)"},
		{"#t_g", "(PARA #t_g#)"},
	})
}

func TestMark(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"[!", "(PARA [!)"},
		{"[!\n", "(PARA [!)"},
		{"[!]", "(PARA (MARK #*))"},
		{"[!][!]", "(PARA (MARK #*) (MARK #*-1))"},
		{"[! ]", "(PARA [! SP ])"},
		{"[!a]", "(PARA (MARK \"a\" #a))"},
		{"[!a][!a]", "(PARA (MARK \"a\" #a) (MARK \"a\" #a-1))"},
		{"[!a ]", "(PARA [!a SP ])"},
		{"[!a_]", "(PARA (MARK \"a_\" #a))"},
		{"[!a_][!a]", "(PARA (MARK \"a_\" #a) (MARK \"a\" #a-1))"},
		{"[!a-b]", "(PARA (MARK \"a-b\" #a-b))"},
	})
}

func TestComment(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"%", "(PARA %)"},
		{"%%", "(PARA {%})"},
		{"%\n", "(PARA %)"},
		{"%%\n", "(PARA {%})"},
		{"%%a", "(PARA {% a})"},
		{"%%%a", "(PARA {% a})"},
		{"%% a", "(PARA {% a})"},
		{"%%%  a", "(PARA {% a})"},
		{"%% % a", "(PARA {% % a})"},
		{"%%a", "(PARA {% a})"},
		{"a%%b", "(PARA a {% b})"},
		{"a %%b", "(PARA a {% b})"},
		{" %%b", "(PARA {% b})"},
		{"%%b ", "(PARA {% b })"},
		{"100%", "(PARA 100%)"},
	})
}

func TestFormat(t *testing.T) {
	t.Parallel()
	// Not for Insert / '>', because collision with quoted list
	for _, ch := range []string{"_", "*", "~", "'", "^", ",", "<", "\"", ":"} {
		checkTcs(t, replace(ch, TestCases{
			{"$", "(PARA $)"},
			{"$$", "(PARA $$)"},
			{"$$$", "(PARA $$$)"},
			{"$$$$", "(PARA {$})"},
		}))
	}
	for _, ch := range []string{"_", "*", ">", "~", "'", "^", ",", "<", "\"", ":"} {
		checkTcs(t, replace(ch, TestCases{
			{"$$a$$", "(PARA {$ a})"},
			{"$$a$$$", "(PARA {$ a} $)"},
			{"$$$a$$", "(PARA {$ $a})"},
			{"$$$a$$$", "(PARA {$ $a} $)"},
			{"$\\$", "(PARA $$)"},
			{"$\\$$", "(PARA $$$)"},
			{"$$\\$", "(PARA $$$)"},
			{"$$a\\$$", "(PARA $$a$$)"},
			{"$$a$\\$", "(PARA $$a$$)"},
			{"$$a\\$$$", "(PARA {$ a$})"},
			{"$$a\na$$", "(PARA {$ a SB a})"},
			{"$$a\n\na$$", "(PARA $$a)(PARA a$$)"},
			{"$$a$${go}", "(PARA {$ a}[ATTR go])"},
		}))
	}
	checkTcs(t, TestCases{
		{"__****__", "(PARA {_ {*}})"},
		{"__**a**__", "(PARA {_ {* a}})"},
		{"__**__**", "(PARA __ {* __})"},
	})
}

func TestLiteral(t *testing.T) {
	t.Parallel()
	for _, ch := range []string{"@", "`", "+", "="} {
		checkTcs(t, replace(ch, TestCases{
			{"$", "(PARA $)"},
			{"$$", "(PARA $$)"},
			{"$$$", "(PARA $$$)"},
			{"$$$$", "(PARA {$})"},
			{"$$a$$", "(PARA {$ a})"},
			{"$$a$$$", "(PARA {$ a} $)"},
			{"$$$a$$", "(PARA {$ $a})"},
			{"$$$a$$$", "(PARA {$ $a} $)"},
			{"$\\$", "(PARA $$)"},
			{"$\\$$", "(PARA $$$)"},
			{"$$\\$", "(PARA $$$)"},
			{"$$a\\$$", "(PARA $$a$$)"},
			{"$$a$\\$", "(PARA $$a$$)"},
			{"$$a\\$$$", "(PARA {$ a$})"},
			{"$$a$${go}", "(PARA {$ a}[ATTR go])"},
		}))
	}
	checkTcs(t, TestCases{
		{"++````++", "(PARA {+ ````})"},
		{"++``a``++", "(PARA {+ ``a``})"},
		{"++``++``", "(PARA {+ ``} ``)"},
		{"++\\+++", "(PARA {+ +})"},
	})
}

func TestMixFormatCode(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"__abc__\n**def**", "(PARA {_ abc} SB {* def})"},
		{"++abc++\n==def==", "(PARA {+ abc} SB {= def})"},
		{"__abc__\n==def==", "(PARA {_ abc} SB {= def})"},
		{"__abc__\n``def``", "(PARA {_ abc} SB {` def})"},
		{"\"\"ghi\"\"\n::abc::\n``def``\n", "(PARA {\" ghi} SB {: abc} SB {` def})"},
	})
}

func TestNDash(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"--", "(PARA \u2013)"},
		{"a--b", "(PARA a\u2013b)"},
	})
}

func TestEntity(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"&", "(PARA &)"},
		{"&;", "(PARA &;)"},
		{"&#;", "(PARA &#;)"},
		{"&#1a;", "(PARA & #1a# ;)"},
		{"&#x;", "(PARA & #x# ;)"},
		{"&#x0z;", "(PARA & #x0z# ;)"},
		{"&1;", "(PARA &1;)"},
		// Good cases
		{"&lt;", "(PARA <)"},
		{"&#48;", "(PARA 0)"},
		{"&#x4A;", "(PARA J)"},
		{"&#X4a;", "(PARA J)"},
		{"&hellip;", "(PARA \u2026)"},
		{"E: &amp;,&#13;;&#xa;.", "(PARA E: SP &,\r;\n.)"},
	})
}

func TestVerbatimZettel(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"@@@\n@@@", "(ZETTEL)"},
		{"@@@\nabc\n@@@", "(ZETTEL\nabc)"},
		{"@@@@draw\nabc\n@@@@", "(ZETTEL\nabc)[ATTR =draw]"},
	})
}

func TestVerbatimCode(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"```\n```", "(PROG)"},
		{"```\nabc\n```", "(PROG\nabc)"},
		{"```\nabc\n````", "(PROG\nabc)"},
		{"````\nabc\n````", "(PROG\nabc)"},
		{"````\nabc\n```\n````", "(PROG\nabc\n```)"},
		{"````go\nabc\n````", "(PROG\nabc)[ATTR =go]"},
	})
}

func TestVerbatimComment(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"%%%\n%%%", "(COMMENT)"},
		{"%%%\nabc\n%%%", "(COMMENT\nabc)"},
		{"%%%%go\nabc\n%%%%", "(COMMENT\nabc)[ATTR =go]"},
	})
}

func TestSpanRegion(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{":::\n:::", "(SPAN)"},
		{":::\nabc\n:::", "(SPAN (PARA abc))"},
		{":::\nabc\n::::", "(SPAN (PARA abc))"},
		{"::::\nabc\n::::", "(SPAN (PARA abc))"},
		{"::::\nabc\n:::\ndef\n:::\n::::", "(SPAN (PARA abc)(SPAN (PARA def)))"},
		{":::{go}\n:::", "(SPAN)[ATTR go]"},
		{":::\nabc\n::: def ", "(SPAN (PARA abc) (LINE def))"},
	})
}

func TestQuoteRegion(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"<<<\n<<<", "(QUOTE)"},
		{"<<<\nabc\n<<<", "(QUOTE (PARA abc))"},
		{"<<<\nabc\n<<<<", "(QUOTE (PARA abc))"},
		{"<<<<\nabc\n<<<<", "(QUOTE (PARA abc))"},
		{"<<<<\nabc\n<<<\ndef\n<<<\n<<<<", "(QUOTE (PARA abc)(QUOTE (PARA def)))"},
		{"<<<go\n<<<", "(QUOTE)[ATTR =go]"},
		{"<<<\nabc\n<<< def ", "(QUOTE (PARA abc) (LINE def))"},
	})
}

func TestVerseRegion(t *testing.T) {
	t.Parallel()
	checkTcs(t, replace("\"", TestCases{
		{"$$$\n$$$", "(VERSE)"},
		{"$$$\nabc\n$$$", "(VERSE (PARA abc))"},
		{"$$$\nabc\n$$$$", "(VERSE (PARA abc))"},
		{"$$$$\nabc\n$$$$", "(VERSE (PARA abc))"},
		{"$$$\nabc\ndef\n$$$", "(VERSE (PARA abc HB def))"},
		{"$$$$\nabc\n$$$\ndef\n$$$\n$$$$", "(VERSE (PARA abc)(VERSE (PARA def)))"},
		{"$$$go\n$$$", "(VERSE)[ATTR =go]"},
		{"$$$\nabc\n$$$ def ", "(VERSE (PARA abc) (LINE def))"},
	}))
}

func TestHeading(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"=h", "(PARA =h)"},
		{"= h", "(PARA = SP h)"},
		{"==h", "(PARA ==h)"},
		{"== h", "(PARA == SP h)"},
		{"===h", "(PARA ===h)"},
		{"=== h", "(H1 h #h)"},
		{"===  h", "(H1 h #h)"},
		{"==== h", "(H2 h #h)"},
		{"===== h", "(H3 h #h)"},
		{"====== h", "(H4 h #h)"},
		{"======= h", "(H5 h #h)"},
		{"======== h", "(H5 h #h)"},
		{"=", "(PARA =)"},
		{"=== h=__=a__", "(H1 h= {_ =a} #h-a)"},
		{"=\n", "(PARA =)"},
		{"a=", "(PARA a=)"},
		{" =", "(PARA =)"},
		{"=== h\na", "(H1 h #h)(PARA a)"},
		{"=== h i {-}", "(H1 h SP i #h-i)[ATTR -]"},
		{"=== h {{a}}", "(H1 h SP (EMBED a) #h)"},
		{"=== h{{a}}", "(H1 h (EMBED a) #h)"},
		{"=== {{a}}", "(H1 (EMBED a))"},
		{"=== h {{a}}{-}", "(H1 h SP (EMBED a)[ATTR -] #h)"},
		{"=== h {{a}} {-}", "(H1 h SP (EMBED a) #h)[ATTR -]"},
		{"=== h {-}{{a}}", "(H1 h #h)[ATTR -]"},
		{"=== h{id=abc}", "(H1 h #h)[ATTR id=abc]"},
		{"=== h\n=== h", "(H1 h #h)(H1 h #h-1)"},
	})
}

func TestHRule(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"-", "(PARA -)"},
		{"---", "(HR)"},
		{"----", "(HR)"},
		{"---A", "(HR)[ATTR =A]"},
		{"---A-", "(HR)[ATTR =A-]"},
		{"-1", "(PARA -1)"},
		{"2-1", "(PARA 2-1)"},
		{"---  {  go  }  ", "(HR)[ATTR go]"},
		{"---  {  .go  }  ", "(HR)[ATTR class=go]"},
	})
}

func TestList(t *testing.T) {
	t.Parallel()
	// No ">" in the following, because quotation lists may have empty items.
	for _, ch := range []string{"*", "#"} {
		checkTcs(t, replace(ch, TestCases{
			{"$", "(PARA $)"},
			{"$$", "(PARA $$)"},
			{"$$$", "(PARA $$$)"},
			{"$ ", "(PARA $)"},
			{"$$ ", "(PARA $$)"},
			{"$$$ ", "(PARA $$$)"},
		}))
	}
	checkTcs(t, TestCases{
		{"* abc", "(UL {(PARA abc)})"},
		{"** abc", "(UL {(UL {(PARA abc)})})"},
		{"*** abc", "(UL {(UL {(UL {(PARA abc)})})})"},
		{"**** abc", "(UL {(UL {(UL {(UL {(PARA abc)})})})})"},
		{"** abc\n**** def", "(UL {(UL {(PARA abc)(UL {(UL {(PARA def)})})})})"},
		{"* abc\ndef", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n def", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n* def", "(UL {(PARA abc)} {(PARA def)})"},
		{"* abc\n  def", "(UL {(PARA abc SB def)})"},
		{"* abc\n   def", "(UL {(PARA abc SB def)})"},
		{"* abc\n\ndef", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n\n def", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n\n  def", "(UL {(PARA abc)(PARA def)})"},
		{"* abc\n\n   def", "(UL {(PARA abc)(PARA def)})"},
		{"* abc\n** def", "(UL {(PARA abc)(UL {(PARA def)})})"},
		{"* abc\n** def\n* ghi", "(UL {(PARA abc)(UL {(PARA def)})} {(PARA ghi)})"},
		{"* abc\n\n  def\n* ghi", "(UL {(PARA abc)(PARA def)} {(PARA ghi)})"},
		{"* abc\n** def\n   ghi\n  jkl", "(UL {(PARA abc)(UL {(PARA def SB ghi)})(PARA jkl)})"},

		// A list does not last beyond a region
		{":::\n# abc\n:::\n# def", "(SPAN (OL {(PARA abc)}))(OL {(PARA def)})"},

		// A HRule creates a new list
		{"* abc\n---\n* def", "(UL {(PARA abc)})(HR)(UL {(PARA def)})"},

		// Changing list type adds a new list
		{"* abc\n# def", "(UL {(PARA abc)})(OL {(PARA def)})"},

		// Quotation lists may have empty items
		{">", "(QL {})"},
	})
}

func TestEnumAfterPara(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"abc\n* def", "(PARA abc)(UL {(PARA def)})"},
		{"abc\n*def", "(PARA abc SB *def)"},
	})
}

func TestDefinition(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{";", "(PARA ;)"},
		{"; ", "(PARA ;)"},
		{"; abc", "(DL (DT abc))"},
		{"; abc\ndef", "(DL (DT abc))(PARA def)"},
		{"; abc\n def", "(DL (DT abc))(PARA def)"},
		{"; abc\n  def", "(DL (DT abc SB def))"},
		{":", "(PARA :)"},
		{": ", "(PARA :)"},
		{": abc", "(PARA : SP abc)"},
		{"; abc\n: def", "(DL (DT abc) (DD (PARA def)))"},
		{"; abc\n: def\nghi", "(DL (DT abc) (DD (PARA def)))(PARA ghi)"},
		{"; abc\n: def\n ghi", "(DL (DT abc) (DD (PARA def)))(PARA ghi)"},
		{"; abc\n: def\n  ghi", "(DL (DT abc) (DD (PARA def SB ghi)))"},
		{"; abc\n: def\n\n  ghi", "(DL (DT abc) (DD (PARA def)(PARA ghi)))"},
		{"; abc\n:", "(DL (DT abc))(PARA :)"},
		{"; abc\n: def\n: ghi", "(DL (DT abc) (DD (PARA def)) (DD (PARA ghi)))"},
		{"; abc\n: def\n; ghi\n: jkl", "(DL (DT abc) (DD (PARA def)) (DT ghi) (DD (PARA jkl)))"},
	})
}

func TestTable(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"|", "(TAB (TR))"},
		{"||", "(TAB (TR (TD)))"},
		{"| |", "(TAB (TR (TD)))"},
		{"|a", "(TAB (TR (TD a)))"},
		{"|a|", "(TAB (TR (TD a)))"},
		{"|a| ", "(TAB (TR (TD a)(TD)))"},
		{"|a|b", "(TAB (TR (TD a)(TD b)))"},
		{"|a|b\n|c|d", "(TAB (TR (TD a)(TD b))(TR (TD c)(TD d)))"},
		{"|%", ""},
		{"|a|b\n|%---\n|c|d", "(TAB (TR (TD a)(TD b))(TR (TD c)(TD d)))"},
		{"|a|b\n|c", "(TAB (TR (TD a)(TD b))(TR (TD c)(TD)))"},
	})
}

func TestBlockEmbed(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"{{{a}}}", "(TRANSCLUDE a)"},
		{"{{{a}}}b", "(TRANSCLUDE a)"},
		{"{{{a}}}}", "(TRANSCLUDE a)"},
		{"{{{a\\}}}}", "(TRANSCLUDE a%5C%7D)"},
		{"{{{a\\}}}}b", "(TRANSCLUDE a%5C%7D)"},
		{"{{{a}}", "(PARA (EMBED %7Ba))"},
	})
}

func TestBlockAttr(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{":::go\n:::", "(SPAN)[ATTR =go]"},
		{":::go=\n:::", "(SPAN)[ATTR =go]"},
		{":::{}\n:::", "(SPAN)"},
		{":::{ }\n:::", "(SPAN)"},
		{":::{.go}\n:::", "(SPAN)[ATTR class=go]"},
		{":::{=go}\n:::", "(SPAN)[ATTR =go]"},
		{":::{go}\n:::", "(SPAN)[ATTR go]"},
		{":::{go=py}\n:::", "(SPAN)[ATTR go=py]"},
		{":::{.go=py}\n:::", "(SPAN)"},
		{":::{go=}\n:::", "(SPAN)[ATTR go]"},
		{":::{.go=}\n:::", "(SPAN)"},
		{":::{go py}\n:::", "(SPAN)[ATTR go py]"},
		{":::{go\npy}\n:::", "(SPAN (PARA py}))"},
		{":::{.go py}\n:::", "(SPAN)[ATTR class=go py]"},
		{":::{go .py}\n:::", "(SPAN)[ATTR class=py go]"},
		{":::{.go py=3}\n:::", "(SPAN)[ATTR class=go py=3]"},
		{":::  {  go  }  \n:::", "(SPAN)[ATTR go]"},
		{":::  {  .go  }  \n:::", "(SPAN)[ATTR class=go]"},
	})
	checkTcs(t, replace("\"", TestCases{
		{":::{py=3}\n:::", "(SPAN)[ATTR py=3]"},
		{":::{py=$2 3$}\n:::", "(SPAN)[ATTR py=$2 3$]"},
		{":::{py=$2\\$3$}\n:::", "(SPAN)[ATTR py=2$3]"},
		{":::{py=2$3}\n:::", "(SPAN)[ATTR py=2$3]"},
		{":::{py=$2\n3$}\n:::", "(SPAN (PARA 3$}))"},
		{":::{py=$2 3}\n:::", "(SPAN)"},
		{":::{py=2 py=3}\n:::", "(SPAN)[ATTR py=$2 3$]"},
		{":::{.go .py}\n:::", "(SPAN)[ATTR class=$go py$]"},
		{":::{go go}\n:::", "(SPAN)[ATTR go]"},
		{":::{=py =go}\n:::", "(SPAN)[ATTR =go]"},
	}))
}

func TestInlineAttr(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"::a::{}", "(PARA {: a})"},
		{"::a::{ }", "(PARA {: a})"},
		{"::a::{.go}", "(PARA {: a}[ATTR class=go])"},
		{"::a::{=go}", "(PARA {: a}[ATTR =go])"},
		{"::a::{go}", "(PARA {: a}[ATTR go])"},
		{"::a::{go=py}", "(PARA {: a}[ATTR go=py])"},
		{"::a::{.go=py}", "(PARA {: a} {.go=py})"},
		{"::a::{go=}", "(PARA {: a}[ATTR go])"},
		{"::a::{.go=}", "(PARA {: a} {.go=})"},
		{"::a::{go py}", "(PARA {: a}[ATTR go py])"},
		{"::a::{go\npy}", "(PARA {: a}[ATTR go py])"},
		{"::a::{.go py}", "(PARA {: a}[ATTR class=go py])"},
		{"::a::{go .py}", "(PARA {: a}[ATTR class=py go])"},
		{"::a::{  \n go \n .py\n  \n}", "(PARA {: a}[ATTR class=py go])"},
		{"::a::{  \n go \n .py\n\n}", "(PARA {: a}[ATTR class=py go])"},
		{"::a::{\ngo\n}", "(PARA {: a}[ATTR go])"},
	})
	checkTcs(t, replace("\"", TestCases{
		{"::a::{py=3}", "(PARA {: a}[ATTR py=3])"},
		{"::a::{py=$2 3$}", "(PARA {: a}[ATTR py=$2 3$])"},
		{"::a::{py=$2\\$3$}", "(PARA {: a}[ATTR py=2$3])"},
		{"::a::{py=2$3}", "(PARA {: a}[ATTR py=2$3])"},
		{"::a::{py=$2\n3$}", "(PARA {: a}[ATTR py=$2 3$])"},
		{"::a::{py=$2 3}", "(PARA {: a} {py=$2 SP 3})"},

		{"::a::{py=2 py=3}", "(PARA {: a}[ATTR py=$2 3$])"},
		{"::a::{.go .py}", "(PARA {: a}[ATTR class=$go py$])"},
	}))
}

func TestTemp(t *testing.T) {
	t.Parallel()
	checkTcs(t, TestCases{
		{"", ""},
	})
}

// --------------------------------------------------------------------------

// TestVisitor serializes the abstract syntax tree to a string.
type TestVisitor struct {
	buf bytes.Buffer
}

func (tv *TestVisitor) String() string { return tv.buf.String() }

func (tv *TestVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.InlineListNode:
		tv.visitInlineList(n)
	case *ast.ParaNode:
		tv.buf.WriteString("(PARA")
		ast.Walk(tv, n.Inlines)
		tv.buf.WriteByte(')')
	case *ast.VerbatimNode:
		code, ok := mapVerbatimKind[n.Kind]
		if !ok {
			panic(fmt.Sprintf("Unknown verbatim code %v", n.Kind))
		}
		tv.buf.WriteString(code)
		if len(n.Content) > 0 {
			tv.buf.WriteByte('\n')
			tv.buf.Write(n.Content)
		}
		tv.buf.WriteByte(')')
		tv.visitAttributes(n.Attrs)
	case *ast.RegionNode:
		code, ok := mapRegionKind[n.Kind]
		if !ok {
			panic(fmt.Sprintf("Unknown region code %v", n.Kind))
		}
		tv.buf.WriteString(code)
		if n.Blocks != nil && len(n.Blocks.List) > 0 {
			tv.buf.WriteByte(' ')
			ast.Walk(tv, n.Blocks)
		}
		if n.Inlines != nil {
			tv.buf.WriteString(" (LINE")
			ast.Walk(tv, n.Inlines)
			tv.buf.WriteByte(')')
		}
		tv.buf.WriteByte(')')
		tv.visitAttributes(n.Attrs)
	case *ast.HeadingNode:
		fmt.Fprintf(&tv.buf, "(H%d", n.Level)
		ast.Walk(tv, n.Inlines)
		if n.Fragment != "" {
			tv.buf.WriteString(" #")
			tv.buf.WriteString(n.Fragment)
		}
		tv.buf.WriteByte(')')
		tv.visitAttributes(n.Attrs)
	case *ast.HRuleNode:
		tv.buf.WriteString("(HR)")
		tv.visitAttributes(n.Attrs)
	case *ast.NestedListNode:
		tv.buf.WriteString(mapNestedListKind[n.Kind])
		for _, item := range n.Items {
			tv.buf.WriteString(" {")
			ast.WalkItemSlice(tv, item)
			tv.buf.WriteByte('}')
		}
		tv.buf.WriteByte(')')
	case *ast.DescriptionListNode:
		tv.buf.WriteString("(DL")
		for _, def := range n.Descriptions {
			tv.buf.WriteString(" (DT")
			ast.Walk(tv, def.Term)
			tv.buf.WriteByte(')')
			for _, b := range def.Descriptions {
				tv.buf.WriteString(" (DD ")
				ast.WalkDescriptionSlice(tv, b)
				tv.buf.WriteByte(')')
			}
		}
		tv.buf.WriteByte(')')
	case *ast.TableNode:
		tv.buf.WriteString("(TAB")
		if len(n.Header) > 0 {
			tv.buf.WriteString(" (TR")
			for _, cell := range n.Header {
				tv.buf.WriteString(" (TH")
				tv.buf.WriteString(alignString[cell.Align])
				ast.Walk(tv, cell.Inlines)
				tv.buf.WriteString(")")
			}
			tv.buf.WriteString(")")
		}
		if len(n.Rows) > 0 {
			tv.buf.WriteString(" ")
			for _, row := range n.Rows {
				tv.buf.WriteString("(TR")
				for i, cell := range row {
					if i == 0 {
						tv.buf.WriteString(" ")
					}
					tv.buf.WriteString("(TD")
					tv.buf.WriteString(alignString[cell.Align])
					ast.Walk(tv, cell.Inlines)
					tv.buf.WriteString(")")
				}
				tv.buf.WriteString(")")
			}
		}
		tv.buf.WriteString(")")
	case *ast.TranscludeNode:
		fmt.Fprintf(&tv.buf, "(TRANSCLUDE %v)", n.Ref)
	case *ast.BLOBNode:
		tv.buf.WriteString("(BLOB ")
		tv.buf.WriteString(n.Syntax)
		tv.buf.WriteString(")")
	case *ast.TextNode:
		tv.buf.WriteString(n.Text)
	case *ast.TagNode:
		tv.buf.WriteByte('#')
		tv.buf.WriteString(n.Tag)
		tv.buf.WriteByte('#')
	case *ast.SpaceNode:
		if len(n.Lexeme) == 1 {
			tv.buf.WriteString("SP")
		} else {
			fmt.Fprintf(&tv.buf, "SP%d", len(n.Lexeme))
		}
	case *ast.BreakNode:
		if n.Hard {
			tv.buf.WriteString("HB")
		} else {
			tv.buf.WriteString("SB")
		}
	case *ast.LinkNode:
		fmt.Fprintf(&tv.buf, "(LINK %v", n.Ref)
		ast.Walk(tv, n.Inlines)
		tv.buf.WriteByte(')')
		tv.visitAttributes(n.Attrs)
	case *ast.EmbedRefNode:
		fmt.Fprintf(&tv.buf, "(EMBED %v", n.Ref)
		if n.Inlines != nil {
			ast.Walk(tv, n.Inlines)
		}
		tv.buf.WriteByte(')')
		tv.visitAttributes(n.Attrs)
	case *ast.EmbedBLOBNode:
		panic("TODO: zmktest blob")
	case *ast.CiteNode:
		fmt.Fprintf(&tv.buf, "(CITE %s", n.Key)
		if n.Inlines != nil {
			ast.Walk(tv, n.Inlines)
		}
		tv.buf.WriteByte(')')
		tv.visitAttributes(n.Attrs)
	case *ast.FootnoteNode:
		tv.buf.WriteString("(FN")
		ast.Walk(tv, n.Inlines)
		tv.buf.WriteByte(')')
		tv.visitAttributes(n.Attrs)
	case *ast.MarkNode:
		tv.buf.WriteString("(MARK")
		if n.Text != "" {
			tv.buf.WriteString(" \"")
			tv.buf.WriteString(n.Text)
			tv.buf.WriteByte('"')
		}
		if n.Fragment != "" {
			tv.buf.WriteString(" #")
			tv.buf.WriteString(n.Fragment)
		}
		tv.buf.WriteByte(')')
	case *ast.FormatNode:
		fmt.Fprintf(&tv.buf, "{%c", mapFormatKind[n.Kind])
		ast.Walk(tv, n.Inlines)
		tv.buf.WriteByte('}')
		tv.visitAttributes(n.Attrs)
	case *ast.LiteralNode:
		code, ok := mapLiteralKind[n.Kind]
		if !ok {
			panic(fmt.Sprintf("No element for code %v", n.Kind))
		}
		tv.buf.WriteByte('{')
		tv.buf.WriteRune(code)
		if n.Text != "" {
			tv.buf.WriteByte(' ')
			tv.buf.WriteString(n.Text)
		}
		tv.buf.WriteByte('}')
		tv.visitAttributes(n.Attrs)
	default:
		return tv
	}
	return nil
}

var mapVerbatimKind = map[ast.VerbatimKind]string{
	ast.VerbatimZettel:  "(ZETTEL",
	ast.VerbatimProg:    "(PROG",
	ast.VerbatimComment: "(COMMENT",
}

var mapRegionKind = map[ast.RegionKind]string{
	ast.RegionSpan:  "(SPAN",
	ast.RegionQuote: "(QUOTE",
	ast.RegionVerse: "(VERSE",
}

var mapNestedListKind = map[ast.NestedListKind]string{
	ast.NestedListOrdered:   "(OL",
	ast.NestedListUnordered: "(UL",
	ast.NestedListQuote:     "(QL",
}

var alignString = map[ast.Alignment]string{
	ast.AlignDefault: "",
	ast.AlignLeft:    "l",
	ast.AlignCenter:  "c",
	ast.AlignRight:   "r",
}

var mapFormatKind = map[ast.FormatKind]rune{
	ast.FormatEmph:      '_',
	ast.FormatStrong:    '*',
	ast.FormatInsert:    '>',
	ast.FormatDelete:    '~',
	ast.FormatMonospace: '\'',
	ast.FormatSuper:     '^',
	ast.FormatSub:       ',',
	ast.FormatQuote:     '"',
	ast.FormatQuotation: '<',
	ast.FormatSpan:      ':',
}

var mapLiteralKind = map[ast.LiteralKind]rune{
	ast.LiteralZettel:  '@',
	ast.LiteralProg:    '`',
	ast.LiteralKeyb:    '+',
	ast.LiteralOutput:  '=',
	ast.LiteralComment: '%',
}

func (tv *TestVisitor) visitInlineList(iln *ast.InlineListNode) {
	for _, in := range iln.List {
		tv.buf.WriteByte(' ')
		ast.Walk(tv, in)
	}
}

func (tv *TestVisitor) visitAttributes(a *ast.Attributes) {
	if a.IsEmpty() {
		return
	}
	tv.buf.WriteString("[ATTR")

	keys := make([]string, 0, len(a.Attrs))
	for k := range a.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		tv.buf.WriteByte(' ')
		tv.buf.WriteString(k)
		v := a.Attrs[k]
		if len(v) > 0 {
			tv.buf.WriteByte('=')
			if strings.ContainsRune(v, ' ') {
				tv.buf.WriteByte('"')
				tv.buf.WriteString(v)
				tv.buf.WriteByte('"')
			} else {
				tv.buf.WriteString(v)
			}
		}
	}

	tv.buf.WriteByte(']')
}

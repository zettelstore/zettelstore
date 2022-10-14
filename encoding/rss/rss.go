//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package rss provides a RSS encoding.
package rss

import (
	"bytes"
	"context"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/query"
	"zettelstore.de/z/strfun"
)

const ContentType = "application/rss+xml"

type Configuration struct {
	Title            string
	Language         string
	Copyright        string
	Generator        string
	NewURLBuilderAbs func() *api.URLBuilder
}

func (c *Configuration) Setup(ctx context.Context, cfg config.Config) {
	baseURL := kernel.Main.GetConfig(kernel.WebService, kernel.WebBaseURL).(string)
	defVals := cfg.AddDefaultValues(ctx, &meta.Meta{})

	c.Title = cfg.GetSiteName()
	c.Language = defVals.GetDefault(api.KeyLang, "")
	c.Copyright = defVals.GetDefault(api.KeyCopyright, "")
	c.Generator = (kernel.Main.GetConfig(kernel.CoreService, kernel.CoreProgname).(string) +
		" " +
		kernel.Main.GetConfig(kernel.CoreService, kernel.CoreVersion).(string))
	c.NewURLBuilderAbs = func() *api.URLBuilder { return api.NewURLBuilder(baseURL, 'h') }
}

func (c *Configuration) Marshal(q *query.Query, ml []*meta.Meta) []byte {
	textEnc := textenc.Create()
	maxPublished := time.Date(1, time.January, 1, 0, 0, 0, 0, time.Local)
	rssPublished := ""
	if maxPublished.Year() > 1 {
		rssPublished = maxPublished.UTC().Format(time.RFC1123Z)
	}
	atomLink := ""
	if s := q.String(); s != "" {
		atomLink = c.NewURLBuilderAbs().AppendQuery(s).String()
	}
	var buf bytes.Buffer
	buf.WriteString(`<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">` + "\n<channel>\n")
	writeXMLTag(&buf, "  ", "title", c.Title)
	writeXMLTag(&buf, "  ", "link", c.NewURLBuilderAbs().String())
	writeXMLTag(&buf, "  ", "description", "")
	writeXMLTag(&buf, "  ", "language", c.Language)
	writeXMLTag(&buf, "  ", "copyright", c.Copyright)
	if rssPublished != "" {
		writeXMLTag(&buf, "  ", "pubDate", rssPublished)
		writeXMLTag(&buf, "  ", "lastBuildDate", rssPublished)
	}
	writeXMLTag(&buf, "  ", "generator", c.Generator)
	buf.WriteString("  <docs>https://www.rssboard.org/rss-specification</docs>\n")
	if atomLink != "" {
		buf.WriteString(`  <atom:link href="`)
		strfun.XMLEscape(&buf, atomLink)
		buf.WriteString(`" rel="self" type="application/rss+xml"></atom:link>` + "\n")
	}
	for _, m := range ml {
		var title bytes.Buffer
		titleIns := parser.ParseMetadata(m.GetTitle())
		if _, err := textEnc.WriteInlines(&title, &titleIns); err != nil {
			title.Reset()
			title.WriteString(m.GetTitle())
		}

		itemPublished := ""
		if val, found := m.Get(api.KeyPublished); found {
			if published, err := time.ParseInLocation(id.ZidLayout, val, time.Local); err == nil {
				itemPublished = published.UTC().Format(time.RFC1123Z)
				if maxPublished.Before(published) {
					maxPublished = published
				}
			}
		}

		link := c.NewURLBuilderAbs().SetZid(api.ZettelID(m.Zid.String())).String()

		buf.WriteString("  <item>\n")
		writeXMLTag(&buf, "    ", "title", title.String())
		writeXMLTag(&buf, "    ", "link", link)
		writeXMLTag(&buf, "    ", "guid", link)
		if itemPublished != "" {
			writeXMLTag(&buf, "    ", "pubDate", itemPublished)
		}
		for _, cat := range getCategories(m) {
			writeXMLTag(&buf, "    ", "category", cat)
		}
		buf.WriteString("  </item>\n")
	}

	buf.WriteString("</channel>\n</rss>")
	return buf.Bytes()
}

func writeXMLTag(buf *bytes.Buffer, prefix, tag, value string) {
	buf.WriteString(prefix)
	buf.WriteByte('<')
	buf.WriteString(tag)
	buf.WriteByte('>')
	strfun.XMLEscape(buf, value)
	buf.WriteString("</")
	buf.WriteString(tag)
	buf.WriteString(">\n")
}

func getCategories(m *meta.Meta) []string {
	if m == nil {
		return nil
	}
	if tags, found := m.GetList(api.KeyTags); found && len(tags) > 0 {
		result := make([]string, 0, len(tags))
		for _, tag := range tags {
			for len(tag) > 0 && tag[0] == '#' {
				tag = tag[1:]
			}
			if tag != "" {
				result = append(result, tag)
			}
		}
		return result
	}
	return nil
}

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
	"encoding/xml"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/parser"
)

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

func (c *Configuration) Marshal(ml []*meta.Meta) ([]byte, error) {
	textEnc := textenc.Create()
	rssItems := make([]*RssItem, 0, len(ml))
	maxPublished := time.Date(1, time.January, 1, 0, 0, 0, 0, time.Local)
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
		rssItems = append(rssItems, &RssItem{
			Title:       title.String(),
			Link:        c.NewURLBuilderAbs().SetZid(api.ZettelID(m.Zid.String())).String(),
			Description: "",
			Author:      "",
			GUID:        "https://zettelstore.de/guid/" + m.Zid.String(),
			PubDate:     itemPublished,
		})
	}

	rssPublished := ""
	if maxPublished.Year() > 1 {
		rssPublished = maxPublished.UTC().Format(time.RFC1123Z)
	}
	rssFeed := RssFeed{
		Version: "2.0",
		Channel: &RssChannel{
			Title:          c.Title,
			Link:           c.NewURLBuilderAbs().String(),
			Description:    "",
			Language:       c.Language,
			Copyright:      c.Copyright,
			ManagingEditor: "",
			WebMaster:      "",
			PubDate:        rssPublished,
			LastBuildDate:  rssPublished,
			Generator:      c.Generator,
			Docs:           "https://www.rssboard.org/rss-specification",
			Items:          rssItems,
		},
	}
	return xml.MarshalIndent(&rssFeed, "", "  ")
}

type (
	RssFeed struct {
		XMLName xml.Name `xml:"rss"`
		Version string   `xml:"version,attr"`
		Channel *RssChannel
	}
	RssChannel struct {
		XMLName        xml.Name   `xml:"channel"`
		Title          string     `xml:"title"`
		Link           string     `xml:"link"`
		Description    string     `xml:"description"`
		Language       string     `xml:"language,omitempty"`
		Copyright      string     `xml:"copyright,omitempty"`
		ManagingEditor string     `xml:"managingEditor,omitempty"`
		WebMaster      string     `xml:"webMaster,omitempty"`
		PubDate        string     `xml:"pubDate,omitempty"`       // RFC822
		LastBuildDate  string     `xml:"lastBuildDate,omitempty"` // RFC822
		Generator      string     `xml:"generator,omitempty"`
		Docs           string     `xml:"docs,omitempty"`
		Items          []*RssItem `xml:"item"`
	}
	RssItem struct {
		XMLName     xml.Name `xml:"item"`
		Title       string   `xml:"title"`
		Link        string   `xml:"link"`
		Description string   `xml:"description"`
		Author      string   `xml:"author,omitempty"`
		GUID        string   `xml:"guid,omitempty"`
		PubDate     string   `xml:"pubDate,omitempty"` // RFC822
	}
)

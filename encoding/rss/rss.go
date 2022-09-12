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
	"encoding/xml"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/parser"
)

type Configuration struct {
	Title            string
	NewURLBuilderAbs func(key byte) *api.URLBuilder
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
			Link:        c.NewURLBuilderAbs('h').SetZid(api.ZettelID(m.Zid.String())).String(),
			Description: "",
			Author:      "",
			Category:    "",
			Comments:    "",
			Enclosure:   nil,
			GUID:        "https://zettelstore.de/guid/" + m.Zid.String(),
			PubDate:     itemPublished,
			Source:      "",
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
			Link:           c.NewURLBuilderAbs('h').String(),
			Description:    "",
			Language:       "",
			Copyright:      "",
			ManagingEditor: "",
			WebMaster:      "",
			PubDate:        rssPublished,
			LastBuildDate:  "",
			Category:       "",
			Generator:      "Zettelstore",
			Docs:           "https://www.rssboard.org/rss-specification",
			Cloud:          "",
			TTL:            0,
			Image:          nil,
			Rating:         "",
			TextInput:      nil,
			SkipHours:      "",
			SkipDays:       "",
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
		XMLName        xml.Name `xml:"channel"`
		Title          string   `xml:"title"`
		Link           string   `xml:"link"`
		Description    string   `xml:"description"`
		Language       string   `xml:"language,omitempty"`
		Copyright      string   `xml:"copyright,omitempty"`
		ManagingEditor string   `xml:"managingEditor,omitempty"`
		WebMaster      string   `xml:"webMaster,omitempty"`
		PubDate        string   `xml:"pubDate,omitempty"`       // RFC822
		LastBuildDate  string   `xml:"lastBuildDate,omitempty"` // RFC822
		Category       string   `xml:"category,omitempty"`
		Generator      string   `xml:"generator,omitempty"`
		Docs           string   `xml:"docs,omitempty"`
		Cloud          string   `xml:"cloud,omitempty"`
		TTL            int      `xml:"ttl,omitempty"`
		Image          *RssImage
		Rating         string `xml:"rating,omitempty"`
		TextInput      *RssTextInput
		SkipHours      string     `xml:"skipHours,omitempty"`
		SkipDays       string     `xml:"skipDays,omitempty"`
		Items          []*RssItem `xml:"item"`
	}
	RssImage struct {
		XMLName xml.Name `xml:"image"`
		URL     string   `xml:"url"`
		Title   string   `xml:"title"`
		Link    string   `xml:"link"`
		Width   int      `xml:"width,omitempty"`
		Height  int      `xml:"height,omitempty"`
	}
	RssTextInput struct {
		XMLName     xml.Name `xml:"textInput"`
		Title       string   `xml:"title"`
		Description string   `xml:"description"`
		Name        string   `xml:"name"`
		Link        string   `xml:"link"`
	}
	RssItem struct {
		XMLName     xml.Name `xml:"item"`
		Title       string   `xml:"title"`
		Link        string   `xml:"link"`
		Description string   `xml:"description"`
		Author      string   `xml:"author,omitempty"`
		Category    string   `xml:"category,omitempty"`
		Comments    string   `xml:"comments,omitempty"`
		Enclosure   *RssEnclosure
		GUID        string `xml:"guid,omitempty"`
		PubDate     string `xml:"pubDate,omitempty"` // RFC822
		Source      string `xml:"source,omitempty"`
	}
	RssEnclosure struct {
		XMLName xml.Name `xml:"enclosure"`
		URL     string   `xml:"url,attr"`
		Length  string   `xml:"length,attr"`
		Type    string   `xml:"type,attr"`
	}
)

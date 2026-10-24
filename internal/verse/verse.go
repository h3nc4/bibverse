/*
 * Copyright (C) 2026  Henrique Almeida
 * This file is part of bibverse.
 *
 * bibverse is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * bibverse is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with bibverse.  If not, see <https://www.gnu.org/licenses/>.
 */

package verse

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// URL is the USCCB feed of the daily Mass readings, newest day first.
const URL = "https://bible.usccb.org/readings.rss"

const userAgent = "bibverse (+https://github.com/h3nc4/bibverse)"

// ErrNoVerse means no item in the feed has a verse before its Gospel.
var ErrNoVerse = errors.New("no verse in the feed")

// The acclamation is named Alleluia, or Verse Before the Gospel in Lent. The Easter Vigil
// has neither, and its last psalm stands in.
var preferred = [][]string{
	{"alleluia", "verse before the gospel"},
	{"responsorial psalm"},
}

type feed struct {
	Items []struct {
		Description string `xml:"description"`
	} `xml:"channel>item"`
}

type section struct {
	name  string
	ref   string
	lines []string
}

// Fetch downloads the feed at url and returns the verse of the newest day in it.
func Fetch(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch %s: %s", url, resp.Status)
	}

	return Parse(resp.Body)
}

// Parse returns the verse of the first item in the feed that has one. A day with several
// Masses lists an empty item beside the ones holding the readings.
func Parse(r io.Reader) (string, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	var f feed
	if err := dec.Decode(&f); err != nil {
		return "", fmt.Errorf("failed to parse feed: %w", err)
	}

	for _, item := range f.Items {
		if s, ok := pick(sections(item.Description)); ok {
			if v := s.verse(); v != "" {
				return v, nil
			}
		}
	}

	return "", ErrNoVerse
}

// sections splits a day's readings at each h4, which holds the name and a reference link.
func sections(readings string) []section {
	var (
		all       []section
		cur       *section
		line      strings.Builder
		inHeading bool
		inLink    bool
	)
	flush := func() {
		if cur != nil {
			cur.lines = append(cur.lines, line.String())
		}
		line.Reset()
	}

	z := html.NewTokenizer(strings.NewReader(readings))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			flush()
			return all
		}
		tok := z.Token()

		switch {
		case tok.DataAtom == atom.H4 && tt == html.StartTagToken:
			flush()
			all = append(all, section{})
			cur = &all[len(all)-1]
			inHeading = true
		case tok.DataAtom == atom.H4 && tt == html.EndTagToken:
			inHeading = false
		case tok.DataAtom == atom.A && inHeading:
			inLink = tt == html.StartTagToken
		case tt == html.TextToken && inHeading && inLink:
			cur.ref += tok.Data
		case tt == html.TextToken && inHeading:
			cur.name += tok.Data
		case tt == html.TextToken:
			line.WriteString(tok.Data)
		case tok.DataAtom == atom.Br, tok.DataAtom == atom.P && tt == html.EndTagToken:
			flush()
		}
	}
}

// pick returns the section sung just before the first Gospel, by the order in preferred.
func pick(all []section) (section, bool) {
	gospel := slices.IndexFunc(all, func(s section) bool { return s.is("gospel") })
	if gospel < 0 {
		return section{}, false
	}

	before := all[:gospel]
	for _, names := range preferred {
		for i := len(before) - 1; i >= 0; i-- {
			if slices.ContainsFunc(names, before[i].is) {
				return before[i], true
			}
		}
	}

	return section{}, false
}

func (s section) is(name string) bool {
	return strings.EqualFold(clean(s.name), name)
}

// verse joins the first stanza, the lines before a refrain or an alternative, and appends
// the reference. A non-scriptural acclamation has an empty link.
func (s section) verse() string {
	var out []string
	for _, l := range s.lines {
		l = clean(l)
		switch {
		case l == "":
		case strings.HasPrefix(l, "R."), strings.EqualFold(l, "or"):
			if len(out) > 0 {
				return s.cite(out)
			}
		default:
			out = append(out, l)
		}
	}
	return s.cite(out)
}

func (s section) cite(lines []string) string {
	text := strings.Join(lines, " ")
	if ref := clean(s.ref); text != "" && ref != "" {
		return text + " — " + ref
	}
	return text
}

// clean collapses every run of whitespace, the non-breaking spaces of the feed included.
func clean(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

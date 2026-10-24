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
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
)

// Sections trimmed from the USCCB feed and its archived copies from Lent and Holy Week 2026.
const (
	psalm = `<h4>Responsorial Psalm  <a href="https://bible.usccb.org/bible/psalms/122?1 ">Psalm 122:1-2</a></h4>
<div class="poetry"><p>R. (1) <strong>Let us go rejoicing to the house of the Lord.</strong><br>I rejoiced because they said to me,<br>“We will go up to the house of the LORD.”<br>R. <strong>Let us go rejoicing to the house of the Lord.</strong></p></div>`

	alleluia = `<h4>Alleluia  <a href="https://bible.usccb.org/bible/ezekiel/33?11 ">Ezekiel 33:11 </a></h4>
<div class="poetry"><p>R.<strong> Alleluia, alleluia.</strong><br>I take no pleasure in the death of the wicked man, says the Lord,<br>but rather in his conversion that he may live.<br>R.<strong> Alleluia, alleluia.</strong></p></div>`

	lent = `<h4>Verse Before the Gospel&nbsp;<a href="https://bible.usccb.org/bible/john/8?12 ">John 8:12</a></h4>
<div class="poetry"><p>I am the light of the world, says the Lord;<br>whoever follows me will have the light of life.<br>&nbsp;</p></div>`

	note = `<h4>     </h4>
<div class="poetry"><p><em>The passion narratives are proclaimed in full.</em><br>&nbsp;</p></div>`

	holyWeek = `<h4>Verse Before the Gospel </h4>
<div class="poetry"><p><span>Hail to you, our King;</span><br><span>you alone are compassionate with our errors.</span></p><p><span><strong>Or</strong></span><br><br><span>Hail to you, our King, obedient to the Father;</span></p></div>`

	blankLink = `<h4>Verse Before the Gospel <a href="https://bible.usccb.orgroute?&lt;nolink&gt; "> </a></h4>
<div class="poetry"><p>Hail to you, our King, obedient to the Father;<br>you were led to your crucifixion like a gentle lamb to the slaughter.<br>&nbsp;</p></div>`

	vigilPsalm = `<h4>Responsorial Psalm  <a href="https://bible.usccb.org/bible/psalms/118?1 ">Psalm 118:1-2, 16-17, 22-23</a></h4>
<div class="poetry"><p>R. <strong>Alleluia, alleluia, alleluia.</strong><br>Give thanks to the LORD, for he is good,<br>for his mercy endures forever.<br>R. <strong>Alleluia, alleluia, alleluia.</strong><br>The right hand of the LORD has struck with power;</p></div>`

	procession = `<h4>At the Procession with Palms - Gospel <a href="https://bible.usccb.org/bible/matthew/21?1 ">Matthew 21:1-11</a></h4>
<div class="poetry"><p>When Jesus and the disciples drew near Jerusalem,</p></div>`

	reading = `<h4>Reading 1 <a href="https://bible.usccb.org/bible/ephesians/4?7 ">Ephesians 4:7-16</a></h4>
<div class="poetry"><p>Brothers and sisters:<br>Grace was given to each of us</p></div>`

	gospel = `<h4>Gospel  <a href="https://bible.usccb.org/bible/luke/13?1 ">Luke 13:1-9</a></h4>
<div class="poetry"><p>Some people told Jesus about the Galileans</p></div>`
)

const want = "I take no pleasure in the death of the wicked man, says the Lord, but rather in his conversion that he may live. — Ezekiel 33:11"

// rss wraps each day's readings in an item, escaped the way the USCCB feed escapes them.
func rss(days ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>USCCB Daily Readings</title>`)
	for _, d := range days {
		b.WriteString("<item><title>day</title><description>" + html.EscapeString(d) + "</description></item>")
	}
	b.WriteString("</channel></rss>")
	return b.String()
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "Ordinary day", input: rss(reading + psalm + alleluia + gospel), want: want},
		{name: "Newest day wins", input: rss(reading+psalm+alleluia+gospel, lent+gospel), want: want},
		{name: "Lent", input: rss(psalm + lent + gospel), want: "I am the light of the world, says the Lord; whoever follows me will have the light of life. — John 8:12"},
		{name: "Note between verse and Gospel", input: rss(psalm + alleluia + note + gospel), want: want},
		{name: "Alternative is dropped", input: rss(psalm + holyWeek + gospel), want: "Hail to you, our King; you alone are compassionate with our errors."},
		{name: "Blank reference", input: rss(psalm + blankLink + gospel), want: "Hail to you, our King, obedient to the Father; you were led to your crucifixion like a gentle lamb to the slaughter."},
		{name: "Easter Vigil takes the last psalm", input: rss(psalm + reading + vigilPsalm + gospel), want: "Give thanks to the LORD, for he is good, for his mercy endures forever. — Psalm 118:1-2, 16-17, 22-23"},
		{name: "Procession Gospel is not the Gospel", input: rss(procession + reading + psalm + alleluia + gospel), want: want},
		{name: "Empty item is skipped", input: rss("", psalm+alleluia+gospel), want: want},
		{name: "Nothing before the Gospel", input: rss(reading + gospel), wantErr: ErrNoVerse},
		{name: "Empty acclamation", input: rss(`<h4>Alleluia</h4><div class="poetry"><p>&nbsp;</p></div>` + gospel), wantErr: ErrNoVerse},
		{name: "No Gospel", input: rss(psalm + alleluia), wantErr: ErrNoVerse},
		{name: "No items", input: rss(), wantErr: ErrNoVerse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tt.input))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Parse() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Parse() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("Read error", func(t *testing.T) {
		readErr := errors.New("broken pipe")
		if _, err := Parse(iotest.ErrReader(readErr)); !errors.Is(err, readErr) {
			t.Errorf("Parse() error = %v, want %v", err, readErr)
		}
	})
}

func TestFetch(t *testing.T) {
	serve := func(status int, body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.UserAgent() != userAgent {
				t.Errorf("User-Agent = %q, want %q", r.UserAgent(), userAgent)
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("Success", func(t *testing.T) {
		srv := serve(http.StatusOK, rss(psalm+alleluia+gospel))
		got, err := Fetch(context.Background(), srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		if got != want {
			t.Errorf("Fetch() = %q, want %q", got, want)
		}
	})

	t.Run("Server error", func(t *testing.T) {
		srv := serve(http.StatusForbidden, "")
		_, err := Fetch(context.Background(), srv.Client(), srv.URL)
		if err == nil || !strings.Contains(err.Error(), "403") {
			t.Errorf("Fetch() error = %v, want a 403", err)
		}
	})

	t.Run("Feed without verse", func(t *testing.T) {
		srv := serve(http.StatusOK, rss())
		if _, err := Fetch(context.Background(), srv.Client(), srv.URL); !errors.Is(err, ErrNoVerse) {
			t.Errorf("Fetch() error = %v, want %v", err, ErrNoVerse)
		}
	})

	t.Run("Unreachable", func(t *testing.T) {
		srv := serve(http.StatusOK, "")
		srv.Close()
		if _, err := Fetch(context.Background(), srv.Client(), srv.URL); err == nil {
			t.Error("Fetch() error = nil, want a connection error")
		}
	})

	t.Run("Bad URL", func(t *testing.T) {
		if _, err := Fetch(context.Background(), http.DefaultClient, "http://\x7f"); err == nil {
			t.Error("Fetch() error = nil, want a request error")
		}
	})
}

package waterfall

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// HARColumns are the columns of the spans ParseHAR returns: a request's
// status, its type, the bytes it took and how long it took.
var HARColumns = []string{"Status", "Type", "Size", "Time"}

// entry is a request of a HAR file.
type entry struct {
	Started time.Time `json:"startedDateTime"`
	Request struct {
		URL string
	}
	Response struct {
		Status  int
		Content struct {
			Size     int64
			MimeType string
		}
		Transfer int64 `json:"_transferSize"`
	}
	Timings struct {
		Blocked, DNS, Connect, SSL, Send, Wait, Receive float64
	}
}

// ParseHAR reads a HAR file, as browsers save a page's network activity,
// and returns a span for each request, in the order they started, with
// the phases browsers show: Blocked, DNS, Connect, TLS, Send, Wait and
// Receive, those it took.
func ParseHAR(r io.Reader) ([]*Span, error) {
	var har struct {
		Log struct {
			Entries []entry
		}
	}
	if err := json.NewDecoder(r).Decode(&har); err != nil {
		return nil, fmt.Errorf("waterfall: %w", err)
	}
	entries := har.Log.Entries
	slices.SortStableFunc(entries, func(a, b entry) int { return a.Started.Compare(b.Started) })
	spans := make([]*Span, len(entries))
	for i, e := range entries {
		tm := e.Timings
		// The connection's time takes in the TLS handshake's.
		connect := tm.Connect
		if tm.SSL > 0 {
			connect -= tm.SSL
		}
		var phases []Phase
		for _, p := range []struct {
			name string
			ms   float64
			hue  float32
		}{
			{"Blocked", tm.Blocked, -1}, {"DNS", tm.DNS, 185}, {"Connect", connect, 55}, {"TLS", tm.SSL, 300},
			{"Send", tm.Send, 100}, {"Wait", tm.Wait, 150}, {"Receive", tm.Receive, 250},
		} {
			if p.ms <= 0 {
				continue
			}
			color := ui.Oklch(0.68, 0.14, p.hue)
			if p.hue < 0 {
				color = ui.Oklch(0.7, 0, 0)
			}
			phases = append(phases, Phase{Name: p.name, Duration: time.Duration(p.ms * float64(time.Millisecond)), Color: color})
		}
		sp := &Span{Name: name(e.Request.URL), Start: e.Started.Sub(entries[0].Started), Phases: phases}
		size := e.Response.Transfer
		if size <= 0 {
			size = e.Response.Content.Size
		}
		status := strconv.Itoa(e.Response.Status)
		if e.Response.Status == 0 {
			status = "failed"
		}
		sp.Info = []string{status, kind(e.Response.Content.MimeType), bytes(size), long(sp.End() - sp.Start)}
		spans[i] = sp
	}
	return spans, nil
}

// name returns what a network panel calls a request to u: the last part
// of its path, or its host for the root.
func name(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return u
	}
	if base := path.Base(p.Path); base != "/" && base != "." {
		return base
	}
	return p.Host
}

// kind returns the type of a response of mime as a network panel names
// it: "document", "script", "image".
func kind(mime string) string {
	mime, _, _ = strings.Cut(mime, ";")
	switch {
	case mime == "text/html":
		return "document"
	case strings.Contains(mime, "javascript"):
		return "script"
	case mime == "text/css":
		return "stylesheet"
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "font/"):
		return "font"
	case strings.Contains(mime, "json"):
		return "fetch"
	case mime == "":
		return "other"
	}
	_, sub, _ := strings.Cut(mime, "/")
	return sub
}

// bytes formats n as a network panel does: "512 B", "14.2 kB".
func bytes(n int64) string {
	switch {
	case n <= 0:
		return "0 B"
	case n < 1000:
		return fmt.Sprintf("%d B", n)
	case n < 1000*1000:
		return fmt.Sprintf("%.1f kB", float64(n)/1000)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1000/1000)
}

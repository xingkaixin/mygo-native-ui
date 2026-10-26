// Package response shows the response to an HTTP request, as Postman and
// Insomnia show one under the request: its status in the color of its
// class, how long it took and how large its body is, and tabs for the
// body, the cookies it sets and its headers. A JSON body shows indented
// and highlighted, as it came, or as a tree to open and close; HTML, XML
// and other text shows highlighted, and a binary body by its size. The
// pointer on the time shows how it went: looking up the host,
// connecting, the TLS handshake, waiting for the first byte and
// downloading the rest.
//
//	var timing response.Timing
//	req = req.WithContext(timing.Trace(req.Context()))
//	res, err := http.DefaultClient.Do(req)
//	...
//	body, err := io.ReadAll(res.Body)
//	timing.Done()
//	...
//	response.View(c, &app.res, res, body, &timing).Grow(1)
package response

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"maps"
	"mime"
	"net/http"
	"net/http/httptrace"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xingkaixin/mygo-native-ui/ui/jsonview"
)

// Timing is how long the parts of a request took, which its Trace
// measures.
type Timing struct {
	// The trace's hooks run on the transport's goroutines, connecting to
	// both of a host's addresses at once.
	mu sync.Mutex

	start, dns, dnsDone, conn, connDone, tls, tlsDone, wrote, first, end time.Time
}

// Trace returns ctx with hooks that time a request made with it, from
// now. Make the request with it, and call Done once its body is read.
func (t *Timing) Trace(ctx context.Context) context.Context {
	*t = Timing{start: time.Now()}
	set := func(at *time.Time) {
		t.mu.Lock()
		if at.IsZero() {
			*at = time.Now()
		}
		t.mu.Unlock()
	}
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { set(&t.dns) },
		DNSDone:              func(httptrace.DNSDoneInfo) { set(&t.dnsDone) },
		ConnectStart:         func(string, string) { set(&t.conn) },
		ConnectDone:          func(string, string, error) { set(&t.connDone) },
		TLSHandshakeStart:    func() { set(&t.tls) },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { set(&t.tlsDone) },
		WroteRequest:         func(httptrace.WroteRequestInfo) { set(&t.wrote) },
		GotFirstResponseByte: func() { set(&t.first) },
	})
}

// Done marks the end of the request, once its body is read.
func (t *Timing) Done() {
	t.mu.Lock()
	t.end = time.Now()
	t.mu.Unlock()
}

// Total returns how long the request took, from Trace to Done.
func (t *Timing) Total() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.end.IsZero() {
		return 0
	}
	return t.end.Sub(t.start)
}

// phase is a part of a request: its name, and when it started and ended.
type phase struct {
	name       string
	start, end time.Time
}

// phases returns the parts of the request that happened, in order. A
// connection used again has no lookup, connecting or handshake.
func (t *Timing) phases() []phase {
	t.mu.Lock()
	defer t.mu.Unlock()
	sent := t.start
	for _, at := range []time.Time{t.dnsDone, t.connDone, t.tlsDone} {
		if at.After(sent) {
			sent = at
		}
	}
	all := []phase{
		{"DNS lookup", t.dns, t.dnsDone},
		{"Connecting", t.conn, t.connDone},
		{"TLS handshake", t.tls, t.tlsDone},
		{"Sending", sent, t.wrote},
		{"Waiting", t.wrote, t.first},
		{"Downloading", t.first, t.end},
	}
	return slices.DeleteFunc(all, func(p phase) bool { return p.start.IsZero() || p.end.IsZero() })
}

// State is what a view keeps: the tab and the way the body shows, and
// what it made of the body. Keep it in the app's state, one for each
// view.
type State struct {
	tab  int
	mode int
	tree jsonview.State

	body  []byte
	kind  kind
	doc   *jsonview.Document
	shown [2]*editor.State
}

// kind is what a body is.
type kind uint8

const (
	empty kind = iota
	binary
	text
	jsonBody
)

// The ways a JSON body shows.
const (
	pretty = iota
	raw
	tree
)

// load makes what the views show of body, typed as contentType, when it
// is not the body they show already.
func (s *State) load(body []byte, contentType string) {
	if len(body) == len(s.body) && (len(body) == 0 || &body[0] == &s.body[0]) {
		return
	}
	s.body, s.doc, s.shown, s.tree = body, nil, [2]*editor.State{}, jsonview.State{}
	media, _, _ := mime.ParseMediaType(contentType)
	switch {
	case len(body) == 0:
		s.kind = empty
	case !utf8.Valid(body):
		s.kind = binary
	case media == "application/json" || strings.HasSuffix(media, "+json") || (media == "" || media == "text/plain") && json.Valid(body):
		s.kind = jsonBody
		if doc, err := jsonview.Parse(body); err == nil {
			s.doc = doc
		}
		var out bytes.Buffer
		if json.Indent(&out, body, "", "  ") == nil {
			s.shown[pretty] = code(out.String(), "body.json", false)
		}
		s.shown[raw] = code(string(body), "", true)
	default:
		s.kind = text
		s.shown[pretty] = code(string(body), "body"+extension(media), true)
	}
	if s.shown[pretty] == nil {
		s.shown[pretty] = s.shown[raw]
	}
}

// code returns a read-only editor of text, highlighted as the file name
// says.
func code(text, name string, wrap bool) *editor.State {
	buf := editor.NewBuffer(text)
	buf.SetLanguage(editor.LanguageFor(name))
	return &editor.State{
		Buffer:          buf,
		ReadOnly:        true,
		SoftWrap:        wrap,
		NoActiveLine:    true,
		NoScrollPastEnd: true,
		Label:           "Body",
	}
}

// extension returns the file extension of a media type the editor
// highlights, or "".
func extension(media string) string {
	switch {
	case media == "text/html":
		return ".html"
	case media == "text/css":
		return ".css"
	case strings.HasSuffix(media, "javascript"):
		return ".js"
	case strings.HasSuffix(media, "xml"):
		return ".xml"
	case strings.HasSuffix(media, "yaml"):
		return ".yaml"
	case media == "text/markdown":
		return ".md"
	}
	return ""
}

// View creates the view of res, whose body is body, which t timed, or
// nil when nothing did.
func View(c *ui.Context, s *State, res *http.Response, body []byte, t *Timing) ui.Element {
	th := c.Theme()
	s.load(body, res.Header.Get("Content-Type"))
	cookies := res.Cookies()
	return ui.Column(c).Children(func() {
		ui.Row(c).Gap(th.Space(4)).AlignItems(ui.Center).Padding(0, th.Space(1)).Children(func() {
			ui.Tabs(c, &s.tab, "Body", count("Cookies", len(cookies)), count("Headers", headers(res.Header))).Grow(1)
			status(c, res)
			if t != nil {
				timing(c, t)
			}
			ui.Text(c, size(len(body))).FontSize(th.FontSize - 1).TextColor(th.TextMuted).FontFeatures("tnum")
		})
		ui.Divider(c)
		ui.Column(c).Grow(1).Children(func() {
			switch s.tab {
			case 0:
				s.bodyView(c)
			case 1:
				cookieTable(c, cookies)
			default:
				headerList(c, res.Header)
			}
		})
	})
}

// status shows the code and reason of res, in green for success, the
// accent for a redirect, amber for the client's error and red for the
// server's.
func status(c *ui.Context, res *http.Response) {
	t := c.Theme()
	color := t.Success
	switch {
	case res.StatusCode >= 500:
		color = t.Danger
	case res.StatusCode >= 400:
		color = t.Warning
	case res.StatusCode >= 300:
		color = t.Accent
	}
	text := res.Status
	if text == "" {
		text = strconv.Itoa(res.StatusCode) + " " + http.StatusText(res.StatusCode)
	}
	ui.Text(c, text).FontSize(t.FontSize-1).Bold().TextColor(color).FontFeatures("tnum").
		Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(1)).Background(color.Alpha(0.12))
}

// timing shows how long t took, and on the pointer, its phases.
func timing(c *ui.Context, t *Timing) {
	th := c.Theme()
	total := t.Total()
	el := ui.Text(c, duration(total)).FontSize(th.FontSize - 1).TextColor(th.TextMuted).FontFeatures("tnum").
		Focusable().Label("Time").Description(duration(total))
	phases := t.phases()
	if len(phases) == 0 || total <= 0 {
		return
	}
	ui.TooltipBase(c, el, func(tip ui.Element) {
		tip.Width(th.Space(80)).Padding(th.Space(3)).Gap(th.Space(1.5)).Radius(th.Space(2.5)).
			Background(th.Background).Border(1, th.Border).Shadow(0, 4, 16, 0, ui.RGB(0, 0, 0).Alpha(0.12))
		start := t.start
		tip.Children(func() {
			for _, p := range phases {
				ui.Row(c.Key(p.name)).Gap(th.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, p.name).FontSize(th.FontSize - 1).TextColor(th.TextMuted).Width(th.Space(24))
					from := float32(p.start.Sub(start)) / float32(total)
					to := float32(p.end.Sub(start)) / float32(total)
					ui.Box(c).Grow(1).Height(th.Space(2)).Draw(func(pt *ui.Painter, r ui.Rect) {
						pt.Fill(r, th.Text.Alpha(0.06), 2)
						x := r.X + r.W*min(from, 1)
						pt.Fill(ui.Rect{X: x, Y: r.Y, W: max(2, r.W*(min(to, 1)-min(from, 1))), H: r.H}, th.Accent, 2)
					})
					ui.Text(c, duration(p.end.Sub(p.start))).FontSize(th.FontSize - 1).FontFeatures("tnum").
						Width(th.Space(16)).TextAlign(ui.End)
				})
			}
			ui.Divider(c)
			ui.Row(c).Children(func() {
				ui.Text(c, "Total").Bold().Grow(1)
				ui.Text(c, duration(total)).Bold().FontFeatures("tnum")
			})
		})
	})
}

// bodyView shows the body as its kind shows.
func (s *State) bodyView(c *ui.Context) {
	t := c.Theme()
	switch s.kind {
	case empty:
		ui.Text(c, "No body").TextColor(t.TextMuted).Margin(t.Space(6)).AlignSelf(ui.Center)
		return
	case binary:
		ui.Text(c, "Binary body, "+size(len(s.body))).TextColor(t.TextMuted).Margin(t.Space(6)).AlignSelf(ui.Center)
		return
	}
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(1)).Children(func() {
		if s.kind == jsonBody {
			labels := []string{"Pretty", "Raw"}
			if s.doc != nil {
				labels = append(labels, "Tree")
			}
			ui.Segmented(c, &s.mode, labels...).Label("Show body as")
		}
		ui.Spacer(c)
		if ui.Button(c, "Copy").Clicked() {
			c.WriteClipboard(string(s.body))
			c.Toast("Copied the body")
		}
	})
	if s.kind == jsonBody && s.mode == tree && s.doc != nil {
		jsonview.View(c, &s.tree, s.doc).Grow(1)
		return
	}
	mode := pretty
	if s.kind == jsonBody && s.mode == raw {
		mode = raw
	}
	editor.View(c.Key(mode), s.shown[mode]).Grow(1)
}

// headerList shows the headers, by name, a row for each value.
func headerList(c *ui.Context, h http.Header) {
	t := c.Theme()
	if len(h) == 0 {
		ui.Text(c, "No headers").TextColor(t.TextMuted).Margin(t.Space(6)).AlignSelf(ui.Center)
		return
	}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Padding(t.Space(1)).Children(func() {
			for i, name := range slices.Sorted(maps.Keys(h)) {
				for j, v := range h[name] {
					r := ui.Row(c.Key([2]int{i, j})).Gap(t.Space(4)).Padding(t.Space(1.5), t.Space(2))
					if (i+j)%2 == 1 {
						r.Background(t.Text.Alpha(0.03))
					}
					r.Children(func() {
						ui.Text(c, name).TextColor(t.TextMuted).Width(t.Space(50)).Selectable()
						ui.Text(c, v).Font("monospace").Grow(1).Shrink(1).Selectable()
					})
				}
			}
		})
	})
}

// cookieTable shows the cookies, a row each.
func cookieTable(c *ui.Context, cookies []*http.Cookie) {
	t := c.Theme()
	if len(cookies) == 0 {
		ui.Text(c, "No cookies").TextColor(t.TextMuted).Margin(t.Space(6)).AlignSelf(ui.Center)
		return
	}
	columns := []ui.TableColumn{
		{Title: "Name", Width: t.Space(30)}, {Title: "Value"}, {Title: "Domain", Width: t.Space(30)},
		{Title: "Path", Width: t.Space(16)}, {Title: "Expires", Width: t.Space(40)},
		{Title: "HttpOnly", Width: t.Space(22), Align: ui.Center}, {Title: "Secure", Width: t.Space(20), Align: ui.Center},
	}
	ui.Table(c, nil, columns, len(cookies), func(row, col int) {
		k := cookies[row]
		var s string
		switch col {
		case 0:
			s = k.Name
		case 1:
			s = k.Value
		case 2:
			s = k.Domain
		case 3:
			s = k.Path
		case 4:
			switch {
			case k.MaxAge < 0:
				s = "Now"
			case k.MaxAge > 0:
				s = c.Now().Add(time.Duration(k.MaxAge) * time.Second).Format("Jan 2, 2006 15:04")
			case !k.Expires.IsZero():
				s = k.Expires.Local().Format("Jan 2, 2006 15:04")
			default:
				s = "Session"
			}
		case 5:
			s = check(k.HttpOnly)
		case 6:
			s = check(k.Secure)
		}
		ui.Text(c, s).SingleLine().Selectable()
	}).Grow(1)
}

func check(on bool) string {
	if on {
		return "✓"
	}
	return ""
}

// headers returns how many header lines h holds.
func headers(h http.Header) int {
	n := 0
	for _, vs := range h {
		n += len(vs)
	}
	return n
}

// count returns name with n after it, when there are any: "Headers 9".
func count(name string, n int) string {
	if n == 0 {
		return name
	}
	return name + " " + strconv.Itoa(n)
}

// size writes n bytes: "512 B", "4.2 KB", "1.3 MB".
func size(n int) string {
	switch {
	case n < 1000:
		return strconv.Itoa(n) + " B"
	case n < 1000*1000:
		return fmt.Sprintf("%.1f KB", float64(n)/1000)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1000/1000)
}

// duration writes d: "840 µs", "128 ms", "2.41 s".
func duration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return strconv.Itoa(int(d.Microseconds())) + " µs"
	case d < time.Second:
		return strconv.Itoa(int(d.Milliseconds())) + " ms"
	}
	return fmt.Sprintf("%.2f s", d.Seconds())
}

// Package request edits an HTTP request, as Postman and Insomnia edit one
// above its response: its method in its color, its URL and Send, and
// under them tabs for the query parameters, the headers and the body. The
// parameters and the URL's query are one: a query typed in the URL fills
// the table, and editing the table rewrites the query, leaving out the
// parameters turned off. Enter in the URL, or Cmd-Enter anywhere in the
// editor, sends.
//
//	if request.View(c, &app.edit, &app.req).Sent() {
//		req, err := app.req.HTTP(ctx)
//		...
//	}
//
// A JSON body that does not parse says so under it, and Format indents
// one that does.
package request

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/kvtable"
)

// Methods are the methods the method button offers.
var Methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// methodColors are the colors of the methods, as API clients color them.
var methodColors = map[string]ui.Color{
	"GET":    ui.Hex("#16a34a"),
	"POST":   ui.Hex("#d97706"),
	"PUT":    ui.Hex("#2563eb"),
	"PATCH":  ui.Hex("#9333ea"),
	"DELETE": ui.Hex("#dc2626"),
}

// Kind is what the body of a request is.
type Kind uint8

const (
	// None sends no body.
	None Kind = iota
	// JSON sends the body as application/json.
	JSON
	// Text sends the body as text/plain.
	Text
)

// Request is a request as the editor shows it. Params are the URL's query,
// which the editor keeps in step with URL; those off are kept but left out
// of it.
type Request struct {
	Method  string
	URL     string
	Params  []kvtable.Pair
	Headers []kvtable.Pair
	Kind    Kind
	Body    string
}

// HTTP returns the request to send, with the headers turned on and the
// body of its kind, whose Content-Type it sets unless a header does.
func (r *Request) HTTP(ctx context.Context) (*http.Request, error) {
	var body io.Reader
	if r.Kind != None {
		body = strings.NewReader(r.Body)
	}
	method := r.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, r.URL, body)
	if err != nil {
		return nil, err
	}
	for _, h := range r.Headers {
		if h.On && h.Key != "" {
			req.Header.Add(h.Key, h.Value)
		}
	}
	if req.Header.Get("Content-Type") == "" {
		switch r.Kind {
		case JSON:
			req.Header.Set("Content-Type", "application/json")
		case Text:
			req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		}
	}
	return req, nil
}

// State is what an editor keeps: the tab shown, and the URL whose query
// the parameters were last made from. Keep it in the app's state, one for
// each editor.
type State struct {
	tab int
	url string
}

// Events is what the user did in an editor in a frame.
type Events struct {
	sent bool
}

// Sent reports whether the user asked to send the request, with Send,
// Enter in the URL or Cmd-Enter.
func (e Events) Sent() bool { return e.sent }

// View creates the editor of *r. Give it a height, or grow it in its
// parent, for the tabs to fill.
func View(c *ui.Context, s *State, r *Request) Events {
	t := c.Theme()
	if r.Method == "" {
		r.Method = http.MethodGet
	}
	var ev Events
	ui.Column(c).Gap(t.Space(3)).Grow(1).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			method(c, r)
			field := ui.TextInput(c, &r.URL).Font("monospace").Placeholder("https://api.example.com/v1/users").
				Label("URL").Grow(1).Shrink(1).MinWidth(0)
			if field.Submitted() {
				ev.sent = true
			}
			if ui.PrimaryButton(c, "Send").Tooltip("Send (⌘↩)").Clicked() {
				ev.sent = true
			}
		})
		if r.URL != s.url {
			r.Params = parse(r.URL, r.Params)
			s.url = r.URL
		}
		labels := []string{count("Params", r.Params), count("Headers", r.Headers), "Body"}
		ui.Tabs(c, &s.tab, labels...)
		switch s.tab {
		case 0:
			if kvtable.Editor(c, &r.Params).Changed() {
				r.URL = withQuery(r.URL, r.Params)
				s.url = r.URL
			}
		case 1:
			kvtable.EditorWith(c, &r.Headers, kvtable.Options{Keys: headerNames})
		case 2:
			body(c, r)
		}
	})
	if c.Shortcut(ui.Cmd, ui.KeyEnter) {
		ev.sent = true
	}
	return ev
}

// method creates the button showing r's method in its color, which opens
// a menu of the others.
func method(c *ui.Context, r *Request) {
	t := c.Theme()
	col, ok := methodColors[r.Method]
	if !ok {
		col = t.TextMuted
	}
	b := ui.ButtonBase(c).Height(t.Space(8)).PaddingX(t.Space(3)).Radius(t.Radius).
		Background(t.Surface).Border(1, t.Border).Label("Method").Value(r.Method)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, r.Method).Font("monospace").Bold().TextColor(col).PassThrough()
			ui.Text(c, "▾").FontSize(t.FontSize - 3).TextColor(t.TextMuted).PassThrough()
		})
	})
	b.Menu(func(m *ui.Menu) {
		for _, name := range Methods {
			if m.Item(name).Checked(name == r.Method).Chosen() {
				r.Method = name
			}
		}
	})
}

// body creates the tab of r's body: its kind, and a field of it unless
// None.
func body(c *ui.Context, r *Request) {
	t := c.Theme()
	kind := int(r.Kind)
	ui.Column(c).Gap(t.Space(2)).Grow(1).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if ui.Segmented(c, &kind, "None", "JSON", "Text").Label("Body").Changed() {
				r.Kind = Kind(kind)
			}
			ui.Spacer(c)
			if r.Kind != JSON {
				return
			}
			var out bytes.Buffer
			err := json.Indent(&out, []byte(r.Body), "", "  ")
			if strings.TrimSpace(r.Body) != "" && err != nil {
				ui.Text(c, "Invalid JSON: "+jsonError(err, r.Body)).FontSize(t.FontSize - 1).TextColor(t.Danger).SingleLine().Shrink(1)
			}
			if ui.Button(c, "Format").Disabled(err != nil).Clicked() {
				r.Body = out.String()
			}
		})
		if r.Kind == None {
			ui.Text(c, "This request has no body.").TextColor(t.TextMuted).AlignSelf(ui.Center).MarginY(t.Space(8))
			return
		}
		ui.TextArea(c, &r.Body).Font("monospace").Label("Body").Grow(1).MinHeight(t.Space(30))
	})
}

// jsonError says where the JSON in s fails to parse, by line and column
// when err knows its offset.
func jsonError(err error, s string) string {
	se, ok := err.(*json.SyntaxError)
	if !ok {
		return err.Error()
	}
	before := s[:min(int(se.Offset), len(s))]
	line := strings.Count(before, "\n") + 1
	col := len(before) - strings.LastIndexByte(before, '\n')
	return se.Error() + " at " + strconv.Itoa(line) + ":" + strconv.Itoa(col)
}

// count labels a tab of pairs with how many are on.
func count(name string, pairs []kvtable.Pair) string {
	n := 0
	for _, p := range pairs {
		if p.On && p.Key != "" {
			n++
		}
	}
	if n == 0 {
		return name
	}
	return name + " " + strconv.Itoa(n)
}

// split returns the parts of a URL around its query: what comes before
// the "?", the query, and the fragment with its "#".
func split(raw string) (base, query, fragment string) {
	base = raw
	if i := strings.IndexByte(base, '#'); i >= 0 {
		base, fragment = base[:i], base[i:]
	}
	if i := strings.IndexByte(base, '?'); i >= 0 {
		base, query = base[:i], base[i+1:]
	}
	return base, query, fragment
}

// parse returns the parameters of raw's query, in order, followed by
// those of old that are off, which the query leaves out.
func parse(raw string, old []kvtable.Pair) []kvtable.Pair {
	_, query, _ := split(raw)
	var pairs []kvtable.Pair
	for part := range strings.SplitSeq(query, "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		pairs = append(pairs, kvtable.Pair{Key: unescape(k), Value: unescape(v), On: true})
	}
	for _, p := range old {
		if !p.On {
			pairs = append(pairs, p)
		}
	}
	return pairs
}

// unescape decodes a part of a query, leaving it as typed when it is not
// escaped well, as while it is being typed.
func unescape(s string) string {
	if u, err := url.QueryUnescape(s); err == nil {
		return u
	}
	return s
}

// withQuery returns raw with the parameters on as its query.
func withQuery(raw string, pairs []kvtable.Pair) string {
	base, _, fragment := split(raw)
	var q strings.Builder
	for _, p := range pairs {
		if !p.On || p.Key == "" {
			continue
		}
		if q.Len() > 0 {
			q.WriteByte('&')
		}
		q.WriteString(escape(p.Key))
		if p.Value != "" {
			q.WriteByte('=')
			q.WriteString(escape(p.Value))
		}
	}
	if q.Len() == 0 {
		return base + fragment
	}
	return base + "?" + q.String() + fragment
}

// escape encodes a part of a query, keeping the braces of template
// variables such as {{token}} as typed.
func escape(s string) string {
	e := url.QueryEscape(s)
	return strings.NewReplacer("%7B", "{", "%7D", "}").Replace(e)
}

// headerNames are suggested as header keys are typed.
var headerNames = []string{
	"Accept", "Accept-Encoding", "Accept-Language", "Authorization", "Cache-Control",
	"Content-Type", "Cookie", "If-Match", "If-None-Match", "Origin", "Referer",
	"User-Agent", "X-Api-Key", "X-Request-Id",
}

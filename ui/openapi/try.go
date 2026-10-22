package openapi

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/jsonview"
	"github.com/xingkaixin/mygo-native-ui/ui/kvtable"
	"github.com/xingkaixin/mygo-native-ui/ui/schemaform"
)

// Request is a request Try it built: its method, its URL, of the server
// chosen, the path with its parameters and the query, and its headers and
// body. A server's URL may be relative to where the document is, as /v1:
// resolve it from there.
type Request struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
	op     *operation
	n      int
}

// Response is what came back for a request: its status, how long it took,
// its headers and body, or why it failed.
type Response struct {
	Status int
	Time   time.Duration
	Header http.Header
	Body   []byte
	Err    error
}

// Respond shows resp as the response to req, which Sent reported, under
// Try it of req's operation: its status, time and size, and its body, as
// a tree when it is JSON, or its headers. A response to a request sent
// before the operation's last shows nowhere.
func (s *State) Respond(req Request, resp Response) {
	tr := s.trials[req.op]
	if tr == nil || tr.sent != req.n {
		return
	}
	tr.pending, tr.resp, tr.json, tr.tab = false, &resp, nil, 0
	if len(bytes.TrimSpace(resp.Body)) > 0 {
		tr.json, _ = jsonview.Parse(resp.Body)
	}
}

// trial is what Try it holds for an operation: the form of its parameters
// and its value, its headers and body, the number of the request last
// sent, whether it is waiting, and the response.
type trial struct {
	schema  *schemaform.Schema
	form    schemaform.State
	params  any
	headers []kvtable.Pair
	body    string
	sent    int
	pending bool
	resp    *Response
	json    *jsonview.Document
	tree    jsonview.State
	tab     int
}

// trial returns op's Try it, filling it in the first time.
func (s *State) trial(op *operation) *trial {
	if tr := s.trials[op]; tr != nil {
		return tr
	}
	tr := &trial{schema: form(op)}
	for _, pa := range op.params {
		if pa.in == "header" {
			v := ""
			if ex := example(pa.field); ex != nil {
				v = strings.Join(texts(plain(ex)), ",")
			}
			tr.headers = append(tr.headers, kvtable.Pair{Key: pa.name, Value: v, On: pa.required || v != ""})
		}
	}
	if b := op.body; b != nil {
		if b.media != "" {
			tr.headers = append(tr.headers, kvtable.Pair{Key: "Content-Type", Value: b.media, On: true})
		}
		text, isText := b.example.(string)
		switch {
		case isText && !strings.Contains(b.media, "json"):
			tr.body = text
		case b.example != nil:
			tr.body = indented(b.example)
		case b.schema != nil:
			tr.body = indented(sample(b.schema, nil))
		}
	}
	s.trials[op] = tr
	return tr
}

// example returns the example of f, or of its schema, or its default.
func example(f *field) any {
	for _, v := range []any{f.example, f.schema.example, f.schema.def} {
		if v != nil {
			return v
		}
	}
	return nil
}

// form returns the form of op's parameters of the path, query and
// cookies, a group of each, or nil without any.
func form(op *operation) *schemaform.Schema {
	root := &schemaform.Schema{Type: "object"}
	for _, in := range []string{"path", "query", "cookie"} {
		group := &schemaform.Schema{Type: "object", Title: places[in]}
		for _, pa := range op.params {
			if pa.in != in {
				continue
			}
			group.Properties = append(group.Properties, schemaform.Property{Name: pa.name, Schema: input(pa.field)})
			if pa.required {
				group.Required = append(group.Required, pa.name)
			}
		}
		if len(group.Properties) > 0 {
			root.Properties = append(root.Properties, schemaform.Property{Name: in, Schema: group})
		}
	}
	if len(root.Properties) == 0 {
		return nil
	}
	return root
}

// input returns the schema of f's field in a form: text, a number, a
// switch, a drop-down of the values it allows, or a list of them; text
// for an object, which few parameters are.
func input(f *field) *schemaform.Schema {
	scalar := func(s *schema) *schemaform.Schema {
		in := &schemaform.Schema{Type: s.kind()}
		for _, v := range s.enum {
			in.Enum = append(in.Enum, plain(v))
		}
		if !slices.Contains([]string{"string", "integer", "number", "boolean"}, in.Type) {
			in.Type = "string"
		}
		return in
	}
	in := scalar(f.schema)
	if f.schema.kind() == "array" && f.schema.items != nil {
		in = &schemaform.Schema{Type: "array", Items: scalar(f.schema.items)}
	}
	if v := example(f); v != nil {
		in.Default = plain(v)
		if _, isMap := in.Default.(map[string]any); isMap {
			in.Default = nil
		}
	}
	return in
}

// values returns the value of the parameter name in params as text,
// each item of a list.
func values(params any, in, name string) []string {
	m, _ := params.(map[string]any)
	group, _ := m[in].(map[string]any)
	return texts(group[name])
}

func texts(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case float64:
		return []string{strconv.FormatFloat(v, 'f', -1, 64)}
	case bool:
		return []string{strconv.FormatBool(v)}
	case []any:
		var out []string
		for _, x := range v {
			out = append(out, texts(x)...)
		}
		return out
	}
	return nil
}

// url returns the URL of op on the server chosen, with the parameters of
// its path and query in tr.
func (s *State) url(op *operation, tr *trial) string {
	path := op.path
	var query []string
	for _, pa := range op.params {
		vs := values(tr.params, pa.in, pa.name)
		switch pa.in {
		case "path":
			if len(vs) > 0 {
				path = strings.ReplaceAll(path, "{"+pa.name+"}", url.PathEscape(strings.Join(vs, ",")))
			}
		case "query":
			for _, v := range vs {
				query = append(query, url.QueryEscape(pa.name)+"="+url.QueryEscape(v))
			}
		}
	}
	u := strings.TrimSuffix(s.server, "/") + path
	if len(query) > 0 {
		u += "?" + strings.Join(query, "&")
	}
	return u
}

func (s *State) request(op *operation, tr *trial) Request {
	req := Request{Method: op.method, URL: s.url(op, tr), Header: http.Header{}, op: op, n: tr.sent}
	for _, h := range tr.headers {
		if h.On && h.Key != "" {
			req.Header.Add(h.Key, h.Value)
		}
	}
	var cookies []string
	for _, pa := range op.params {
		if pa.in == "cookie" {
			for _, v := range values(tr.params, pa.in, pa.name) {
				cookies = append(cookies, pa.name+"="+v)
			}
		}
	}
	if len(cookies) > 0 {
		req.Header.Add("Cookie", strings.Join(cookies, "; "))
	}
	if op.body != nil {
		req.Body = []byte(tr.body)
	}
	return req
}

// try shows Try it of op above its response.
func (s *State) try(c *ui.Context, op *operation) Events {
	t := c.Theme()
	tr := s.trial(op)
	var ev Events
	ui.SplitVerticalPercent(c, &s.split, func() {
		ui.Scroll(c.Key(op.key())).Fill().Children(func() {
			ui.Column(c).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
				heading(c, "Try it")
				if len(s.doc.servers) > 1 {
					ui.Field(c, "Server", func() { ui.Select(c, &s.server, s.doc.servers) })
				}
				if tr.schema != nil {
					schemaform.View(c, &tr.form, tr.schema, &tr.params)
				}
				group(c, "Headers", func() { kvtable.Editor(c, &tr.headers) })
				if op.body != nil {
					group(c, "Body", func() {
						ui.TextArea(c, &tr.body).Font("monospace").FontSize(t.FontSize-1).Lines(6, 18).Label("Body")
					})
				}
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(2.5)).Radius(t.Radius).
					Background(t.Surface).Children(func() {
					badge(c, op.method, t.FontSize-3)
					ui.Text(c, s.url(op, tr)).Font("monospace").FontSize(t.FontSize - 1).Selectable().Shrink(1)
				})
				ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
					if ui.PrimaryButton(c, "Send").Clicked() {
						if tr.schema != nil && len(schemaform.Validate(tr.schema, tr.params)) > 0 {
							tr.form.ShowErrors()
						} else {
							s.sent++
							tr.sent, tr.pending = s.sent, true
							ev = Events{req: s.request(op, tr), sent: true}
						}
					}
					if tr.pending {
						ui.Spinner(c)
					}
				})
			})
		})
	}, func() {
		s.result(c, tr)
	}).Fill()
	return ev
}

// group shows what fn builds under title.
func group(c *ui.Context, title string, fn func()) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted)
		fn()
	})
}

// result shows the response of tr: a line of its status, time and size,
// and its body, or headers.
func (s *State) result(c *ui.Context, tr *trial) {
	t := c.Theme()
	r := tr.resp
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(5)).MinHeight(t.Space(11)).Children(func() {
			heading(c, "Response")
			if r != nil && r.Err == nil {
				color := t.TextMuted
				switch r.Status / 100 {
				case 2:
					color = t.Success
				case 4:
					color = t.Warning
				case 5:
					color = t.Danger
				}
				ui.Text(c, strings.TrimSpace(fmt.Sprintf("%d %s", r.Status, http.StatusText(r.Status)))).Bold().TextColor(color).SingleLine()
				ui.Text(c, duration(r.Time)+"  ·  "+size(len(r.Body))).FontFeatures("tnum").TextColor(t.TextMuted).SingleLine()
			}
			ui.Box(c).Grow(1)
			if r != nil && r.Err == nil {
				ui.Segmented(c, &tr.tab, "Body", "Headers").Label("Show")
			}
		})
		ui.Divider(c)
		pad := ui.Column(c).Grow(1).Padding(t.Space(4), t.Space(5))
		switch {
		case r == nil && tr.pending:
			pad.Children(func() { ui.Text(c, "Waiting for the response…").TextColor(t.TextMuted) })
		case r == nil:
			pad.Children(func() { ui.Text(c, "Send the request to see its response").TextColor(t.TextMuted) })
		case r.Err != nil:
			pad.Children(func() { ui.Text(c, r.Err.Error()).TextColor(t.Danger).Selectable() })
		case tr.tab == 1:
			keys := make([]string, 0, len(r.Header))
			for k := range r.Header {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			pad.Padding(0).Children(func() {
				ui.Scroll(c).Grow(1).Children(func() {
					ui.Column(c).Gap(t.Space(1.5)).Padding(t.Space(4), t.Space(5)).Children(func() {
						for _, k := range keys {
							ui.Row(c.Key(k)).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
								ui.Text(c, k).TextColor(t.TextMuted).Width(t.Space(40)).Shrink(0)
								ui.Text(c, strings.Join(r.Header[k], ", ")).Font("monospace").FontSize(t.FontSize - 1).Selectable().Shrink(1)
							})
						}
					})
				})
			})
		case len(r.Body) == 0:
			pad.Children(func() { ui.Text(c, "No body").TextColor(t.TextMuted) })
		case tr.json != nil:
			pad.Padding(t.Space(1), 0, 0).Children(func() { jsonview.View(c, &tr.tree, tr.json).Grow(1) })
		default:
			pad.Padding(0).Children(func() {
				ui.Scroll(c).Grow(1).Children(func() {
					ui.Text(c, string(r.Body)).Font("monospace").FontSize(t.FontSize-1).Selectable().Padding(t.Space(4), t.Space(5))
				})
			})
		}
	})
}

// duration formats how long a request took: "0.42 ms", "38 ms", "1.20 s".
func duration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	switch {
	case ms >= 1000:
		return fmt.Sprintf("%.2f s", ms/1000)
	case ms >= 10:
		return fmt.Sprintf("%.0f ms", ms)
	}
	return fmt.Sprintf("%.2f ms", ms)
}

// size formats a count of bytes: "512 B", "3.4 kB".
func size(n int) string {
	if n < 1000 {
		return strconv.Itoa(n) + " B"
	}
	return fmt.Sprintf("%.1f kB", float64(n)/1000)
}

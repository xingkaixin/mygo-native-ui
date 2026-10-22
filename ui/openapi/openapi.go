// Package openapi shows an OpenAPI document as API reference sites do, as
// Redoc and Scalar show one: a sidebar of the API's operations by tag, each
// with its method in color, to search; the reference of the operation
// chosen, its parameters by where they go, its request body and its
// responses, their schemas as trees of properties to unfold; and beside
// it, Try it, fields for its parameters and its body, filled in from its
// examples, which sends the request through the app.
//
//	doc, err := openapi.Parse(data) // once, not every frame
//	...
//	if req, ok := openapi.View(c, &app.api, doc).Sent(); ok {
//		go func() {
//			resp := app.do(req) // with net/http, say
//			app.win.Update(func() { app.api.Respond(req, resp) })
//		}()
//	}
//
// It reads documents in JSON, of OpenAPI 3.0 and 3.1. YAML ones need
// converting to JSON first.
package openapi

import (
	"cmp"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// State is what a view keeps: the operation chosen, the search, the
// sections and trees open, the server chosen, what each operation's Try it
// holds, and the sizes of the panes. Keep it in the app's state, one for
// each view.
type State struct {
	doc *Document
	// chosen is the ID of the sidebar's item chosen: its tag and its
	// operation's key.
	chosen   string
	query    string
	sections map[string]*bool
	// open holds the trees of the reference opened or closed, by where
	// they are, "GET /notes 200/author".
	open   map[string]bool
	server string
	trials map[*operation]*trial
	sent   int
	// side is the sidebar's width, docs the reference's share of the
	// rest and split Try it's share of its pane, above the response.
	side, docs, split float32
}

// Events are what the user did with a view in a frame.
type Events struct {
	req  Request
	sent bool
}

// Sent returns the request the user sent with Try it. Make it, and give
// its response to Respond once it comes, in a later frame; Try it shows it
// is waiting until then.
func (e Events) Sent() (Request, bool) { return e.req, e.sent }

// View creates the browser of doc. It grows to take the room left in its
// parent.
//
// The sidebar shows the API's title and version, a search field, and the
// operations, those without a tag first and then by tag, as the document
// orders them, each with its method and its summary, or its path without
// one, those deprecated struck through. Typing in the search field shows
// those whose method, path, summary or operationId hold every word typed.
//
// The reference shows the operation chosen: its summary, method and path,
// its description as Markdown, its parameters grouped by where they go,
// its request body and its responses by status. A schema shows as a tree
// of its properties, each with its type, format, whether it is required,
// the values it allows, its description and example; a click opens an
// object's properties, or an array's items'. A schema that holds itself
// shows as circular where it comes back. The responses of success show
// open, the others closed.
//
// Try it holds fields for the parameters of the path, query and cookies,
// filled in from their examples or defaults, the headers, and the body,
// filled in from the request body's example or one made from its schema.
// Send checks the fields, showing what is wrong, and otherwise reports
// the request in Events.
func View(c *ui.Context, s *State, doc *Document) Events {
	t := c.Theme()
	if s.doc != doc {
		*s = State{doc: doc, sections: map[string]*bool{}, open: map[string]bool{}, server: doc.servers[0],
			trials: map[*operation]*trial{}, side: s.side, docs: s.docs, split: s.split}
	}
	if s.side == 0 {
		s.side = t.Space(70)
	}
	if s.docs == 0 {
		s.docs = 56
	}
	if s.split == 0 {
		s.split = 58
	}
	op := s.operation()
	var ev Events
	ui.Split(c, &s.side, func() {
		s.sidebar(c)
	}, func() {
		if op == nil {
			ui.Text(c, "The document has no operations").TextColor(t.TextMuted).Padding(t.Space(8))
			return
		}
		ui.SplitPercent(c, &s.docs, func() {
			s.reference(c.Key(op.key()), op)
		}, func() {
			ev = s.try(c, op)
		}).Fill()
	}).Grow(1)
	return ev
}

// id returns the ID of op's item in the sidebar, under tag.
func id(tag string, op *operation) string { return tag + "\n" + op.key() }

// operation returns the operation chosen, choosing the first when none
// is, or nil for a document without any.
func (s *State) operation() *operation {
	var first *operation
	firstID := ""
	for tag, op := range s.doc.items {
		if id(tag, op) == s.chosen {
			return op
		}
		if first == nil {
			first, firstID = op, id(tag, op)
		}
	}
	s.chosen = firstID
	return first
}

// items yields the operations as the sidebar lists them, with the tag of
// each.
func (d *Document) items(yield func(tag string, op *operation) bool) {
	for _, op := range d.untagged {
		if !yield("", op) {
			return
		}
	}
	for _, t := range d.tags {
		for _, op := range t.ops {
			if !yield(t.name, op) {
				return
			}
		}
	}
}

func (s *State) sidebar(c *ui.Context) {
	t := c.Theme()
	doc := s.doc
	words := strings.Fields(strings.ToLower(s.query))
	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(4), t.Space(4), t.Space(3)).Children(func() {
			ui.Text(c, cmp.Or(doc.title, "API")).Bold().FontSize(t.FontSize + 2).Shrink(1)
			if doc.version != "" {
				ui.Badge(c, doc.version)
			}
		})
		ui.SearchField(c, &s.query).Label("Search operations").Margin(0, t.Space(3), t.Space(1))
		ui.Sidebar(c, &s.chosen, func() {
			shown := false
			for _, op := range doc.untagged {
				if match(op, words) {
					s.item(c, "", op)
					shown = true
				}
			}
			for _, tg := range doc.tags {
				ops := slices.DeleteFunc(slices.Clone(tg.ops), func(op *operation) bool { return !match(op, words) })
				if len(ops) == 0 {
					continue
				}
				shown = true
				open := s.sections[tg.name]
				if open == nil {
					open = new(bool)
					*open = true
					s.sections[tg.name] = open
				}
				if len(words) > 0 {
					open = nil
				}
				ui.SidebarSection(c.Key(tg.name), tg.name, open, func() {
					for _, op := range ops {
						s.item(c, tg.name, op)
					}
				})
			}
			if !shown {
				ui.Text(c, "No operations match").TextColor(t.TextMuted).Padding(t.Space(2))
			}
		}).Label("Operations").Grow(1)
	})
}

// match reports whether op's method, path, summary or operationId hold
// each of words, in lower case.
func match(op *operation, words []string) bool {
	text := strings.ToLower(op.method + " " + op.path + " " + op.summary + " " + op.id)
	for _, w := range words {
		if !strings.Contains(text, w) {
			return false
		}
	}
	return true
}

func (s *State) item(c *ui.Context, tag string, op *operation) {
	t := c.Theme()
	it := ui.SidebarItem(c, id(tag, op), nil, cmp.Or(op.summary, op.path)).Tooltip(op.key())
	if op.deprecated {
		it.Strikethrough()
	}
	it.Children(func() { badge(c, op.method, t.FontSize-3) })
}

var methodColors = map[string]ui.Color{
	"GET":    ui.Hex("#2563eb"),
	"POST":   ui.Hex("#15803d"),
	"PUT":    ui.Hex("#c2410c"),
	"PATCH":  ui.Hex("#7c3aed"),
	"DELETE": ui.Hex("#dc2626"),
}

// badge shows method in white letters size DIPs high on a pill of its
// color, dark enough for them in either theme. It draws them, so that the
// strikethrough of a deprecated operation's item leaves them be.
func badge(c *ui.Context, method string, size float32) ui.Element {
	fill, ok := methodColors[method]
	if !ok {
		fill = ui.Hex("#6b7280")
	}
	label := ui.Span{Text: method, Size: size, Weight: 700, Color: ui.Hex("#ffffff")}
	return ui.Box(c).Size(size*4.8, size*1.8).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, fill, r.H/4)
		w, h := p.MeasureText(0, label)
		p.RichText(r.X+(r.W-w)/2, r.Y+(r.H-h)/2, 0, label)
	})
}

// places names where parameters go.
var places = map[string]string{"path": "Path parameters", "query": "Query parameters",
	"header": "Header parameters", "cookie": "Cookie parameters"}

// reference shows op: its summary, method and path, description,
// parameters, request body and responses.
func (s *State) reference(c *ui.Context, op *operation) {
	t := c.Theme()
	key := op.key()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Padding(t.Space(6), t.Space(8), t.Space(12)).Gap(t.Space(8)).Children(func() {
			ui.Column(c).Gap(t.Space(3)).Children(func() {
				ui.Text(c, cmp.Or(op.summary, op.path)).Bold().FontSize(t.FontSize + 8).Selectable()
				ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
					badge(c, op.method, t.FontSize-2)
					ui.Text(c, op.path).Font("monospace").Selectable().Shrink(1)
					if op.deprecated {
						ui.Text(c, "Deprecated").FontSize(t.FontSize-1).Bold().TextColor(t.Warning).
							Padding(0, t.Space(1.5)).Radius(t.Radius).Border(1, t.Warning)
					}
				})
				if op.description != "" {
					markdown.View(c, op.description, nil)
				}
			})
			if len(op.params) > 0 {
				ui.Column(c).Gap(t.Space(4)).Children(func() {
					heading(c, "Parameters")
					for _, in := range []string{"path", "query", "header", "cookie"} {
						var fs []*field
						for _, pa := range op.params {
							if pa.in == in {
								fs = append(fs, pa.field)
							}
						}
						if len(fs) == 0 {
							continue
						}
						ui.Column(c.Key(in)).Gap(t.Space(1)).Children(func() {
							ui.Text(c, places[in]).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted)
							ui.Divider(c)
							s.fields(c, key+" "+in, fs, nil)
						})
					}
				})
			}
			if b := op.body; b != nil {
				ui.Column(c).Gap(t.Space(3)).Children(func() {
					ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
						heading(c, "Request body")
						if b.required {
							ui.Text(c, "required").FontSize(t.FontSize - 1).TextColor(t.Danger)
						}
					})
					if b.description != "" {
						ui.Text(c, b.description).Selectable()
					}
					if b.schema != nil {
						s.tree(c, key+" body", b.media, b.schema)
					}
				})
			}
			if len(op.responses) > 0 {
				ui.Column(c).Gap(t.Space(4)).Children(func() {
					heading(c, "Responses")
					ui.Column(c).Border(1, t.Border).Radius(t.Radius).Clip().Children(func() {
						for i, r := range op.responses {
							if i > 0 {
								ui.Divider(c)
							}
							s.response(c.Key(r.status), key+" "+r.status, r)
						}
					})
				})
			}
		})
	})
}

func heading(c *ui.Context, text string) ui.Element {
	t := c.Theme()
	return ui.Text(c, text).Bold().FontSize(t.FontSize + 3)
}

// response shows r, a row of its status and description, and while open,
// below it, the tree of its schema.
func (s *State) response(c *ui.Context, key string, r *response) {
	t := c.Theme()
	open, set := s.open[key]
	if !set {
		open = strings.HasPrefix(r.status, "2")
	}
	branch := r.schema != nil
	open = open && branch
	ui.Column(c).Children(func() {
		var head ui.Element
		if branch {
			head = ui.ButtonBase(c).Justify(ui.Start).Label(r.status + " " + r.description).Expanded(open)
			if head.Hovered() {
				head.Background(t.SurfaceHover)
			}
			if head.Clicked() {
				open = !open
				s.open[key] = open
			}
		} else {
			head = ui.Row(c)
		}
		color := t.TextMuted
		switch {
		case strings.HasPrefix(r.status, "2"):
			color = t.Success
		case strings.HasPrefix(r.status, "4"):
			color = t.Warning
		case strings.HasPrefix(r.status, "5"):
			color = t.Danger
		}
		head.Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(2.5), t.Space(3)).Children(func() {
			arrow(c, open, branch)
			ui.Text(c, r.status).Font("monospace").Bold().TextColor(color).Shrink(0)
			ui.Text(c, r.description).Grow(1).Shrink(1)
			if r.media != "" {
				ui.Text(c, r.media).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine().Shrink(1)
			}
		})
		if open {
			ui.Column(c).Padding(t.Space(1), t.Space(4), t.Space(3), t.Space(9)).Children(func() {
				s.tree(c, key, "", r.schema)
			})
		}
	})
}

// tree shows sc, a body's schema in media: its type, and below, its
// fields.
func (s *State) tree(c *ui.Context, key, media string, sc *schema) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Wrap().Children(func() {
			if media != "" {
				ui.Text(c, media).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
			typeName(c, sc)
		})
		if d := sc.description; d != "" {
			ui.Text(c, d).TextColor(t.TextMuted).Selectable()
		}
		if fs := sc.fields(); len(fs) > 0 {
			ui.Divider(c).Margin(t.Space(1), 0, 0)
			s.fields(c, key, fs, []*schema{sc.inner()})
		}
	})
}

func typeName(c *ui.Context, sc *schema) {
	t := c.Theme()
	color := t.TextMuted
	if sc.missing != "" {
		color = t.Danger
	}
	ui.Text(c, sc.typeName()).Font("monospace").FontSize(t.FontSize - 1).TextColor(color)
	if f := sc.inner().format; f != "" {
		ui.Text(c, f).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted.Alpha(0.75))
	}
}

// fields shows fs, the fields of a schema under those of seen, a row
// each, leaving room for arrows when one of them opens.
func (s *State) fields(c *ui.Context, key string, fs []*field, seen []*schema) {
	arrows := slices.ContainsFunc(fs, func(f *field) bool {
		return len(f.schema.fields()) > 0 && !slices.Contains(seen, f.schema.inner())
	})
	ui.Column(c).Children(func() {
		for i, f := range fs {
			if i > 0 {
				ui.Divider(c)
			}
			s.field(c.Key(i), key+"/"+f.name, f, seen, arrows)
		}
	})
}

// field shows f: its name, type and format, whether it is required, its
// description, the values it allows and its example, and while open, the
// fields under it.
func (s *State) field(c *ui.Context, key string, f *field, seen []*schema, arrows bool) {
	t := c.Theme()
	in := f.schema.inner()
	circular := slices.Contains(seen, in)
	kids := f.schema.fields()
	if circular {
		kids = nil
	}
	branch := len(kids) > 0
	open := branch && s.open[key]
	names := lightName
	if t.Dark {
		names = darkName
	}
	ui.Column(c).Padding(t.Space(2), 0).Gap(t.Space(1)).Children(func() {
		var head ui.Element
		if branch {
			head = ui.ButtonBase(c).Justify(ui.Start).Label(f.name).Expanded(open)
			if head.Clicked() {
				open = !open
				s.open[key] = open
			}
		} else {
			head = ui.Row(c)
		}
		head.Gap(t.Space(1.5)).AlignItems(ui.Center).Wrap().Children(func() {
			if arrows {
				arrow(c, open, branch)
			}
			name := ui.Text(c, f.name).Font("monospace").FontWeight(600).TextColor(names)
			if f.deprecated {
				name.Strikethrough()
			}
			typeName(c, f.schema)
			if f.required {
				ui.Text(c, "required").FontSize(t.FontSize - 2).TextColor(t.Danger)
			}
			if f.deprecated {
				ui.Text(c, "deprecated").FontSize(t.FontSize - 2).TextColor(t.Warning)
			}
			if circular {
				ui.Text(c, "circular").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			}
		})
		description, example := cmp.Or(f.description, f.schema.description), f.example
		if example == nil {
			example = f.schema.example
		}
		if description == "" && len(in.enum) == 0 && example == nil {
			return
		}
		indent := float32(0)
		if arrows {
			indent = t.Space(5.5)
		}
		ui.Column(c).Gap(t.Space(1)).PaddingStart(indent).Children(func() {
			if description != "" {
				ui.Text(c, description).Selectable()
			}
			if len(in.enum) > 0 {
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Wrap().Children(func() {
					ui.Text(c, "Allowed:").FontSize(t.FontSize - 1).TextColor(t.TextMuted).MarginEnd(t.Space(0.5))
					for i, v := range in.enum {
						ui.Text(c.Key(i), encode(v)).Font("monospace").FontSize(t.FontSize-2).Selectable().
							Padding(t.Space(0.25), t.Space(1.5)).Radius(t.Radius).Background(t.Text.Alpha(0.07))
					}
				})
			}
			if example != nil {
				ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Start).Children(func() {
					ui.Text(c, "Example:").FontSize(t.FontSize - 1).TextColor(t.TextMuted).Shrink(0)
					ui.Text(c, encode(example)).Font("monospace").FontSize(t.FontSize - 1).Selectable().Shrink(1)
				})
			}
		})
		if open {
			ui.Column(c).MarginStart(t.Space(2)).PaddingStart(t.Space(3.5)).BorderStartWidth(1).BorderColor(t.Border).Children(func() {
				s.fields(c, key, kids, append(seen[:len(seen):len(seen)], in))
			})
		}
	})
}

// The names of fields are in the colors Visual Studio Code gives JSON's,
// as jsonview's.
var (
	lightName = ui.Hex("#0451a5")
	darkName  = ui.Hex("#9cdcfe")
)

// arrow shows the arrow of a row that opens, pointing down while open, or
// the room of one.
func arrow(c *ui.Context, open, shown bool) {
	t := c.Theme()
	color := t.TextMuted
	ui.Box(c).Size(t.Space(4), t.Space(4)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		if !shown {
			return
		}
		var path ui.Path
		if open {
			path.MoveTo(r.X+r.W/4, r.Y+r.H*3/8).LineTo(r.X+r.W/2, r.Y+r.H*5/8).LineTo(r.X+r.W*3/4, r.Y+r.H*3/8)
		} else {
			path.MoveTo(r.X+r.W*3/8, r.Y+r.H/4).LineTo(r.X+r.W*5/8, r.Y+r.H/2).LineTo(r.X+r.W*3/8, r.Y+r.H*3/4)
		}
		p.StrokePath(&path, 1.5, color)
	})
}

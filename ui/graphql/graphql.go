// Package graphql shows a GraphQL schema as GraphiQL's documentation
// explorer does: the root types and every type by kind, then a type's
// description, its fields with their arguments and types, an enum's
// values, a union's members and an interface's implementations, then a
// field's arguments with their defaults. Names of types are links, and
// Back returns along the way taken. Deprecated fields show struck through
// with the reason. Typing in the search finds types and fields by name.
//
//	schema, err := graphql.Parse(sdl) // once, not every frame
//	graphql.View(c, &app.docs, schema).Grow(1)
package graphql

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// State is what an explorer keeps: the pages visited to the one shown,
// and the search. Keep it in the app's state, one for each explorer.
type State struct {
	stack []place
	query string
}

// place is a page: a type, or a field of one.
type place struct {
	typ, field string
}

func (pl place) title() string {
	if pl.field != "" {
		return pl.typ + "." + pl.field
	}
	return pl.typ
}

// colors are GraphiQL's: fields blue, arguments purple, types amber.
type colors struct {
	field, arg, typ ui.Color
}

func palette(t *ui.Theme) colors {
	if t.Dark {
		return colors{ui.Oklch(0.75, 0.12, 250), ui.Oklch(0.75, 0.13, 320), ui.Oklch(0.8, 0.13, 75)}
	}
	return colors{ui.Oklch(0.5, 0.17, 255), ui.Oklch(0.5, 0.19, 320), ui.Oklch(0.58, 0.14, 60)}
}

// View creates the explorer of schema.
func View(c *ui.Context, s *State, schema *Schema) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(3)).Role(ui.RoleGroup).Label("Documentation explorer")
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if len(s.stack) > 0 {
				back := "Docs"
				if len(s.stack) > 1 {
					back = s.stack[len(s.stack)-2].title()
				}
				if ui.Button(c, "‹ "+back).Label("Back to " + back).Clicked() {
					s.stack = s.stack[:len(s.stack)-1]
				}
			}
			ui.SearchField(c, &s.query).Grow(1)
		})
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c.Key(len(s.stack))).Gap(t.Space(4)).PaddingY(t.Space(2)).Children(func() {
				switch {
				case strings.TrimSpace(s.query) != "":
					s.results(c, schema)
				case len(s.stack) == 0:
					s.home(c, schema)
				default:
					pl := s.stack[len(s.stack)-1]
					typ := schema.Lookup(pl.typ)
					if typ == nil {
						break
					}
					if pl.field == "" {
						s.typePage(c, schema, typ)
						break
					}
					for _, f := range typ.Fields {
						if f.Name == pl.field {
							s.fieldPage(c, schema, typ, f)
						}
					}
				}
			})
		})
	})
	return root
}

func (s *State) open(pl place) {
	s.stack = append(s.stack, pl)
	s.query = ""
}

func heading(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.TextMuted)
}

func description(c *ui.Context, text string) {
	if text != "" {
		markdown.View(c, text, nil)
	}
}

func (s *State) home(c *ui.Context, schema *Schema) {
	t := c.Theme()
	col := palette(t)
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		heading(c, "Root types")
		for _, r := range []struct{ op, name string }{{"query", schema.Query}, {"mutation", schema.Mutation}, {"subscription", schema.Subscription}} {
			if r.name == "" {
				continue
			}
			ui.Row(c.Key(r.op)).Gap(t.Space(1)).Children(func() {
				ui.Text(c, r.op+":").TextColor(col.field).Font("monospace").MarginEnd(t.Space(1.5))
				s.typeLink(c, schema, r.name)
			})
		}
	})
	for _, k := range []struct{ kind, title string }{
		{"type", "Objects"}, {"interface", "Interfaces"}, {"union", "Unions"}, {"enum", "Enums"}, {"input", "Inputs"}, {"scalar", "Scalars"},
	} {
		var names []string
		for _, typ := range schema.Types {
			if typ.Kind == k.kind && typ.Name != schema.Query && typ.Name != schema.Mutation && typ.Name != schema.Subscription {
				names = append(names, typ.Name)
			}
		}
		if len(names) == 0 {
			continue
		}
		slices.Sort(names)
		ui.Column(c.Key(k.kind)).Gap(t.Space(1.5)).Children(func() {
			heading(c, k.title)
			for _, n := range names {
				ui.Row(c.Key(n)).Children(func() { s.typeLink(c, schema, n) })
			}
		})
	}
}

// typeLink creates typ, as written, a link to its named type if the schema
// has it, and reports whether it was clicked.
func (s *State) typeLink(c *ui.Context, schema *Schema, typ string) bool {
	t := c.Theme()
	e := ui.Text(c, typ).TextColor(palette(t).typ).Font("monospace")
	named := Named(typ)
	if schema.Lookup(named) == nil {
		return false
	}
	e.Cursor(ui.CursorPointer).Role(ui.RoleLink)
	if e.Hovered() {
		e.Underline()
	}
	if e.Clicked() {
		s.open(place{typ: named})
		return true
	}
	return false
}

func (s *State) typePage(c *ui.Context, schema *Schema, typ *Type) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, typ.Name).FontSize(t.FontSize + 6).FontWeight(600).Font("monospace")
			ui.Badge(c, typ.Kind)
		})
		description(c, typ.Description)
	})
	links := func(title string, names []string) {
		if len(names) == 0 {
			return
		}
		ui.Column(c.Key(title)).Gap(t.Space(1.5)).Children(func() {
			heading(c, title)
			for _, n := range names {
				ui.Row(c.Key(n)).Children(func() { s.typeLink(c, schema, n) })
			}
		})
	}
	links("Implements", typ.Interfaces)
	if len(typ.Fields) > 0 {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			heading(c, "Fields")
			for _, f := range typ.Fields {
				s.fieldRow(c.Key(f.Name), schema, typ, f)
			}
		})
	}
	if len(typ.Values) > 0 {
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			heading(c, "Values")
			for _, v := range typ.Values {
				ui.Column(c.Key(v.Name)).Gap(t.Space(0.5)).Children(func() {
					name := ui.Text(c, v.Name).Font("monospace").TextColor(palette(t).field)
					if v.Deprecated != "" {
						name.Strikethrough()
					}
					if v.Description != "" {
						ui.Text(c, v.Description).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
					}
					deprecated(c, v.Deprecated)
				})
			}
		})
	}
	links("Possible types", typ.Members)
	if typ.Kind == "interface" {
		var impl []string
		for _, o := range schema.Types {
			if slices.Contains(o.Interfaces, typ.Name) {
				impl = append(impl, o.Name)
			}
		}
		links("Implementations", impl)
	}
}

func deprecated(c *ui.Context, reason string) {
	if reason == "" {
		return
	}
	t := c.Theme()
	ui.Text(c, "Deprecated: "+reason).FontSize(t.FontSize - 1).TextColor(t.Warning)
}

// fieldRow creates the row of field f of typ: its signature, its
// description's first lines, and a link to its page.
func (s *State) fieldRow(c *ui.Context, schema *Schema, typ *Type, f Field) {
	t := c.Theme()
	col := palette(t)
	e := ui.Column(c).Gap(t.Space(0.5)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
		Cursor(ui.CursorPointer).Role(ui.RoleLink).Label(typ.Name + "." + f.Name)
	if e.Hovered() {
		e.Background(t.Text.Alpha(0.05))
	}
	linked := false
	e.Children(func() {
		ui.Row(c).Wrap().AlignItems(ui.Center).Children(func() {
			name := ui.Text(c, f.Name).Font("monospace").TextColor(col.field)
			if f.Deprecated != "" {
				name.Strikethrough()
			}
			if len(f.Args) > 0 {
				ui.Text(c, "(").Font("monospace").TextColor(t.TextMuted)
				if len(f.Args) <= 2 {
					for i, a := range f.Args {
						if i > 0 {
							ui.Text(c.Key(i), ",").Font("monospace").TextColor(t.TextMuted).MarginEnd(t.Space(1.5))
						}
						ui.Text(c.Key(a.Name), a.Name+":").Font("monospace").TextColor(col.arg).MarginEnd(t.Space(1.5))
						linked = s.typeLink(c.Key("arg"+a.Name), schema, a.Type) || linked
					}
				} else {
					ui.Text(c, "…").Font("monospace").TextColor(t.TextMuted)
				}
				ui.Text(c, ")").Font("monospace").TextColor(t.TextMuted)
			}
			ui.Text(c, ":").Font("monospace").TextColor(t.TextMuted).MarginEnd(t.Space(1.5))
			linked = s.typeLink(c, schema, f.Type) || linked
			if f.Default != "" {
				ui.Text(c, "= "+f.Default).Font("monospace").TextColor(t.TextMuted).MarginStart(t.Space(1.5))
			}
		})
		if f.Description != "" {
			ui.Text(c, plain(f.Description)).TextColor(t.TextMuted).FontSize(t.FontSize - 1).MaxLines(2)
		}
	})
	if e.Clicked() && !linked && typ.Kind != "input" {
		s.open(place{typ: typ.Name, field: f.Name})
	}
}

func (s *State) fieldPage(c *ui.Context, schema *Schema, typ *Type, f Field) {
	t := c.Theme()
	col := palette(t)
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, typ.Name+".").FontSize(t.FontSize + 6).Font("monospace").TextColor(t.TextMuted)
			ui.Text(c, f.Name).FontSize(t.FontSize + 6).FontWeight(600).Font("monospace").TextColor(col.field)
		})
		deprecated(c, f.Deprecated)
		description(c, f.Description)
	})
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		heading(c, "Type")
		ui.Row(c).Children(func() { s.typeLink(c, schema, f.Type) })
	})
	if len(f.Args) == 0 {
		return
	}
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		heading(c, "Arguments")
		for _, a := range f.Args {
			ui.Column(c.Key(a.Name)).Gap(t.Space(0.5)).Children(func() {
				ui.Row(c).Wrap().Children(func() {
					ui.Text(c, a.Name+":").Font("monospace").TextColor(col.arg).MarginEnd(t.Space(1.5))
					s.typeLink(c, schema, a.Type)
					if a.Default != "" {
						ui.Text(c, "= "+a.Default).Font("monospace").TextColor(t.TextMuted).MarginStart(t.Space(1.5))
					}
				})
				if a.Description != "" {
					ui.Text(c, plain(a.Description)).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
				}
			})
		}
	})
}

func (s *State) results(c *ui.Context, schema *Schema) {
	t := c.Theme()
	col := palette(t)
	q := strings.ToLower(strings.TrimSpace(s.query))
	found := 0
	row := func(key string, pl place, kind string) {
		found++
		e := ui.Row(c.Key(key)).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
			Cursor(ui.CursorPointer).Role(ui.RoleLink).Label(pl.title())
		if e.Hovered() {
			e.Background(t.Text.Alpha(0.05))
		}
		e.Children(func() {
			if pl.field != "" {
				ui.Row(c).Children(func() {
					ui.Text(c, pl.typ+".").Font("monospace").TextColor(t.TextMuted)
					ui.Text(c, pl.field).Font("monospace").TextColor(col.field)
				})
			} else {
				ui.Text(c, pl.typ).Font("monospace").TextColor(col.typ)
			}
			ui.Text(c, kind).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
		if e.Clicked() {
			s.open(pl)
		}
	}
	ui.Column(c).Children(func() {
		for _, typ := range schema.Types {
			if strings.Contains(strings.ToLower(typ.Name), q) {
				row("t"+typ.Name, place{typ: typ.Name}, typ.Kind)
			}
		}
		for _, typ := range schema.Types {
			if typ.Kind == "input" {
				continue
			}
			for _, f := range typ.Fields {
				if strings.Contains(strings.ToLower(f.Name), q) {
					row("f"+typ.Name+"."+f.Name, place{typ: typ.Name, field: f.Name}, f.Type)
				}
			}
		}
		if found == 0 {
			ui.Text(c, "Nothing matches").TextColor(t.TextMuted)
		}
	})
}

// plain returns the first paragraph of a Markdown description without
// its marks of emphasis and code, for a line under a name.
func plain(md string) string {
	md, _, _ = strings.Cut(md, "\n\n")
	return strings.NewReplacer("**", "", "__", "", "`", "", "\n", " ").Replace(md)
}

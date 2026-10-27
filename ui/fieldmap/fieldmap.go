// Package fieldmap maps the columns of a file being imported to the
// fields they fill, as the import of a CSV file into a CRM asks: the
// file's columns with a sample of their values on the left, the fields on
// the right, and a line between each field and the column it takes.
// Dragging from a column to a field joins them, and each field has a menu
// of the columns too. A required field without a column is marked, and a
// field shows the first value it would take.
//
//	if app.mapping == nil {
//		app.mapping = fieldmap.Match(columns, fields)
//	}
//	fieldmap.View(c, &app.mapping, columns, fields)
//
// (*mapping)[i] is the column field i takes, -1 for none. A column can
// fill several fields.
package fieldmap

import (
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// none is the menu's choice of no column.
const none = "—"

// Column is a column of the file: its heading and some of its values.
type Column struct {
	Name    string
	Samples []string
}

// Field is a field the columns fill.
type Field struct {
	Name     string
	Required bool
}

// Match returns a mapping of fields to the columns of the same name, case,
// spaces and punctuation aside, or one name holding the other, as "Email"
// and "E-mail address"; -1 where none matches.
func Match(columns []Column, fields []Field) []int {
	m := make([]int, len(fields))
	for i, f := range fields {
		m[i] = -1
		want := plain(f.Name)
		for j, col := range columns {
			if have := plain(col.Name); have == want {
				m[i] = j
				break
			} else if m[i] < 0 && have != "" && want != "" && (strings.Contains(have, want) || strings.Contains(want, have)) {
				m[i] = j
			}
		}
	}
	return m
}

func plain(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// state is what a view keeps from frame to frame: the column dragged
// from, -1 for none, and the pointer, in the view's axes.
type state struct {
	from   int
	px, py float32
}

// View creates the columns and fields, with the lines between them, of
// *mapping, which it makes as long as fields. Changed reports a change to
// it.
func View(c *ui.Context, mapping *[]int, columns []Column, fields []Field) ui.Element {
	t := c.Theme()
	for len(*mapping) < len(fields) {
		*mapping = append(*mapping, -1)
	}
	m := (*mapping)[:len(fields)]
	rowH, headH := t.Space(13), t.Space(8)
	left, gap := t.Space(64), t.Space(32)
	root := ui.Row(c).Height(headH + float32(max(len(columns), len(fields)))*rowH)
	st := ui.Local(root, "fieldmap.state", func() state { return state{from: -1} })
	hover, dropped := -1, false

	names := make([]string, len(columns)+1)
	names[0] = none
	for j, col := range columns {
		names[j+1] = col.Name
	}
	at := func(i int) float32 { return headH + float32(i)*rowH + rowH/2 }

	root.Children(func() {
		ui.Column(c).Width(left).Children(func() {
			heading(c, "Columns in the file")
			for j, col := range columns {
				row := ui.Row(c.Key(j)).Height(rowH).Padding(0, t.Space(3)).AlignItems(ui.Center).Gap(t.Space(2)).
					Radius(t.Radius).Cursor(ui.CursorGrab)
				if row.Hovered() || st.from == j {
					hover = j
					row.Background(t.Text.Alpha(0.05))
				}
				// The drag goes on past the row, its pointer read in the
				// row's axes.
				_, _, pressing := row.Dragged()
				switch {
				case pressing && st.from < 0:
					st.from = j
				case !pressing && st.from == j:
					dropped = true
				}
				if st.from == j {
					x, y, _ := row.PointerPosition()
					st.px, st.py = x, headH+float32(j)*rowH+y
				}
				row.Children(func() {
					ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(0.5)).Children(func() {
						ui.Text(c, col.Name).Bold().SingleLine().Ellipsis("…")
						ui.Text(c, strings.Join(slices.DeleteFunc(slices.Clone(col.Samples), func(v string) bool { return v == "" }), ", ")).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
					})
				})
			}
		})
		ui.Box(c).Width(gap)
		over := -1
		if st.from >= 0 && st.px > left+gap && st.py > headH {
			over = int((st.py - headH) / rowH)
		}
		ui.Column(c).Grow(1).Children(func() {
			heading(c, "Fields")
			for i, f := range fields {
				if field(c.Key(i), &m[i], f, columns, names, rowH, i == over || m[i] >= 0 && m[i] == hover) {
					root.SetChanged()
				}
			}
		})
	})

	if dropped {
		if st.px > left+gap && st.py > headH {
			if i := int((st.py - headH) / rowH); i < len(fields) && m[i] != st.from {
				m[i] = st.from
				root.SetChanged()
			}
		}
		st.from = -1
	}

	from := st.from
	dx, dy := st.px, st.py
	root.DrawOver(func(p *ui.Painter, r ui.Rect) {
		x0, x1 := r.X+left, r.X+left+gap
		curve := func(y0, x, y float32, col ui.Color, w float32) {
			path := (&ui.Path{}).MoveTo(x0, r.Y+y0).CubeTo(x0+gap/2, r.Y+y0, x-gap/2, r.Y+y, x, r.Y+y)
			p.StrokePath(path, w, col)
			p.Fill(ui.Rect{X: x0 - 3, Y: r.Y + y0 - 3, W: 6, H: 6}, col, 3)
			p.Fill(ui.Rect{X: x - 3, Y: r.Y + y - 3, W: 6, H: 6}, col, 3)
		}
		for i, j := range m {
			if j < 0 || j >= len(columns) {
				continue
			}
			col, w := t.Accent.Alpha(0.45), float32(1.5)
			if j == hover {
				col, w = t.Accent, 2
			}
			curve(at(j), x1, at(i), col, w)
		}
		if from >= 0 {
			curve(at(from), r.X+dx, dy, t.Accent, 2)
		}
	})
	return root
}

func heading(c *ui.Context, s string) {
	t := c.Theme()
	ui.Text(c, s).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted).Height(t.Space(8)).PaddingX(t.Space(3))
}

// field builds the row of field f, which takes column *taken, and
// reports whether its menu changed that.
func field(c *ui.Context, taken *int, f Field, columns []Column, names []string, rowH float32, lit bool) bool {
	t := c.Theme()
	row := ui.Row(c).Height(rowH).Padding(0, t.Space(3)).AlignItems(ui.Center).Gap(t.Space(3)).Radius(t.Radius)
	if lit {
		row.Background(t.Text.Alpha(0.05))
	}
	missing := f.Required && *taken < 0
	changed := false
	row.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(0.5)).Children(func() {
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, f.Name).Bold().SingleLine()
				if f.Required {
					color := t.TextMuted
					if missing {
						color = t.Danger
					}
					ui.Text(c, "Required").FontSize(t.FontSize - 2).TextColor(color)
				}
			})
			sample := "Not imported"
			if j := *taken; j >= 0 && j < len(columns) && len(columns[j].Samples) > 0 {
				sample = columns[j].Samples[0]
			}
			ui.Text(c, sample).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
		})
		chosen := none
		if *taken >= 0 && *taken < len(columns) {
			chosen = names[*taken+1]
		}
		if ui.Select(c, &chosen, names).Width(t.Space(44)).Label(f.Name).Changed() {
			*taken = -1
			for j, n := range names[1:] {
				if n == chosen {
					*taken = j
					break
				}
			}
			changed = true
		}
	})
	if missing {
		row.Border(1, t.Danger.Alpha(0.5))
	}
	return changed
}

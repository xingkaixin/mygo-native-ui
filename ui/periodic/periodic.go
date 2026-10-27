// Package periodic shows the periodic table of the elements: each a cell
// with its atomic number, symbol, and its name and weight where there is
// room, colored by its kind, the lanthanides and actinides in two rows
// below. Hovering a cell names the element in the gap at the top, and a
// click chooses it; hovering a kind in the legend dims the others, and a
// search dims the elements whose name, symbol or number do not match it.
//
//	ui.SearchField(c, &app.query)
//	periodic.View(c, &app.table, app.query).Grow(1)
//	if app.table.Chosen > 0 {
//		e := periodic.Elements[app.table.Chosen-1]
//	}
package periodic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// hues are the kinds' hues in Oklch.
var hues = []float32{25, 65, 345, 310, 240, 205, 170, 135, 100, 275}

func (k Category) color() ui.Color { return ui.Oklch(0.68, 0.15, hues[k]) }

// State is what a table keeps: the element chosen. Keep it in the app's
// state, one for each view.
type State struct {
	// Chosen is the atomic number of the element chosen, which a click
	// changes, 0 for none.
	Chosen int
	held   bool
}

// grid returns where element z sits, in cells from the top left, the
// rows of the lanthanides and actinides half a cell below the rest.
func grid(z int) (col, row float32) {
	p, g, f := place(z)
	if g == 0 {
		return float32(f + 2), float32(p) + 1.5
	}
	return float32(g - 1), float32(p - 1)
}

// matches reports whether e's name or symbol starts with query, or its
// number is query.
func matches(e Element, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if n, err := strconv.Atoi(q); err == nil {
		return e.Number == n
	}
	return strings.HasPrefix(strings.ToLower(e.Name), q) || strings.HasPrefix(strings.ToLower(e.Symbol), q)
}

// View creates the table, dimming the elements that do not match query.
// Changed reports a new choice. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, query string) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Role(ui.RoleGroup).Label("Periodic table")
	b := view.LayoutBounds()
	gap := float32(3)
	cw := (b.W - 17*gap) / 18
	ch := cw * 1.15
	if h := 9.5*ch + 8.5*gap; h > b.H {
		ch = (b.H - 8.5*gap) / 9.5
		cw = ch / 1.15
	}
	x0 := (b.W - (18*cw + 17*gap)) / 2
	cell := func(col, row float32) ui.Rect {
		return ui.Rect{X: x0 + col*(cw+gap), Y: row * (ch + gap), W: cw, H: ch}
	}
	// The legend fills the gap above the transition metals, two columns of
	// five kinds.
	legendArea := ui.Rect{X: cell(2, 1).X + cw*0.3, Y: cell(2, 1).Y, W: 10*(cw+gap) - cw*0.6, H: 2*ch + gap}
	legend := func(k int) ui.Rect {
		w, h := legendArea.W/2, legendArea.H/5
		return ui.Rect{X: legendArea.X + float32(k/5)*w, Y: legendArea.Y + float32(k%5)*h, W: w, H: h}
	}
	hover, kind := 0, -1
	if px, py, ok := view.PointerPosition(); ok {
		for _, e := range Elements {
			if cell(grid(e.Number)).Contains(px, py) {
				hover = e.Number
			}
		}
		for k := range categoryNames {
			if legend(k).Contains(px, py) {
				kind = k
			}
		}
	}
	if hover > 0 {
		view.Cursor(ui.CursorPointer)
	}
	if view.Pressed() && !s.held && hover > 0 {
		s.Chosen = hover
		view.SetChanged()
	}
	s.held = view.Pressed()
	chosen := s.Chosen
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		at := func(q ui.Rect) ui.Rect { return ui.Rect{X: r.X + q.X, Y: r.Y + q.Y, W: q.W, H: q.H} }
		small := max(7, min(11, cw*0.2))
		for _, e := range Elements {
			q := at(cell(grid(e.Number)))
			col := e.Category.color()
			dim := kind >= 0 && int(e.Category) != kind || !matches(e, query)
			fill, ink := col.Alpha(0.24), t.Text
			if dim {
				fill, ink = col.Alpha(0.06), t.TextMuted.Alpha(0.5)
			}
			p.Fill(q, fill, 3)
			switch e.Number {
			case chosen:
				p.Stroke(q, t.Accent, 3, 2)
			case hover:
				p.Stroke(q, t.Text.Alpha(0.5), 3, 1)
			}
			p.RichText(q.X+3, q.Y+2, 0, ui.Span{Text: fmt.Sprint(e.Number), Size: small, Color: ink, Features: "tnum"})
			sym := ui.Span{Text: e.Symbol, Size: cw * 0.36, Weight: 700, Color: ink}
			w, h := p.MeasureText(0, sym)
			y := q.Y + q.H*0.5 - h/2
			if cw >= 48 {
				y -= q.H * 0.06
			}
			p.RichText(q.X+q.W/2-w/2, y, 0, sym)
			if cw >= 48 {
				name := ui.Span{Text: e.Name, Size: max(7, cw*0.14), Color: ink}
				nw, _ := p.MeasureText(0, name)
				if nw < q.W-2 {
					p.RichText(q.X+q.W/2-nw/2, y+h-1, 0, name)
				}
			}
		}
		for i, z := range []int{57, 89} {
			q := at(cell(2, float32(5+i)))
			span := ui.Span{Text: fmt.Sprintf("%d–%d", z, z+14), Size: small, Color: t.TextMuted, Features: "tnum"}
			w, h := p.MeasureText(0, span)
			p.StrokeDashed(q, t.Border, 3, 1)
			p.RichText(q.X+q.W/2-w/2, q.Y+q.H/2-h/2, 0, span)
		}
		for k := range categoryNames {
			q := at(legend(k))
			col := Category(k).color()
			sw := ui.Rect{X: q.X, Y: q.Y + q.H/2 - 5, W: 10, H: 10}
			p.Fill(sw, col.Alpha(0.5), 2)
			p.Stroke(sw, col, 2, 1)
			ink := t.Text
			if kind >= 0 && k != kind {
				ink = t.TextMuted
			}
			span := ui.Span{Text: categoryNames[k], Size: small + 1, Color: ink}
			_, h := p.MeasureText(0, span)
			p.RichText(sw.X+16, q.Y+q.H/2-h/2, 0, span)
		}
		shown := hover
		if shown == 0 {
			shown = chosen
		}
		if shown > 0 {
			e := Elements[shown-1]
			q := at(cell(2, 0))
			q.X += cw * 0.3
			big := ui.Span{Text: e.Symbol, Size: ch * 0.62, Weight: 700, Color: e.Category.color()}
			w, h := p.MeasureText(0, big)
			p.RichText(q.X, q.Y+q.H/2-h/2, 0, big)
			lines := []ui.Span{
				{Text: e.Name + "\n", Weight: 600, Size: small + 2, Color: t.Text},
				{Text: fmt.Sprintf("%d  ·  %s  ·  %s", e.Number, e.Mass, e.Category), Size: small + 1, Color: t.TextMuted, Features: "tnum"},
			}
			_, lh := p.MeasureText(0, lines...)
			p.RichText(q.X+w+10, q.Y+q.H/2-lh/2, 0, lines...)
		}
	})
	return view
}

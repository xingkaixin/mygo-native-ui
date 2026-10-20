// Package dashboard lays widgets out on a grid of cells, as Grafana's
// dashboards do: each widget a card of its title above what the app
// builds. Dragging a widget by its title moves it, and by its bottom
// right corner resizes it, by whole cells and no smaller than its least
// size. While it moves, a placeholder shows the cells it will take, the
// widgets in its way go down, and all go up as far as they can; those
// moved aside slide to their places.
//
//	board := dashboard.View(c, &app.board, &app.widgets, 12, 64, func(i int) {
//		app.panels[i](c)
//	}).Grow(1)
//	if board.Changed() {
//		app.save()
//	}
package dashboard

import (
	"cmp"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Widget is a widget of a grid: its title, its column and row from the
// top left, and its width and height, in cells. MinW and MinH are the
// least it resizes to, a cell when they are 0.
type Widget struct {
	Title      string
	X, Y, W, H int
	MinW, MinH int
}

// State is what a grid keeps: the widget dragged or resized, and how far.
// Keep it in the app's state, one for each grid.
type State struct {
	// drag is the index of the widget dragged, plus 1, so that the zero
	// State drags none.
	drag   int
	resize bool
	dx, dy float32
}

// cells is how a grid lays out: n columns, stepX DIPs apart, rows stepY
// apart, and gap between them.
type cells struct {
	n                 int
	stepX, stepY, gap float32
}

// put places el at column x and row y, w columns wide and h rows high,
// sliding there, or at once while it follows the pointer.
func (g cells) put(el ui.Element, x, y, w, h float32, follow bool) {
	slide := func(key string, v float32) float32 {
		if follow {
			return el.Jump(key, v)
		}
		return el.AnimateTo(key, v, ui.Smooth)
	}
	x, y, w, h = slide("x", x), slide("y", y), slide("w", w), slide("h", h)
	el.Left(x*g.stepX).Top(y*g.stepY).Size(max(0, w*g.stepX-g.gap), max(0, h*g.stepY-g.gap))
}

// View creates a grid of *widgets, columns columns across and rows rowH
// high, body building the content of the widget at index i. It moves and
// resizes the widgets in place and reports it as changes once the pointer
// lets go. Give it a size, or grow it in its parent; it scrolls when the
// widgets are taller.
func View(c *ui.Context, s *State, widgets *[]Widget, columns int, rowH float32, body func(i int)) ui.Element {
	t := c.Theme()
	ws := *widgets
	gap := t.Space(3)
	columns = max(1, columns)
	if s.drag > len(ws) {
		s.drag = 0
	}
	d := s.drag - 1
	changed := false
	view := ui.Scroll(c)
	view.Children(func() {
		ui.Column(c).Padding(gap).Children(func() {
			grid := ui.Box(c)
			w := grid.LayoutBounds().W
			g := cells{columns, (w + gap) / float32(columns), rowH + gap, gap}
			shown := ws
			rows := 0
			if d >= 0 {
				shown = slices.Clone(ws)
				s.place(shown, d, g)
				_, y, _, h := s.at(ws[d], g)
				rows = int(math.Ceil(float64(y + h)))
			}
			for _, o := range shown {
				rows = max(rows, o.Y+o.H)
			}
			grid.Height(max(0, float32(rows)*g.stepY-gap))
			grid.Children(func() {
				if d >= 0 {
					p := shown[d]
					box := ui.Box(c.Key("placeholder")).Absolute().Radius(t.Space(2.5)).
						Background(t.Accent.Alpha(0.12)).Border(1, t.Accent.Alpha(0.6))
					g.put(box, float32(p.X), float32(p.Y), float32(p.W), float32(p.H), false)
				}
				dropped := false
				for i := range ws {
					if i != d {
						dropped = s.widget(c.Key(i), ws, i, shown[i], g, body) || dropped
					}
				}
				// The widget dragged is built last, to show above the others.
				if d >= 0 {
					dropped = s.widget(c.Key(d), ws, d, shown[d], g, body) || dropped
				}
				if dropped {
					if !slices.Equal(ws, shown) {
						copy(ws, shown)
						changed = true
					}
					s.drag = 0
				}
			})
			grid.Draw(func(p *ui.Painter, r ui.Rect) {
				// The cells were laid out at the width of the last frame.
				if r.W != w {
					c.Invalidate()
				}
			})
		})
	})
	if changed {
		view.SetChanged()
	}
	return view
}

// widget builds widget i of ws where shown puts it, or for the one
// dragged, where the pointer does: its title, which drags it, above what
// body builds, and its bottom right corner, which resizes it. It reports
// whether the pointer let go of it.
func (s *State) widget(c *ui.Context, ws []Widget, i int, shown Widget, g cells, body func(int)) bool {
	t := c.Theme()
	w := ws[i]
	box := ui.Column(c).Absolute().Radius(t.Space(2.5)).Background(t.Background).Border(1, t.Border).Clip().
		Shadow(0, 1, 2, 0, ui.RGB(0, 0, 0).Alpha(0.06))
	hovered := box.Hovered()
	ended := false
	box.Children(func() {
		head := ui.Row(c).Height(t.Space(9)).Padding(0, t.Space(3)).AlignItems(ui.Center).Cursor(ui.CursorGrab)
		head.Children(func() {
			ui.Text(c, w.Title).Bold().SingleLine().Shrink(1)
		})
		ended = s.track(head, i, false)
		if s.drag == i+1 && !s.resize {
			head.Cursor(ui.CursorGrabbing)
		}
		ui.Column(c).Grow(1).Padding(0, t.Space(3), t.Space(3)).Children(func() { body(i) })
		corner := ui.Box(c).Absolute().Right(0).Bottom(0).Size(t.Space(4), t.Space(4)).
			Cursor(ui.CursorResizeNWSE).Label("Resize " + w.Title)
		ended = s.track(corner, i, true) || ended
		if hovered || (s.drag == i+1 && s.resize) {
			corner.Draw(func(p *ui.Painter, r ui.Rect) {
				x, y := r.X+r.W-t.Space(1), r.Y+r.H-t.Space(1)
				for _, k := range []float32{t.Space(1), t.Space(2)} {
					p.StrokePolyline([][2]float32{{x - k, y}, {x, y - k}}, 1.5, t.TextMuted)
				}
			})
		}
	})
	x, y, cw, ch := float32(shown.X), float32(shown.Y), float32(shown.W), float32(shown.H)
	dragged := s.drag == i+1
	if dragged {
		x, y, cw, ch = s.at(w, g)
		box.Shadow(0, 8, 24, 0, ui.RGB(0, 0, 0).Alpha(0.18))
	}
	g.put(box, x, y, cw, ch, dragged)
	return ended
}

// track follows a drag of widget i by part, its title, or with resize its
// corner, and reports whether it ended.
func (s *State) track(part ui.Element, i int, resize bool) bool {
	if dx, dy, ok := part.Dragged(); ok {
		if s.drag != i+1 || s.resize != resize {
			s.drag, s.resize, s.dx, s.dy = i+1, resize, 0, 0
		}
		s.dx, s.dy = s.dx+dx, s.dy+dy
	}
	_, _, ended := part.DragEnded()
	return ended && s.drag == i+1 && s.resize == resize
}

// at returns where w, the widget dragged, shows, following the pointer:
// its column and row, and its width and height, in cells, within the grid
// and no smaller than its least size.
func (s *State) at(w Widget, g cells) (x, y, cw, ch float32) {
	x, y, cw, ch = float32(w.X), float32(w.Y), float32(w.W), float32(w.H)
	if s.resize {
		cw = max(float32(max(1, w.MinW)), min(float32(g.n-w.X), cw+s.dx/g.stepX))
		ch = max(float32(max(1, w.MinH)), ch+s.dy/g.stepY)
	} else {
		x = max(0, min(float32(g.n-w.W), x+s.dx/g.stepX))
		y = max(0, y+s.dy/g.stepY)
	}
	return x, y, cw, ch
}

// place moves ws[d], the widget dragged, to the whole cells nearest
// where it shows, and the others out of its way. Moved up, it goes before
// the widgets whose middles its top is above; moved down, after those
// whose middles its bottom is below; resized, it keeps its place.
func (s *State) place(ws []Widget, d int, g cells) {
	x, y, cw, ch := s.at(ws[d], g)
	o := &ws[d]
	// A quarter of a row keeps the rank off the middles of the others,
	// which are in whole and half rows.
	rank := float32(o.Y) - 0.25
	if s.resize {
		o.W, o.H = round(cw), round(ch)
	} else {
		o.X = round(x)
		if ty := round(y); ty > o.Y {
			rank = float32(ty+o.H) + 0.25
		} else {
			rank = float32(ty) - 0.25
		}
	}
	arrange(ws, d, rank)
}

// arrange orders the widgets by their middles, top to bottom, ws[d] by
// rank instead, and puts each as high as it goes below those before it
// that share a column with it.
func arrange(ws []Widget, d int, rank float32) {
	key := func(i int) float32 {
		if i == d {
			return rank
		}
		return float32(ws[i].Y) + float32(ws[i].H)/2
	}
	order := make([]int, len(ws))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(key(a), key(b)), cmp.Compare(ws[a].X, ws[b].X))
	})
	for k, i := range order {
		y := 0
		for _, j := range order[:k] {
			if ws[j].X < ws[i].X+ws[i].W && ws[i].X < ws[j].X+ws[j].W {
				y = max(y, ws[j].Y+ws[j].H)
			}
		}
		ws[i].Y = y
	}
}

func round(v float32) int { return int(math.Round(float64(v))) }

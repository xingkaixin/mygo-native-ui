// Package tactics draws a tactics board, a football pitch seen from above
// as a coach sketches a move on it: the pitch to scale, mown in stripes,
// with its lines, each player a disc in the side's color with the shirt
// number on it and the name under it, the ball, and arrows for runs and
// passes.
//
// Dragging a player or the ball moves it. Dragging on the grass draws an
// arrow from where the drag starts, or from the player or the ball next
// to it: a run, solid, or a pass, dashed, when Shift is held as the drag
// starts. A double click on an arrow removes it. The board changes the
// app's board in place and reports the changes:
//
//	if tactics.View(c, &app.tactics, &app.board).Grow(1).Changed() {
//		app.save()
//	}
package tactics

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Length and Width are the pitch's size in meters.
const (
	Length = 105.0
	Width  = 68.0
)

// Player is a player: the shirt number, the name, whether they play for
// the away side, and where they stand in meters, x from the home goal
// line to the right and y from the top touchline down.
type Player struct {
	Number int
	Name   string
	Away   bool
	X, Y   float64
}

// Arrow is a move from From to To, in meters: a pass if Pass is set, a
// run if not.
type Arrow struct {
	From, To [2]float64
	Pass     bool
}

// Board is what the board shows: the players, the ball and the arrows.
type Board struct {
	Players []Player
	Ball    [2]float64
	Arrows  []Arrow
}

// State is what a board keeps: the piece being dragged, or the arrow
// being drawn. Keep it in the app's state, one for each board.
type State struct {
	// drag is the player being dragged, from 1, or -1 for the ball.
	drag    int
	dx, dy  float64
	drawing bool
	arrow   Arrow
}

// Colors returns the colors of the home and the away side.
func Colors() (home, away ui.Color) {
	return ui.Oklch(0.5, 0.17, 260), ui.Oklch(0.6, 0.2, 25)
}

// margin is how far beyond the lines, in meters, the board shows, for
// the goals and a winger on the touchline.
const margin = 4.0

// View creates the board of *b, which it changes in place, reporting the
// changes. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, b *Board) ui.Element {
	t := c.Theme()
	box := ui.Box(c).MinWidth(t.Space(120)).MinHeight(t.Space(80)).Clip().Role(ui.RoleGroup).Label("Tactics board")
	bb := box.LayoutBounds()
	scale := min(float64(bb.W)/(Length+2*margin), float64(bb.H)/(Width+2*margin))
	scale = max(scale, 1)
	ox := float64(bb.W)/2 - Length/2*scale
	oy := float64(bb.H)/2 - Width/2*scale
	toPx := func(x, y float64) (float32, float32) { return float32(ox + x*scale), float32(oy + y*scale) }
	toM := func(x, y float32) [2]float64 {
		return clamp((float64(x)-ox)/scale, (float64(y)-oy)/scale)
	}
	disc, ball := t.Space(7), t.Space(3.5)
	changed := false
	box.Children(func() {
		grass := ui.Box(c).Absolute().Left(0).Top(0).Size(bb.W, bb.H).Cursor(ui.CursorCrosshair)
		if dx, dy, held := grass.Dragged(); held {
			if !s.drawing {
				px, py, _ := grass.PointerPosition()
				from := near(b, toM(px-dx, py-dy), 3)
				s.drawing, s.arrow = true, Arrow{From: from, To: from, Pass: c.Modifiers()&ui.Shift != 0}
			}
			s.arrow.To = clamp(s.arrow.To[0]+float64(dx)/scale, s.arrow.To[1]+float64(dy)/scale)
		}
		if _, _, ended := grass.DragEnded(); ended && s.drawing {
			s.drawing = false
			if math.Hypot(s.arrow.To[0]-s.arrow.From[0], s.arrow.To[1]-s.arrow.From[1]) > 2 {
				b.Arrows = append(b.Arrows, s.arrow)
				changed = true
			}
		}
		if grass.DoubleClicked() {
			px, py, _ := grass.PointerPosition()
			if k := arrowAt(b.Arrows, toM(px, py), 8/scale); k >= 0 {
				b.Arrows = append(b.Arrows[:k], b.Arrows[k+1:]...)
				changed = true
			}
		}
		move := func(e ui.Element, id int, x, y *float64) {
			if s.drag == id {
				e.Cursor(ui.CursorGrabbing)
			} else {
				e.Cursor(ui.CursorGrab)
			}
			if dx, dy, held := e.Dragged(); held {
				if s.drag != id {
					s.drag, s.dx, s.dy = id, *x, *y
				}
				s.dx += float64(dx) / scale
				s.dy += float64(dy) / scale
				p := clamp(s.dx, s.dy)
				*x, *y, changed = p[0], p[1], true
			} else if s.drag == id {
				s.drag = 0
			}
		}
		home, away := Colors()
		for i := range b.Players {
			pl := &b.Players[i]
			px, py := toPx(pl.X, pl.Y)
			e := ui.Box(c.Key(i)).Absolute().Left(px-disc/2).Top(py-disc/2).Size(disc, disc).
				Role(ui.RoleImage).Label(strconv.Itoa(pl.Number) + " " + pl.Name)
			move(e, i+1, &pl.X, &pl.Y)
			col := home
			if pl.Away {
				col = away
			}
			e.Draw(func(p *ui.Painter, r ui.Rect) {
				d := r.W / 2
				p.Shadow(r, d, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.35))
				p.Fill(r, col, d)
				p.Stroke(r, ui.RGB(255, 255, 255), d, 1.5)
				sp := ui.Span{Text: strconv.Itoa(pl.Number), Size: t.FontSize - 1, Weight: 700, Color: ui.RGB(255, 255, 255), Features: "tnum"}
				w, h := p.MeasureText(0, sp)
				p.RichText(r.X+(r.W-w)/2, r.Y+(r.H-h)/2, 0, sp)
			})
		}
		px, py := toPx(b.Ball[0], b.Ball[1])
		e := ui.Box(c.Key("ball")).Absolute().Left(px-ball/2).Top(py-ball/2).Size(ball, ball).Role(ui.RoleImage).Label("Ball")
		move(e, -1, &b.Ball[0], &b.Ball[1])
		e.Draw(func(p *ui.Painter, r ui.Rect) {
			d := r.W / 2
			p.Shadow(r, d, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.4))
			p.Fill(r, ui.RGB(255, 255, 255), d)
			p.Stroke(r, ui.RGB(26, 26, 26), d, 1)
			k := d * 0.45
			p.Fill(ui.Rect{X: r.X + d - k, Y: r.Y + d - k, W: 2 * k, H: 2 * k}, ui.RGB(26, 26, 26), k)
		})
	})
	if changed {
		box.SetChanged()
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != bb.W || r.H != bb.H {
			c.Invalidate()
		}
		at := func(x, y float64) [2]float32 { px, py := toPx(x, y); return [2]float32{r.X + px, r.Y + py} }
		pitch(p, t, r, at, scale)
		arrows := b.Arrows
		if s.drawing {
			arrows = append(arrows[:len(arrows):len(arrows)], s.arrow)
		}
		for _, a := range arrows {
			arrow(p, at(a.From[0], a.From[1]), at(a.To[0], a.To[1]), a.Pass)
		}
	})
	box.DrawOver(func(p *ui.Painter, r ui.Rect) {
		for _, pl := range b.Players {
			x, y := toPx(pl.X, pl.Y)
			sp := ui.Span{Text: pl.Name, Size: t.FontSize - 3, Weight: 600, Color: ui.RGB(255, 255, 255)}
			w, _ := p.MeasureText(0, sp)
			p.RichText(r.X+x-w/2, r.Y+y+disc/2+2, 0, sp)
		}
	})
	return box
}

func clamp(x, y float64) [2]float64 {
	return [2]float64{min(max(x, -margin), Length+margin), min(max(y, -margin), Width+margin)}
}

// near returns where the player or the ball nearest to q stands, if one
// is within r meters, or q.
func near(b *Board, q [2]float64, r float64) [2]float64 {
	best, at := r, q
	try := func(x, y float64) {
		if d := math.Hypot(x-q[0], y-q[1]); d < best {
			best, at = d, [2]float64{x, y}
		}
	}
	for _, pl := range b.Players {
		try(pl.X, pl.Y)
	}
	try(b.Ball[0], b.Ball[1])
	return at
}

// arrowAt returns the index of the last arrow within r meters of q, or
// -1.
func arrowAt(arrows []Arrow, q [2]float64, r float64) int {
	for k := len(arrows) - 1; k >= 0; k-- {
		a, z := arrows[k].From, arrows[k].To
		dx, dy := z[0]-a[0], z[1]-a[1]
		u := max(0, min(1, ((q[0]-a[0])*dx+(q[1]-a[1])*dy)/(dx*dx+dy*dy)))
		if math.Hypot(a[0]+u*dx-q[0], a[1]+u*dy-q[1]) <= r {
			return k
		}
	}
	return -1
}

// pitch paints the grass and the lines, at mapping meters to the window.
func pitch(p *ui.Painter, t *ui.Theme, r ui.Rect, at func(x, y float64) [2]float32, scale float64) {
	l, ch := float32(0.56), float32(0.12)
	if t.Dark {
		l, ch = 0.42, 0.08
	}
	p.Fill(r, ui.Oklch(l-0.03, ch, 145), 0)
	const stripes = 14
	for i := range stripes {
		a, z := at(Length*float64(i)/stripes, -margin), at(Length*float64(i+1)/stripes, Width+margin)
		if i%2 == 1 {
			p.Fill(ui.Rect{X: a[0], Y: a[1], W: z[0] - a[0], H: z[1] - a[1]}, ui.Oklch(l+0.03, ch, 145), 0)
		}
	}
	line := ui.RGBA(255, 255, 255, 0.85)
	w := float32(max(1.5, 0.12*scale))
	rect := func(x0, y0, x1, y1 float64) {
		a, z := at(min(x0, x1), min(y0, y1)), at(max(x0, x1), max(y0, y1))
		p.Stroke(ui.Rect{X: a[0], Y: a[1], W: z[0] - a[0], H: z[1] - a[1]}, line, 0, w)
	}
	circle := func(x, y, rad float64, fill bool) {
		a, z := at(x-rad, y-rad), at(x+rad, y+rad)
		box := ui.Rect{X: a[0], Y: a[1], W: z[0] - a[0], H: z[1] - a[1]}
		if fill {
			p.Fill(box, line, box.W/2)
		} else {
			p.Stroke(box, line, box.W/2, w)
		}
	}
	arc := func(x, y, rad, from, to float64) {
		var pts [][2]float32
		for k := range 25 {
			a := from + (to-from)*float64(k)/24
			pts = append(pts, at(x+rad*math.Cos(a), y+rad*math.Sin(a)))
		}
		p.StrokePolyline(pts, w, line)
	}
	rect(0, 0, Length, Width)
	a, z := at(Length/2, 0), at(Length/2, Width)
	p.Line(a[0], a[1], z[0], z[1], w, line)
	circle(Length/2, Width/2, 9.15, false)
	circle(Length/2, Width/2, 0.3, true)
	spread := math.Acos((16.5 - 11) / 9.15)
	for _, side := range []struct{ x, dir, turn float64 }{{0, 1, 0}, {Length, -1, math.Pi}} {
		x, d := side.x, side.dir
		rect(x, Width/2-20.16, x+d*16.5, Width/2+20.16)
		rect(x, Width/2-9.16, x+d*5.5, Width/2+9.16)
		rect(x, Width/2-3.66, x-d*2, Width/2+3.66)
		circle(x+d*11, Width/2, 0.3, true)
		arc(x+d*11, Width/2, 9.15, side.turn-spread, side.turn+spread)
	}
	for _, k := range []struct{ x, y, from float64 }{{0, 0, 0}, {0, Width, -math.Pi / 2}, {Length, 0, math.Pi / 2}, {Length, Width, math.Pi}} {
		arc(k.x, k.y, 1, k.from, k.from+math.Pi/2)
	}
}

// arrow paints an arrow from a to b, dashed for a pass.
func arrow(p *ui.Painter, a, b [2]float32, pass bool) {
	col := ui.Oklch(0.92, 0.16, 95)
	if pass {
		col = ui.RGB(255, 255, 255)
	}
	dx, dy := b[0]-a[0], b[1]-a[1]
	n := float32(math.Hypot(float64(dx), float64(dy)))
	if n < 1 {
		return
	}
	ux, uy := dx/n, dy/n
	head := float32(11)
	end := [2]float32{b[0] - ux*head*0.8, b[1] - uy*head*0.8}
	var path ui.Path
	path.MoveTo(a[0], a[1]).LineTo(end[0], end[1])
	if pass {
		path.Dash(0, 8, 6)
	}
	p.StrokePath(&path, 2.5, col)
	var tip ui.Path
	tip.MoveTo(b[0], b[1]).
		LineTo(b[0]-ux*head-uy*head*0.5, b[1]-uy*head+ux*head*0.5).
		LineTo(b[0]-ux*head+uy*head*0.5, b[1]-uy*head-ux*head*0.5).
		Close()
	p.FillPath(&tip, col)
}

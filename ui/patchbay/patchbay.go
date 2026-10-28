// Package patchbay patches modules together with cables, as a modular
// synthesizer is patched: each module a panel with its name and its
// jacks, outputs on a dark plate, and each cable a colored cord hanging
// between two jacks, sagging the more the farther apart they are.
//
// Dragging from a jack pulls a new cable out of it, following the
// pointer, and the jacks it can go into are ringed: an output's cable
// goes into an input and an input's into an output. Letting go over one
// plugs it in; an input takes one cable, so plugging into a taken input
// replaces its cable. Dragging from an input that has a cable pulls that
// cable's end out of it, to plug elsewhere or to drop, which removes the
// cable. The patch changes the app's cables in place and reports the
// changes:
//
//	if patchbay.View(c, &app.rack, app.modules, &app.cables).Changed() {
//		app.route()
//	}
package patchbay

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Jack is a socket of a module: its name and whether it is an output.
type Jack struct {
	Name string
	Out  bool
}

// Module is a panel of jacks.
type Module struct {
	Name  string
	Jacks []Jack
}

// Port is a jack of a module, by their indices.
type Port struct {
	Module, Jack int
}

// Cable joins an output, From, to an input, To, in color Color, an index
// into the patch's colors.
type Cable struct {
	From, To Port
	Color    int
}

// State is what a patch keeps: the jack a cable is being pulled from and
// where the pointer has taken its end. Keep it in the app's state, one
// for each patch.
type State struct {
	// from is the jack the pulled cable hangs from, and active whether a
	// cable is being pulled.
	from   Port
	active bool
	color  int
	x, y   float32
	// rects are the boxes of the sockets in the last frame, in window
	// coordinates, where a pulled cable can be let go.
	rects map[Port]ui.Rect
}

// Color returns cable color i, as the patch draws it.
func Color(i int) ui.Color {
	hues := [...]float32{25, 85, 145, 230, 300, 0}
	if k := i % len(hues); k == len(hues)-1 {
		return ui.Oklch(0.5, 0, 0)
	}
	return ui.Oklch(0.66, 0.17, hues[i%len(hues)])
}

// View creates the panels of modules and the cables of *cables, which it
// changes in place as the user patches, reporting the changes.
func View(c *ui.Context, s *State, modules []Module, cables *[]Cable) ui.Element {
	t := c.Theme()
	root := ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Stretch).Role(ui.RoleGroup).Label("Patch")
	jacks := map[Port]ui.Rect{}
	changed := false
	root.Children(func() {
		for mi, m := range modules {
			ui.Column(c.Key(mi)).Width(t.Space(30)).Padding(t.Space(3), t.Space(2)).Gap(t.Space(3)).
				Background(t.Surface).Border(1, t.Border).Radius(t.Space(1.5)).Role(ui.RoleGroup).Label(m.Name).Children(func() {
				ui.Text(c, m.Name).FontWeight(700).FontSize(t.FontSize - 1).LetterSpacing(1).TextAlign(ui.Center).FillWidth()
				ui.Grid(c).FillWidth().ColumnTracks(ui.Fr(1), ui.Fr(1)).GapY(t.Space(3)).Children(func() {
					for ji, j := range m.Jacks {
						port := Port{mi, ji}
						jacks[port] = jack(c.Key(ji), s, port, j, modules, cables, &changed)
					}
				})
			})
		}
	})
	s.rects = jacks
	if changed {
		root.SetChanged()
	}
	root.DrawOver(func(p *ui.Painter, r ui.Rect) {
		for _, cb := range *cables {
			a, okA := jacks[cb.From]
			b, okB := jacks[cb.To]
			if okA && okB {
				cord(p, mid(a), mid(b), Color(cb.Color))
			}
		}
		if s.active {
			if a, ok := jacks[s.from]; ok {
				cord(p, mid(a), [2]float32{s.x, s.y}, Color(s.color))
			}
		}
	})
	return root
}

// jack creates the jack at port and handles a cable pulled from it,
// returning its socket's box in window coordinates.
func jack(c *ui.Context, s *State, port Port, j Jack, modules []Module, cables *[]Cable, changed *bool) ui.Rect {
	t := c.Theme()
	size := t.Space(7)
	var socket ui.Element
	ui.Column(c).AlignItems(ui.Center).Gap(t.Space(1)).Children(func() {
		plate := ui.Box(c).Size(size+t.Space(2), size+t.Space(2)).Radius(t.Space(1.5)).Center()
		if j.Out {
			plate.Background(t.Text.Alpha(0.85))
		}
		plate.Children(func() {
			socket = ui.Box(c).Size(size, size).Cursor(ui.CursorGrab).Role(ui.RoleButton).Label(j.Name)
		})
		name := ui.Text(c, j.Name).FontSize(t.FontSize - 3).FontWeight(600).TextColor(t.TextMuted).SingleLine()
		if j.Out {
			name.TextColor(t.Text)
		}
	})
	box := socket.LayoutBounds()
	if dx, dy, held := socket.Dragged(); held {
		if !s.active {
			s.active, s.x, s.y = true, box.X+box.W/2, box.Y+box.H/2
			s.from, s.color = port, nextColor(*cables)
			if !j.Out {
				if k := into(*cables, port); k >= 0 {
					s.from, s.color = (*cables)[k].From, (*cables)[k].Color
					*cables = append((*cables)[:k], (*cables)[k+1:]...)
					*changed = true
				}
			}
		}
		s.x += dx
		s.y += dy
	}
	if _, _, ended := socket.DragEnded(); ended && s.active {
		s.active = false
		if to, ok := portAt(s.rects, s.x, s.y); ok {
			if cb, ok := plug(modules, s.from, to, s.color); ok {
				if k := into(*cables, cb.To); k >= 0 {
					*cables = append((*cables)[:k], (*cables)[k+1:]...)
				}
				*cables = append(*cables, cb)
				*changed = true
			}
		}
	}
	ring := false
	if s.active {
		from := modules[s.from.Module].Jacks[s.from.Jack]
		ring = from.Out != j.Out && port != s.from
	}
	socket.Draw(func(p *ui.Painter, r ui.Rect) {
		d := r.W / 2
		cx, cy := r.X+d, r.Y+d
		nut := ui.Oklch(0.82, 0, 0)
		p.Fill(r, nut, d)
		p.Stroke(r, ui.Oklch(0.55, 0, 0), d, 1)
		hole := d * 0.5
		p.Fill(ui.Rect{X: cx - hole, Y: cy - hole, W: 2 * hole, H: 2 * hole}, ui.Oklch(0.12, 0, 0), hole)
		if ring {
			g := d + 3
			p.Stroke(ui.Rect{X: cx - g, Y: cy - g, W: 2 * g, H: 2 * g}, t.Accent, g, 2)
		}
	})
	return box
}

// portAt returns the jack of rects under x, y, or a few DIPs from it.
func portAt(rects map[Port]ui.Rect, x, y float32) (Port, bool) {
	for port, r := range rects {
		if x >= r.X-4 && x < r.X+r.W+4 && y >= r.Y-4 && y < r.Y+r.H+4 {
			return port, true
		}
	}
	return Port{}, false
}

// plug returns the cable between a and b, an output and an input in
// either order, in color.
func plug(modules []Module, a, b Port, color int) (Cable, bool) {
	ja, jb := modules[a.Module].Jacks[a.Jack], modules[b.Module].Jacks[b.Jack]
	switch {
	case ja.Out && !jb.Out:
		return Cable{From: a, To: b, Color: color}, true
	case !ja.Out && jb.Out:
		return Cable{From: b, To: a, Color: color}, true
	}
	return Cable{}, false
}

// into returns the index of the cable into input port, or -1.
func into(cables []Cable, port Port) int {
	for k, cb := range cables {
		if cb.To == port {
			return k
		}
	}
	return -1
}

// nextColor returns the color least used by cables.
func nextColor(cables []Cable) int {
	var used [6]int
	for _, cb := range cables {
		used[cb.Color%len(used)]++
	}
	best := 0
	for i, n := range used {
		if n < used[best] {
			best = i
		}
	}
	return best
}

func mid(r ui.Rect) [2]float32 { return [2]float32{r.X + r.W/2, r.Y + r.H/2} }

// cord paints a cable from a to b, hanging lower the longer it is, with
// a plug at each end.
func cord(p *ui.Painter, a, b [2]float32, col ui.Color) {
	dist := float32(math.Hypot(float64(b[0]-a[0]), float64(b[1]-a[1])))
	sag := 20 + dist*0.3
	mx, my := (a[0]+b[0])/2, max(a[1], b[1])+sag
	var path ui.Path
	path.MoveTo(a[0], a[1]).QuadTo(mx, my, b[0], b[1])
	p.StrokePath(&path, 7, col.Mix(ui.RGB(0, 0, 0), 0.35).Alpha(0.9))
	p.StrokePath(&path, 5, col)
	for _, e := range [][2]float32{a, b} {
		d := float32(6)
		p.Fill(ui.Rect{X: e[0] - d, Y: e[1] - d, W: 2 * d, H: 2 * d}, col.Mix(ui.RGB(0, 0, 0), 0.25), d)
		d = 3
		p.Fill(ui.Rect{X: e[0] - d, Y: e[1] - d, W: 2 * d, H: 2 * d}, ui.Oklch(0.8, 0, 0), d)
	}
}

// Package meshgradient edits a mesh gradient, as SwiftUI's MeshGradient
// and design tools' mesh fills do: a grid of points, each with a color,
// blended smoothly between them by a shader. Dragging a point bends the
// mesh, the corners staying put and the points of the edges sliding
// along them; a click chooses a point, whose color the well below edits,
// and the arrows nudge it, ten times further with Shift. The mesh's
// lines show or hide. Paint fills any rectangle with a mesh, as a
// background.
//
//	app.mesh = meshgradient.New(3, 3, colors)
//	meshgradient.View(c, &app.editor, &app.mesh).Grow(1)
//	...
//	code := app.mesh.SwiftUI()
package meshgradient

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// Mesh is a grid of points with their colors.
type Mesh struct {
	// Cols and Rows are how many points the mesh has across and down,
	// from 2 to 5.
	Cols, Rows int
	// Points are where the points are, row by row, from 0 to 1 across
	// and down the rectangle the mesh fills.
	Points []ui.Point
	// Colors are the points' colors, row by row.
	Colors []ui.Color
}

// New returns a mesh of cols by rows points spread evenly, from 2 to 5
// each way, colored by colors in turn.
func New(cols, rows int, colors []ui.Color) Mesh {
	cols, rows = max(2, min(5, cols)), max(2, min(5, rows))
	m := Mesh{Cols: cols, Rows: rows}
	for j := range rows {
		for i := range cols {
			m.Points = append(m.Points, ui.Point{X: float32(i) / float32(cols-1), Y: float32(j) / float32(rows-1)})
			c := ui.RGB(128, 128, 128)
			if len(colors) > 0 {
				c = colors[(j*cols+i)%len(colors)]
			}
			m.Colors = append(m.Colors, c)
		}
	}
	return m
}

func (m Mesh) valid() bool {
	return m.Cols >= 2 && m.Rows >= 2 && m.Cols <= 5 && m.Rows <= 5 && len(m.Points) == m.Cols*m.Rows && len(m.Colors) == len(m.Points)
}

// SwiftUI returns the mesh as SwiftUI's MeshGradient, which blends its
// colors a little differently.
func (m Mesh) SwiftUI() string {
	var b strings.Builder
	fmt.Fprintf(&b, "MeshGradient(width: %d, height: %d, points: [\n", m.Cols, m.Rows)
	num := func(v float32) string { return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.3f", v), "0"), ".") }
	for j := range m.Rows {
		b.WriteString("    ")
		for i := range m.Cols {
			pt := m.Points[j*m.Cols+i]
			fmt.Fprintf(&b, "[%s, %s], ", num(pt.X), num(pt.Y))
		}
		b.WriteString("\n")
	}
	b.WriteString("], colors: [\n")
	for j := range m.Rows {
		b.WriteString("    ")
		for i := range m.Cols {
			c := m.Colors[j*m.Cols+i].SRGB()
			fmt.Fprintf(&b, "Color(red: %s, green: %s, blue: %s), ", num(float32(c.R)/255), num(float32(c.G)/255), num(float32(c.B)/255))
		}
		b.WriteString("\n")
	}
	b.WriteString("])")
	return b.String()
}

// oklab returns c in Oklab.
func oklab(c ui.Color) [3]float32 {
	c = c.SRGB()
	linear := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	r, g, b := linear(c.R), linear(c.G), linear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return [3]float32{
		float32(0.2104542553*l + 0.7936177850*m - 0.0040720468*s),
		float32(1.9779984951*l - 2.4285922050*m + 0.4505937099*s),
		float32(0.0259040371*l + 0.7827717662*m - 0.8086757660*s),
	}
}

// Paint fills r with the mesh.
func Paint(p *ui.Painter, r ui.Rect, m Mesh) {
	if !m.valid() {
		return
	}
	pts := make([]float32, 0, 2*len(m.Points))
	for _, pt := range m.Points {
		pts = append(pts, pt.X*r.W, pt.Y*r.H)
	}
	labs := make([]float32, 0, 3*len(m.Colors))
	for _, c := range m.Colors {
		lab := oklab(c)
		labs = append(labs, lab[:]...)
	}
	v := meshShader.NewValues().Int(uCols, int32(m.Cols)).Int(uRows, int32(m.Rows)).Floats(uP, pts...).Floats(uC, labs...)
	meshShader.Draw(p, r, shader.Options{Values: v})
}

// State is what an editor keeps: the point chosen, the one dragged, and
// whether the mesh's lines show. Keep it in the app's state, one for
// each editor.
type State struct {
	chosen, dragging int
	held, hideLines  bool
}

// move moves point i of m by dx, dy of the rectangle, the corners not at
// all and the edges' points along their edge.
func move(m *Mesh, i int, dx, dy float32) {
	col, row := i%m.Cols, i/m.Cols
	pt := &m.Points[i]
	if col > 0 && col < m.Cols-1 {
		pt.X = max(0, min(1, pt.X+dx))
	}
	if row > 0 && row < m.Rows-1 {
		pt.Y = max(0, min(1, pt.Y+dy))
	}
}

// View creates the editor of *m. Changed reports a change. Give it a
// size, or grow it in its parent.
func View(c *ui.Context, s *State, m *Mesh) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(3))
	if !m.valid() {
		return root
	}
	s.chosen = max(0, min(len(m.Points)-1, s.chosen))
	changed := false
	root.Children(func() {
		canvas := ui.Box(c).Grow(1).Focusable().FocusRing(false).Role(ui.RoleGroup).Label("Mesh gradient")
		handle := t.Space(2)
		// The mesh leaves room around it for the edges' handles to be
		// pressed whole.
		pad := handle * 1.5
		b := canvas.LayoutBounds()
		b = ui.Rect{X: pad, Y: pad, W: max(1, b.W-2*pad), H: max(1, b.H-2*pad)}
		if canvas.Pressed() && !s.held {
			s.dragging = -1
			if px, py, ok := canvas.PointerPosition(); ok {
				best := float32(handle * 3)
				for i, pt := range m.Points {
					if d := float32(math.Hypot(float64(b.X+pt.X*b.W-px), float64(b.Y+pt.Y*b.H-py))); d < best {
						best, s.dragging, s.chosen = d, i, i
					}
				}
			}
		}
		s.held = canvas.Pressed()
		if dx, dy, ok := canvas.Dragged(); ok && s.dragging >= 0 {
			move(m, s.dragging, dx/b.W, dy/b.H)
			changed = true
			canvas.Cursor(ui.CursorGrabbing)
		}
		for _, mods := range []ui.Modifiers{0, ui.Shift} {
			step := float32(0.005)
			if mods == ui.Shift {
				step = 0.05
			}
			for _, k := range []struct {
				key    ui.Key
				dx, dy float32
			}{{ui.KeyLeft, -step, 0}, {ui.KeyRight, step, 0}, {ui.KeyUp, 0, -step}, {ui.KeyDown, 0, step}} {
				if canvas.Shortcut(mods, k.key) {
					move(m, s.chosen, k.dx, k.dy)
					changed = true
				}
			}
		}
		mesh, chosen, lines, focus := *m, s.chosen, !s.hideLines, canvas.FocusVisible()
		canvas.Draw(func(p *ui.Painter, r ui.Rect) {
			r = ui.Rect{X: r.X + pad, Y: r.Y + pad, W: r.W - 2*pad, H: r.H - 2*pad}
			p.Clip(r, t.Radius, func() { Paint(p, r, mesh) })
			at := func(i int) [2]float32 {
				return [2]float32{r.X + mesh.Points[i].X*r.W, r.Y + mesh.Points[i].Y*r.H}
			}
			if lines {
				for j := range mesh.Rows {
					var row [][2]float32
					for i := range mesh.Cols {
						row = append(row, at(j*mesh.Cols+i))
					}
					p.StrokePolyline(row, 1, ui.RGBA(255, 255, 255, 0.6))
				}
				for i := range mesh.Cols {
					var col [][2]float32
					for j := range mesh.Rows {
						col = append(col, at(j*mesh.Cols+i))
					}
					p.StrokePolyline(col, 1, ui.RGBA(255, 255, 255, 0.6))
				}
			}
			for i := range mesh.Points {
				xy := at(i)
				rad := handle
				if i == chosen {
					rad = handle * 1.5
				}
				dot := ui.Rect{X: xy[0] - rad, Y: xy[1] - rad, W: 2 * rad, H: 2 * rad}
				p.Shadow(dot, rad, 0, 1, 4, 0, ui.RGB(0, 0, 0).Alpha(0.35))
				p.Fill(dot, mesh.Colors[i], rad)
				p.Stroke(dot, ui.RGB(255, 255, 255), rad, 2)
			}
			if focus {
				p.FocusRing(r, [4]float32{t.Radius, t.Radius, t.Radius, t.Radius})
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("Point %d, %d", chosen%m.Cols+1, chosen/m.Cols+1)).FontFeatures("tnum").TextColor(t.TextMuted)
			if ui.ColorWell(c, &m.Colors[s.chosen]).Label("Point color").Changed() {
				changed = true
			}
			ui.Spacer(c)
			ui.Checkbox(c, &lines, "Show mesh")
			s.hideLines = !lines
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

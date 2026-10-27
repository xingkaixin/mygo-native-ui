// Package pagesetup sets up the page a document prints on, as the Page
// Setup of word processors does: the paper, portrait or landscape, the
// margins and the scale, beside a preview of the page with the app's
// content in it. Dragging a margin's dashed line in the preview moves
// the margin; Normal, Narrow and Wide set all four at once. Lengths show
// in inches where the region measures in them, as the United States
// does, and in millimeters elsewhere.
//
//	pagesetup.View(c, &app.setupView, &app.setup, func(p *ui.Painter, r ui.Rect) {
//		p.RichText(r.X, r.Y, r.W, ui.Span{Text: app.text, Size: 11})
//	}).Grow(1)
//
// The setup keeps lengths in points, 72 to the inch, as PDF does.
package pagesetup

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Paper is a paper size, in points.
type Paper struct {
	Name          string
	Width, Height float64
}

// Papers are the paper sizes offered.
var Papers = []Paper{
	{"US Letter", 612, 792},
	{"US Legal", 612, 1008},
	{"Tabloid", 792, 1224},
	{"A3", 841.89, 1190.55},
	{"A4", 595.28, 841.89},
	{"A5", 419.53, 595.28},
	{"B5", 498.9, 708.66},
	{"Envelope #10", 297, 684},
}

// Setup is how a document prints: on what paper and which way, the
// margins in points, and Scale, 1 for 100%.
type Setup struct {
	Paper                    Paper
	Landscape                bool
	Top, Right, Bottom, Left float64
	Scale                    float64
}

// Size returns the page's width and height as it prints, turned when
// landscape.
func (s Setup) Size() (w, h float64) {
	if s.Landscape {
		return s.Paper.Height, s.Paper.Width
	}
	return s.Paper.Width, s.Paper.Height
}

// margins are the presets of the margins, in points: top, right, bottom
// and left.
var margins = []struct {
	name string
	m    [4]float64
}{
	{"Normal", [4]float64{72, 72, 72, 72}},
	{"Narrow", [4]float64{36, 36, 36, 36}},
	{"Wide", [4]float64{72, 144, 72, 144}},
}

// minContent is the least width and height the margins leave, in points.
const minContent = 72

// State is what a page setup keeps: the margin dragged in the preview.
// Keep it in the app's state, one for each view.
type State struct {
	dragging int
	held     bool
}

// unit returns the length the region measures in, in points, and its
// name.
func unit(c *ui.Context) (float64, string) {
	loc := c.Region().Locale
	for _, r := range []string{"-US", "-LR", "-MM"} {
		if strings.HasSuffix(loc, r) {
			return 72, "in"
		}
	}
	return 72 / 25.4, "mm"
}

// View creates the setup of *setup, page drawing the app's content in r,
// the part of the page within the margins, in points as it prints at the
// scale; the preview shrinks it to fit. Changed reports a change. Give it
// a size, or grow it in its parent.
func View(c *ui.Context, s *State, setup *Setup, page func(p *ui.Painter, r ui.Rect)) ui.Element {
	t := c.Theme()
	if setup.Paper.Width == 0 {
		setup.Paper = Papers[0]
	}
	if setup.Scale == 0 {
		setup.Scale = 1
	}
	root := ui.Row(c).Gap(t.Space(6))
	changed := false
	root.Children(func() {
		if s.preview(c, setup, page) {
			changed = true
		}
		if s.controls(c, setup) {
			changed = true
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// preview creates the page with its margins and content, and reports a
// margin dragged.
func (s *State) preview(c *ui.Context, setup *Setup, page func(p *ui.Painter, r ui.Rect)) bool {
	t := c.Theme()
	canvas := ui.Box(c).Grow(1).FillHeight().Role(ui.RoleImage).Label("Page preview")
	b := canvas.LayoutBounds()
	pw, ph := setup.Size()
	pad := float64(t.Space(8))
	k := max(0.01, min((float64(b.W)-2*pad)/pw, (float64(b.H)-2*pad)/ph))
	ox, oy := (float64(b.W)-pw*k)/2, (float64(b.H)-ph*k)/2
	// edges are where the margins' lines are in the canvas: top, right,
	// bottom and left.
	edges := [4]float64{oy + setup.Top*k, ox + (pw-setup.Right)*k, oy + (ph-setup.Bottom)*k, ox + setup.Left*k}
	near := func(x, y float32) int {
		best, edge := float64(t.Space(1.5)), -1
		for i, e := range edges {
			v := float64(y)
			if i%2 == 1 {
				v = float64(x)
			}
			if d := math.Abs(v - e); d < best {
				best, edge = d, i
			}
		}
		return edge
	}
	px, py, over := canvas.PointerPosition()
	inPage := float64(px) > ox && float64(px) < ox+pw*k && float64(py) > oy && float64(py) < oy+ph*k
	if canvas.Pressed() && !s.held {
		s.dragging = -1
		if inPage {
			s.dragging = near(px, py)
		}
	}
	s.held = canvas.Pressed()
	changed := false
	if s.held && s.dragging >= 0 {
		m := [4]*float64{&setup.Top, &setup.Right, &setup.Bottom, &setup.Left}
		var v float64
		switch s.dragging {
		case 0:
			v = (float64(py) - oy) / k
		case 1:
			v = pw - (float64(px)-ox)/k
		case 2:
			v = ph - (float64(py)-oy)/k
		case 3:
			v = (float64(px) - ox) / k
		}
		span := ph
		if s.dragging%2 == 1 {
			span = pw
		}
		v = math.Round(max(0, min(span-*m[(s.dragging+2)%4]-minContent, v)))
		if v != *m[s.dragging] {
			*m[s.dragging] = v
			changed = true
		}
	}
	edge := s.dragging
	if !s.held {
		edge = -1
		if over && inPage {
			edge = near(px, py)
		}
	}
	if edge >= 0 {
		canvas.Cursor(ui.CursorResizeNS)
		if edge%2 == 1 {
			canvas.Cursor(ui.CursorResizeEW)
		}
	}
	size := ui.Rect{X: float32(ox), Y: float32(oy), W: float32(pw * k), H: float32(ph * k)}
	area := ui.Rect{
		X: float32(ox + setup.Left*k), Y: float32(oy + setup.Top*k),
		W: float32((pw - setup.Left - setup.Right) * k), H: float32((ph - setup.Top - setup.Bottom) * k),
	}
	scale, dragging := float32(k*setup.Scale), s.held && s.dragging >= 0
	per, name := unit(c)
	label := ""
	if dragging {
		label = length([4]float64{setup.Top, setup.Right, setup.Bottom, setup.Left}[edge], per, name)
	}
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		pg := ui.Rect{X: r.X + size.X, Y: r.Y + size.Y, W: size.W, H: size.H}
		p.Shadow(pg, 2, 0, 2, 12, 0, ui.RGB(0, 0, 0).Alpha(0.25))
		p.Fill(pg, ui.RGB(255, 255, 255), 2)
		in := ui.Rect{X: r.X + area.X, Y: r.Y + area.Y, W: area.W, H: area.H}
		if page != nil {
			p.Clip(in, 0, func() {
				p.Transform(ui.Scaling(scale, scale), in.X, in.Y, func() {
					page(p, ui.Rect{X: in.X, Y: in.Y, W: in.W / scale, H: in.H / scale})
				})
			})
		}
		guide := ui.RGB(0, 0, 0).Alpha(0.2)
		lines := [4]ui.Rect{
			{X: pg.X, Y: in.Y, W: pg.W, H: 0},
			{X: in.X + in.W, Y: pg.Y, W: 0, H: pg.H},
			{X: pg.X, Y: in.Y + in.H, W: pg.W, H: 0},
			{X: in.X, Y: pg.Y, W: 0, H: pg.H},
		}
		for i, l := range lines {
			col, w := guide, float32(1)
			if i == edge {
				col, w = t.Accent, 1.5
			}
			dashes(p, l, col, w)
		}
		if label != "" {
			lw, lh := p.MeasureText(0, ui.Span{Text: label, Size: t.FontSize - 1})
			x, y := r.X+px+12, r.Y+py+12
			tag := ui.Rect{X: x - 6, Y: y - 3, W: lw + 12, H: lh + 6}
			p.Fill(tag, t.Accent, 4)
			p.RichText(x, y, lw+1, ui.Span{Text: label, Size: t.FontSize - 1, Color: ui.RGB(255, 255, 255)})
		}
	})
	return changed
}

// dashes draws the line l, flat one way, in dashes.
func dashes(p *ui.Painter, l ui.Rect, col ui.Color, w float32) {
	const dash, gap = 4, 3
	if l.H == 0 {
		for x := l.X; x < l.X+l.W; x += dash + gap {
			p.Line(x, l.Y, min(x+dash, l.X+l.W), l.Y, w, col)
		}
		return
	}
	for y := l.Y; y < l.Y+l.H; y += dash + gap {
		p.Line(l.X, y, l.X, min(y+dash, l.Y+l.H), w, col)
	}
}

// length returns v points in the unit of per points, named name.
func length(v, per float64, name string) string {
	if name == "in" {
		return fmt.Sprintf("%.2f in", v/per)
	}
	return fmt.Sprintf("%.0f mm", v/per)
}

// controls creates the paper, orientation, margins and scale, and
// reports a change.
func (s *State) controls(c *ui.Context, setup *Setup) bool {
	t := c.Theme()
	per, name := unit(c)
	changed := false
	pw, ph := setup.Size()
	// fit keeps the margins within the page as its size changes.
	fit := func() {
		pw, ph = setup.Size()
		setup.Left = max(0, min(setup.Left, pw-setup.Right-minContent))
		setup.Right = max(0, min(setup.Right, pw-setup.Left-minContent))
		setup.Top = max(0, min(setup.Top, ph-setup.Bottom-minContent))
		setup.Bottom = max(0, min(setup.Bottom, ph-setup.Top-minContent))
		changed = true
	}
	ui.Column(c).Width(t.Space(64)).Gap(t.Space(5)).Children(func() {
		ui.Field(c, "Paper", func() {
			names := make([]string, len(Papers))
			for i, p := range Papers {
				names[i] = p.Name
			}
			paper := setup.Paper.Name
			if ui.Select(c, &paper, names).FillWidth().Changed() {
				for _, p := range Papers {
					if p.Name == paper {
						setup.Paper = p
					}
				}
				fit()
			}
			w, h := setup.Paper.Width/per, setup.Paper.Height/per
			size := fmt.Sprintf("%.0f × %.0f mm", w, h)
			if name == "in" {
				size = fmt.Sprintf("%g × %g in", math.Round(w*100)/100, math.Round(h*100)/100)
			}
			ui.Text(c, size).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Field(c, "Orientation", func() {
			way := 0
			if setup.Landscape {
				way = 1
			}
			if ui.Segmented(c, &way, "Portrait", "Landscape").Changed() {
				setup.Landscape = way == 1
				fit()
			}
		})
		ui.Field(c, "Margins", func() {
			ui.Row(c).Gap(t.Space(1.5)).Children(func() {
				for _, m := range margins {
					cur := m.m == [4]float64{setup.Top, setup.Right, setup.Bottom, setup.Left}
					b := ui.Button(c.Key(m.name), m.name).Grow(1)
					if cur {
						b.Background(t.Selection)
					}
					if b.Clicked() {
						setup.Top, setup.Right, setup.Bottom, setup.Left = m.m[0], m.m[1], m.m[2], m.m[3]
						fit()
					}
				}
			})
			step, hi := 1.0, 0.0
			if name == "in" {
				step = 0.05
			}
			ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(t.Space(3)).GapY(t.Space(2)).AlignItems(ui.Center).Children(func() {
				for _, side := range []struct {
					label string
					v     *float64
					span  float64
					other float64
				}{
					{"Top", &setup.Top, ph, setup.Bottom},
					{"Bottom", &setup.Bottom, ph, setup.Top},
					{"Left", &setup.Left, pw, setup.Right},
					{"Right", &setup.Right, pw, setup.Left},
				} {
					ui.Text(c.Key(side.label), side.label).TextColor(t.TextMuted)
					hi = max(0, side.span-side.other-minContent) / per
					v := *side.v / per
					ui.Row(c.Key(side.label + ".input")).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						if ui.NumberInput(c, &v, 0, hi, step).Label(side.label + " margin").Changed() {
							*side.v = v * per
							changed = true
						}
						ui.Text(c, name).TextColor(t.TextMuted)
					})
				}
			})
		})
		ui.Field(c, "Scale", func() {
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
				pct := math.Round(setup.Scale * 100)
				if ui.NumberInput(c, &pct, 10, 400, 5).Label("Scale").Changed() {
					setup.Scale = pct / 100
					changed = true
				}
				ui.Text(c, "%").TextColor(t.TextMuted)
			})
		})
	})
	return changed
}

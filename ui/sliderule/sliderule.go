// Package sliderule is a slide rule: K and A on the upper stock, B, CI
// and C on the slide, D and L on the lower stock, all as long as the
// rule is wide. Dragging the slide draws it out to either side; pressing
// the stock puts the cursor's hairline there and dragging moves it. What
// each scale reads under the hairline shows below the rule, so that with
// C's 1 over 2 on D, the hairline over 3 on C reads 6 on D.
//
//	app.rule = sliderule.State{Slide: math.Log10(2), Cursor: math.Log10(6)}
//	sliderule.View(c, &app.rule)
package sliderule

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// State is what a slide rule keeps: where its slide and cursor are, and
// the drag. Keep it in the app's state, one for each rule.
type State struct {
	// Slide is how far the slide is drawn out, as a fraction of the
	// rule: C's 1 sits over D's 10^Slide. It is from -1 to 1.
	Slide float64
	// Cursor is where the hairline is, as a fraction of the rule: it
	// reads 10^Cursor on D.
	Cursor float64

	drag  int
	from  float64
	fromX float32
}

const (
	dragSlide = iota + 1
	dragCursor
)

// scale is one scale of the rule: its name, where along it a value
// sits, the ticks it has, and how its ticks hang.
type scale struct {
	name  string
	value func(f float64) float64
	ticks []tick
	top   bool
	red   bool
	plain bool
}

// tick is a mark of a scale: where it is as a fraction of the rule, its
// size from 0, the longest, and its label.
type tick struct {
	at    float64
	size  int
	label string
}

// logTicks returns the ticks of a scale of decades decades, finer as
// length leaves room for them.
func logTicks(decades int, length float32) []tick {
	var out []tick
	span := float64(length) / float64(decades)
	for d := range decades {
		base := math.Pow(10, float64(d))
		for i := 1; i <= 9; i++ {
			label := fmt.Sprint(i)
			switch {
			case i == 1:
				label = fmt.Sprint(base)
			case span*math.Log10(float64(i+1)/float64(i)) < 16:
				label = ""
			}
			at := func(v float64) float64 { return (float64(d) + math.Log10(v)) / float64(decades) }
			out = append(out, tick{at(float64(i)), 0, label})
			n := 0
			for _, k := range []int{50, 20, 10, 5, 2} {
				if span*math.Log10(float64(i+1)/(float64(i+1)-1/float64(k))) >= 3.5 {
					n = k
					break
				}
			}
			for k := 1; k < n; k++ {
				size := 2
				if k*2 == n || n > 10 && k%(n/10) == 0 {
					size = 1
				}
				out = append(out, tick{at(float64(i) + float64(k)/float64(n)), size, ""})
			}
		}
	}
	return append(out, tick{1, 0, fmt.Sprint(math.Pow(10, float64(decades)))})
}

func linearTicks(length float32) []tick {
	n := 100
	if length < 400 {
		n = 50
	}
	out := make([]tick, 0, n+1)
	for k := 0; k <= n; k++ {
		t := tick{float64(k) / float64(n), 2, ""}
		switch {
		case k%(n/10) == 0:
			t.size, t.label = 0, fmt.Sprint(k/(n/10))
		case k%(n/20) == 0:
			t.size = 1
		}
		out = append(out, t)
	}
	return out
}

func reversed(ts []tick) []tick {
	out := make([]tick, len(ts))
	for i, t := range ts {
		t.at = 1 - t.at
		out[i] = t
	}
	return out
}

func format(v float64) string {
	switch {
	case v < 10:
		return fmt.Sprintf("%.3f", v)
	case v < 100:
		return fmt.Sprintf("%.2f", v)
	case v < 1000:
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.0f", v)
}

// View creates the slide rule.
func View(c *ui.Context, s *State) ui.Element {
	t := c.Theme()
	row := t.Space(7)
	name := t.Space(8)
	pad := t.Space(4)
	groove := float32(2)
	height := row*7 + groove*2 + pad*2
	root := ui.Column(c).Gap(t.Space(5)).AlignSelf(ui.Stretch).AlignItems(ui.Stretch)
	root.Children(func() {
		rule := ui.Box(c).Height(height).Label("Slide rule").Cursor(ui.CursorGrab)
		length := max(rule.LayoutBounds().W-name-t.Space(6), 100)
		pow := func(n float64) func(float64) float64 {
			return func(f float64) float64 { return math.Pow(10, n*f) }
		}
		upper := []scale{
			{name: "K", value: pow(3), ticks: logTicks(3, length)},
			{name: "A", value: pow(2), ticks: logTicks(2, length)},
		}
		slide := []scale{
			{name: "B", value: pow(2), ticks: logTicks(2, length), top: true},
			{name: "CI", value: func(f float64) float64 { return math.Pow(10, 1-f) }, ticks: reversed(logTicks(1, length)), top: true, red: true},
			{name: "C", value: pow(1), ticks: logTicks(1, length)},
		}
		lower := []scale{
			{name: "D", value: pow(1), ticks: logTicks(1, length), top: true},
			{name: "L", value: func(f float64) float64 { return f }, ticks: linearTicks(length), top: true, plain: true},
		}
		x, y, _ := rule.PointerPosition()
		if _, _, held := rule.Dragged(); held {
			slideTop, slideBottom := pad+row*2+groove, pad+row*5+groove
			if s.drag == 0 {
				s.fromX = x
				if y >= slideTop && y < slideBottom {
					s.drag, s.from = dragSlide, s.Slide
				} else {
					s.drag, s.from = dragCursor, float64((x-name)/length)
				}
			}
			moved := float64((x - s.fromX) / length)
			if s.drag == dragSlide {
				s.Slide = min(max(s.from+moved, -1), 1)
			} else {
				s.Cursor = min(max(s.from+moved, 0), 1)
			}
		} else {
			s.drag = 0
		}
		slideAt, cursorAt := s.Slide, s.Cursor
		rule.Draw(func(p *ui.Painter, b ui.Rect) {
			body := ui.Rect{X: b.X + name, Y: b.Y + pad, W: b.W - name - t.Space(6), H: b.H - pad*2}
			stock := t.Surface
			face := t.Background
			if t.Dark {
				face = ui.Hex("#2A2A2E")
			}
			p.Fill(ui.Rect{X: b.X, Y: body.Y, W: b.W, H: row * 2}, stock, t.Radius)
			p.Fill(ui.Rect{X: b.X, Y: body.Y + row*5 + groove*2, W: b.W, H: row * 2}, stock, t.Radius)
			p.Stroke(ui.Rect{X: b.X, Y: body.Y, W: b.W, H: row * 2}, t.Border, t.Radius, 1)
			p.Stroke(ui.Rect{X: b.X, Y: body.Y + row*5 + groove*2, W: b.W, H: row * 2}, t.Border, t.Radius, 1)
			draw := func(sc scale, y, dx float32, ink ui.Color) {
				for _, tk := range sc.ticks {
					x := body.X + dx + float32(tk.at)*body.W
					if x < b.X || x > b.X+b.W {
						continue
					}
					l := row * []float32{0.42, 0.3, 0.18}[tk.size]
					y0, y1 := y+row, y+row-l
					if sc.top {
						y0, y1 = y, y+l
					}
					p.Line(x, y0, x, y1, 1, ink)
					if tk.label != "" {
						fs := t.FontSize - 3
						w, h := p.MeasureText(0, ui.Span{Text: tk.label, Size: fs})
						ly := y1 - h - 1
						if sc.top {
							ly = y1 + 1
						}
						p.Text(x-w/2, ly, tk.label, fs, ink)
					}
				}
			}
			label := func(sc scale, x, y float32, ink ui.Color) {
				w, h := p.MeasureText(0, ui.Span{Text: sc.name, Size: t.FontSize - 2})
				p.Text(x-w/2, y+row/2-h/2, sc.name, t.FontSize-2, ink)
			}
			for i, sc := range upper {
				y := body.Y + row*float32(i)
				draw(sc, y, 0, t.Text)
				label(sc, b.X+name/2, y, t.TextMuted)
			}
			for i, sc := range lower {
				y := body.Y + row*float32(5+i) + groove*2
				draw(sc, y, 0, t.Text)
				label(sc, b.X+name/2, y, t.TextMuted)
			}
			dx := float32(slideAt) * body.W
			slideRect := ui.Rect{X: b.X + dx, Y: body.Y + row*2 + groove, W: b.W, H: row * 3}
			p.Clip(b, 0, func() {
				p.Shadow(slideRect, 2, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.15))
				p.Fill(slideRect, face, 2)
				p.Stroke(slideRect, t.Border, 2, 1)
				for i, sc := range slide {
					ink := t.Text
					if sc.red {
						ink = t.Danger
					}
					y := slideRect.Y + row*float32(i)
					draw(sc, y, dx, ink)
					label(sc, slideRect.X+name/2, y, ink.Alpha(0.7))
				}
			})
			hx := body.X + float32(cursorAt)*body.W
			win := ui.Rect{X: hx - t.Space(5), Y: b.Y, W: t.Space(10), H: b.H}
			p.Fill(win, t.Accent.Alpha(0.08), 3)
			p.Stroke(win, t.Accent.Alpha(0.5), 3, 1)
			p.Line(hx, b.Y, hx, b.Y+b.H, 1, t.Danger)
		})
		ui.Row(c).Wrap().GapX(t.Space(6)).GapY(t.Space(2)).Children(func() {
			read := func(sc scale, f float64, ok bool) {
				ui.Row(c.Key(sc.name)).Gap(t.Space(1.5)).AlignItems(ui.Baseline).Children(func() {
					ui.Text(c, sc.name).TextColor(t.TextMuted)
					v := "–"
					if ok {
						v = format(sc.value(f))
						if sc.plain {
							v = fmt.Sprintf("%.3f", f)
						}
					}
					ui.Text(c, v).Bold().FontFeatures("tnum").Width(t.Space(14))
				})
			}
			for _, sc := range upper {
				read(sc, s.Cursor, true)
			}
			u := s.Cursor - s.Slide
			for _, sc := range slide {
				read(sc, u, u >= 0 && u <= 1)
			}
			for _, sc := range lower {
				read(sc, s.Cursor, true)
			}
		})
	})
	return root
}

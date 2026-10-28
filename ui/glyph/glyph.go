// Package glyph browses the glyphs of a font, as Font Book's repertoire
// and Glyphs' font view do: every glyph in a grid with the code point it
// draws, and the one picked drawn large between the font's ascender, cap
// height, x-height, baseline and descender, with its advance boxed and
// its name, code point, index and advance below. Points shows its
// outline with the points on it as squares and the control points off it
// as circles. The search finds a glyph by the character itself, its code
// point as U+00E9 or 00e9, or a part of its name.
//
//	f, _ := sfnt.Parse(goregular.TTF)
//	glyph.View(c, &app.glyphs, f)
package glyph

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// State is what a browser keeps: the glyph picked, the search, whether
// points show, and what it read of its font. Keep it in the app's state,
// one for each browser.
type State struct {
	picked   bool
	selected sfnt.GlyphIndex
	query    string
	points   bool

	font    *sfnt.Font
	buf     sfnt.Buffer
	family  string
	glyphs  []info
	metrics font.Metrics
}

type info struct {
	index   sfnt.GlyphIndex
	r       rune
	name    string
	advance float32
	outline sfnt.Segments
}

// read reads every glyph of f, in font units, and the characters they
// draw, which a font maps only from character to glyph.
func (s *State) read(f *sfnt.Font) {
	s.font, s.glyphs = f, nil
	ppem := fixed.I(int(f.UnitsPerEm()))
	s.family, _ = f.Name(&s.buf, sfnt.NameIDFamily)
	s.metrics, _ = f.Metrics(&s.buf, ppem, font.HintingNone)
	runes := make([]rune, f.NumGlyphs())
	for r := rune(0x20); r <= 0xFFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		if x, err := f.GlyphIndex(&s.buf, r); err == nil && x != 0 && runes[x] == 0 {
			runes[x] = r
		}
	}
	for i := range f.NumGlyphs() {
		x := sfnt.GlyphIndex(i)
		g := info{index: x, r: runes[i]}
		g.name, _ = f.GlyphName(&s.buf, x)
		if adv, err := f.GlyphAdvance(&s.buf, x, ppem, font.HintingNone); err == nil {
			g.advance = units(adv)
		}
		if segs, err := f.LoadGlyph(&s.buf, x, ppem, nil); err == nil {
			g.outline = slices.Clone(segs)
		}
		s.glyphs = append(s.glyphs, g)
	}
	if x, err := f.GlyphIndex(&s.buf, 'g'); err == nil && !s.picked {
		s.selected = x
	}
}

func units(v fixed.Int26_6) float32 { return float32(v) / 64 }

func (g info) codePoint() string {
	if g.r == 0 {
		return ""
	}
	return fmt.Sprintf("U+%04X", g.r)
}

// matches reports whether g is what query looks for.
func (g info) matches(query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return true
	}
	if utf8.RuneCountInString(q) == 1 {
		r, _ := utf8.DecodeRuneInString(q)
		if g.r == r {
			return true
		}
	}
	hex := strings.TrimPrefix(strings.TrimPrefix(strings.ToUpper(q), "U+"), "0X")
	if n, err := strconv.ParseUint(hex, 16, 32); err == nil && len(hex) >= 2 && rune(n) == g.r {
		return true
	}
	return strings.Contains(strings.ToLower(g.name), strings.ToLower(q))
}

// path returns g's outline scaled by k with its origin at x, y.
func (g info) path(x, y, k float32) *ui.Path {
	var p ui.Path
	pt := func(a fixed.Point26_6) (float32, float32) { return x + units(a.X)*k, y + units(a.Y)*k }
	for _, seg := range g.outline {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			p.MoveTo(pt(seg.Args[0]))
		case sfnt.SegmentOpLineTo:
			p.LineTo(pt(seg.Args[0]))
		case sfnt.SegmentOpQuadTo:
			cx, cy := pt(seg.Args[0])
			ex, ey := pt(seg.Args[1])
			p.QuadTo(cx, cy, ex, ey)
		case sfnt.SegmentOpCubeTo:
			c1x, c1y := pt(seg.Args[0])
			c2x, c2y := pt(seg.Args[1])
			ex, ey := pt(seg.Args[2])
			p.CubeTo(c1x, c1y, c2x, c2y, ex, ey)
		}
	}
	return &p
}

// View creates the browser of f.
func View(c *ui.Context, s *State, f *sfnt.Font) ui.Element {
	t := c.Theme()
	if s.font != f {
		s.read(f)
	}
	var shown []info
	for _, g := range s.glyphs {
		if g.matches(s.query) {
			shown = append(shown, g)
		}
	}
	m := s.metrics
	em := units(m.Ascent + m.Descent)
	root := ui.Column(c).Gap(t.Space(4)).Fill()
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, s.family).FontSize(t.FontSize + 4).Bold()
			ui.Text(c, fmt.Sprintf("%d glyphs", len(s.glyphs))).TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.SearchField(c, &s.query).Width(t.Space(60))
		})
		ui.Row(c).Gap(t.Space(5)).Grow(1).Basis(0).AlignItems(ui.Stretch).Children(func() {
			ui.Scroll(c).Grow(1).Basis(0).Children(func() {
				ui.Row(c).Wrap().Gap(t.Space(1)).Children(func() {
					for _, g := range shown {
						s.cell(c, g, em)
					}
				})
				if len(shown) == 0 {
					ui.Text(c, "No glyph matches").TextColor(t.TextMuted).Padding(t.Space(4))
				}
			})
			if int(s.selected) < len(s.glyphs) {
				s.detail(c, s.glyphs[s.selected])
			}
		})
	})
	return root
}

func (s *State) cell(c *ui.Context, g info, em float32) {
	t := c.Theme()
	selected := g.index == s.selected
	b := ui.ButtonBase(c.Key(g.index)).Column().Size(t.Space(14), t.Space(17)).Radius(t.Radius).Label(g.name).Tooltip(strings.TrimSpace(g.name + "  " + g.codePoint()))
	switch {
	case selected:
		b.Background(t.Accent)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	default:
		b.Background(t.Surface)
	}
	if b.Clicked() {
		s.selected, s.picked = g.index, true
	}
	ink, muted := t.Text, t.TextMuted
	if selected {
		ink, muted = t.AccentText, t.AccentText.Alpha(0.75)
	}
	ascent := units(s.metrics.Ascent)
	b.Children(func() {
		ui.Box(c).FillWidth().Grow(1).Draw(func(p *ui.Painter, r ui.Rect) {
			k := (r.H - 6) / em
			p.FillPath(g.path(r.X+(r.W-g.advance*k)/2, r.Y+3+ascent*k, k), ink)
		})
		label := strings.TrimPrefix(g.codePoint(), "U+")
		if label == "" {
			label = g.name
		}
		ui.Text(c, label).Font("monospace").FontSize(t.FontSize-4).TextColor(muted).SingleLine().Ellipsis("…").AlignSelf(ui.Center).Padding(0, t.Space(1), t.Space(1))
	})
}

var guides = [...]string{"Ascender", "Cap height", "x-height", "Baseline", "Descender"}

func (s *State) detail(c *ui.Context, g info) {
	t := c.Theme()
	m := s.metrics
	heights := [...]float32{units(m.Ascent), units(m.CapHeight), units(m.XHeight), 0, -units(m.Descent)}
	points := s.points
	ui.Column(c).Width(t.Space(96)).Gap(t.Space(3)).Children(func() {
		ui.Box(c).FillWidth().Height(t.Space(96)).Radius(t.Radius).Background(t.Surface).Label("Glyph " + g.name).Draw(func(p *ui.Painter, r ui.Rect) {
			pad := t.Space(8)
			k := (r.H - 2*pad) / (heights[0] - heights[4])
			base := r.Y + pad + heights[0]*k
			x0 := r.X + (r.W-g.advance*k)/2
			for i, h := range heights {
				y := base - h*k
				col := t.Text.Alpha(0.15)
				if i == 3 {
					col = t.Text.Alpha(0.4)
				}
				p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: 1}, col, 0)
				p.Text(r.X+t.Space(2), y-t.Space(4), guides[i], t.FontSize-3, t.TextMuted)
			}
			box := ui.Rect{X: x0, Y: base - heights[0]*k, W: g.advance * k, H: (heights[0] - heights[4]) * k}
			p.Fill(ui.Rect{X: box.X, Y: box.Y, W: 1, H: box.H}, t.Accent.Alpha(0.5), 0)
			p.Fill(ui.Rect{X: box.X + box.W, Y: box.Y, W: 1, H: box.H}, t.Accent.Alpha(0.5), 0)
			outline := g.path(x0, base, k)
			if !points {
				p.FillPath(outline, t.Text)
				return
			}
			p.FillPath(outline, t.Text.Alpha(0.08))
			p.StrokePath(outline, 1.5, t.Accent)
			drawPoints(p, g.outline, x0, base, k, t)
		})
		ui.Checkbox(c, &s.points, "Points")
		ui.Grid(c).ColumnTracks(ui.Fixed(t.Space(20)), ui.Fr(1)).GapX(t.Space(3)).GapY(t.Space(1.5)).Children(func() {
			row := func(label, value string) {
				ui.Text(c.Key(label), label).TextColor(t.TextMuted)
				ui.Text(c.Key(label+"/value"), value).Font("monospace").Selectable()
			}
			row("Name", g.name)
			if g.r != 0 {
				row("Unicode", g.codePoint()+"  "+string(g.r))
			}
			row("Index", strconv.Itoa(int(g.index)))
			row("Advance", fmt.Sprintf("%g units", g.advance))
		})
	})
}

// drawPoints draws the points of outline: those on it as squares, the
// control points off it as circles tied to the points they pull.
func drawPoints(p *ui.Painter, outline sfnt.Segments, x, y, k float32, t *ui.Theme) {
	pt := func(a fixed.Point26_6) (float32, float32) { return x + units(a.X)*k, y + units(a.Y)*k }
	var on [][2]float32
	var off [][2]float32
	var px, py float32
	handle := func(ax, ay, bx, by float32) {
		var h ui.Path
		h.MoveTo(ax, ay).LineTo(bx, by)
		p.StrokePath(&h, 1, t.TextMuted)
	}
	for _, seg := range outline {
		n := map[sfnt.SegmentOp]int{sfnt.SegmentOpMoveTo: 1, sfnt.SegmentOpLineTo: 1, sfnt.SegmentOpQuadTo: 2, sfnt.SegmentOpCubeTo: 3}[seg.Op]
		ex, ey := pt(seg.Args[n-1])
		switch n {
		case 2:
			cx, cy := pt(seg.Args[0])
			handle(px, py, cx, cy)
			handle(cx, cy, ex, ey)
			off = append(off, [2]float32{cx, cy})
		case 3:
			c1x, c1y := pt(seg.Args[0])
			c2x, c2y := pt(seg.Args[1])
			handle(px, py, c1x, c1y)
			handle(c2x, c2y, ex, ey)
			off = append(off, [2]float32{c1x, c1y}, [2]float32{c2x, c2y})
		}
		on = append(on, [2]float32{ex, ey})
		px, py = ex, ey
	}
	for _, o := range off {
		var dot ui.Path
		dot.Circle(o[0], o[1], 3)
		p.FillPath(&dot, t.Surface)
		p.StrokePath(&dot, 1.5, t.Accent)
	}
	for _, o := range on {
		p.Fill(ui.Rect{X: o[0] - 3, Y: o[1] - 3, W: 6, H: 6}, t.Accent, 0)
	}
}

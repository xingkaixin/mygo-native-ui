// Package embeddings explores embeddings laid out in two dimensions, as
// TensorFlow's Embedding Projector does after PCA, t-SNE or UMAP: a dot
// for each item in the color of its group, a search that rings the items
// whose label has the words and dims the rest, and a chip for each group
// with its count, which hides or shows it. The pointer on a dot names it
// and draws lines to its nearest neighbors, by cosine similarity of the
// full vectors when they are given, or by distance on the plot; a click
// keeps it chosen, and the panel beside the plot lists its neighbors with
// their similarity, a click on one choosing it in turn. Escape, or a
// click on nothing, lets it go.
//
//	embeddings.View(c, &app.projector, app.points, 8).Grow(1)
package embeddings

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Point is an item: its label, its group, where the layout put it, and
// its vector, for neighbors by cosine similarity, or nil.
type Point struct {
	Label, Group string
	X, Y         float64
	Vec          []float32
}

// State is what a projector keeps: the search, the groups hidden and the
// item chosen. Keep it in the app's state, one for each projector.
type State struct {
	query  string
	hidden map[string]bool
	// chosen is the index of the item chosen, plus 1.
	chosen int
}

// Color returns the color of the group at index i of the groups in the
// order they first come.
func Color(t *ui.Theme, i int) ui.Color {
	l := float32(0.62)
	if t.Dark {
		l = 0.74
	}
	return ui.Oklch(l, 0.15, float32(i)*137.5+30)
}

// neighbor is an item near another, and how similar.
type neighbor struct {
	i   int
	sim float64
}

// View creates the plot of pts beside its panel, with k neighbors for
// the item under the pointer or chosen. Give it a size, or grow it in
// its parent.
func View(c *ui.Context, s *State, pts []Point, k int) ui.Element {
	t := c.Theme()
	if s.hidden == nil {
		s.hidden = map[string]bool{}
	}
	if s.chosen > len(pts) {
		s.chosen = 0
	}
	var groups []string
	index := map[string]int{}
	for _, p := range pts {
		if _, ok := index[p.Group]; !ok {
			index[p.Group] = len(groups)
			groups = append(groups, p.Group)
		}
	}
	words := strings.Fields(strings.ToLower(s.query))
	match := func(p Point) bool {
		if len(words) == 0 {
			return false
		}
		l := strings.ToLower(p.Label)
		return !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(l, w) })
	}
	return ui.Row(c).AlignItems(ui.Stretch).Gap(t.Space(4)).Children(func() {
		hot := plot(c, s, pts, index, match, len(words) > 0, k)
		ui.Column(c).Width(t.Space(64)).Gap(t.Space(4)).Children(func() {
			ui.SearchField(c, &s.query).FillWidth().Label("Search labels")
			ui.Row(c).Wrap().Gap(t.Space(1.5)).Children(func() {
				for gi, g := range groups {
					chip(c.Key(g), s, pts, g, Color(t, gi))
				}
			})
			focus := s.chosen - 1
			if focus < 0 {
				focus = hot
			}
			if focus >= 0 {
				panel(c, s, pts, focus, index, k)
			}
		})
	})
}

// plot creates the plot and returns the index of the item under the
// pointer, or -1.
func plot(c *ui.Context, s *State, pts []Point, index map[string]int, match func(Point) bool, searching bool, k int) int {
	t := c.Theme()
	box := ui.Box(c).Grow(1).MinWidth(t.Space(80)).MinHeight(t.Space(80)).Radius(t.Radius+2).
		Border(1, t.Border).Background(t.Surface).Clip().Focusable().FocusRing(false).Role(ui.RoleImage).Label("Embeddings")
	b := box.LayoutBounds()
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, p := range pts {
		lo = [2]float64{min(lo[0], p.X), min(lo[1], p.Y)}
		hi = [2]float64{max(hi[0], p.X), max(hi[1], p.Y)}
	}
	pad := t.Space(8)
	span := max(hi[0]-lo[0], hi[1]-lo[1], 1e-9)
	scale := float64(min(b.W, b.H)-2*pad) / span
	ox := float64(b.W)/2 - (lo[0]+hi[0])/2*scale
	oy := float64(b.H)/2 + (lo[1]+hi[1])/2*scale
	at := func(p Point) (float32, float32) { return float32(ox + p.X*scale), float32(oy - p.Y*scale) }

	hot := -1
	if px, py, over := box.PointerPosition(); over {
		best := t.Space(3)
		for i, p := range pts {
			if s.hidden[p.Group] {
				continue
			}
			x, y := at(p)
			if d := float32(math.Hypot(float64(x-px), float64(y-py))); d < best {
				hot, best = i, d
			}
		}
	}
	if hot >= 0 {
		box.Cursor(ui.CursorPointer)
	}
	if box.Clicked() {
		s.chosen = hot + 1
	}
	if box.Shortcut(0, ui.KeyEscape) {
		s.chosen = 0
	}
	focus := s.chosen - 1
	if hot >= 0 {
		focus = hot
	}
	var near []neighbor
	if focus >= 0 {
		near = neighbors(s, pts, focus, k)
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		pos := func(i int) (float32, float32) { x, y := at(pts[i]); return r.X + x, r.Y + y }
		if focus >= 0 {
			fx, fy := pos(focus)
			for _, n := range near {
				x, y := pos(n.i)
				p.StrokePolyline([][2]float32{{fx, fy}, {x, y}}, 1.25, t.Text.Alpha(0.35))
			}
		}
		dot := t.Space(1.1)
		for i, pt := range pts {
			if s.hidden[pt.Group] {
				continue
			}
			col := Color(t, index[pt.Group])
			if searching && !match(pt) || focus >= 0 && i != focus && !slices.ContainsFunc(near, func(n neighbor) bool { return n.i == i }) {
				col = col.Alpha(0.25)
			}
			x, y := pos(i)
			p.Fill(ui.Rect{X: x - dot, Y: y - dot, W: 2 * dot, H: 2 * dot}, col, dot)
			if searching && match(pt) {
				d := dot + 2.5
				p.Stroke(ui.Rect{X: x - d, Y: y - d, W: 2 * d, H: 2 * d}, t.Text, d, 1.25)
			}
		}
		label := func(i int, bold bool) {
			x, y := pos(i)
			sp := ui.Span{Text: pts[i].Label, Size: t.FontSize - 1, Color: t.Text}
			if bold {
				sp.Weight = 700
			}
			w, h := p.MeasureText(0, sp)
			bx := ui.Rect{X: x + t.Space(2), Y: y - h/2 - 1, W: w + t.Space(2), H: h + 2}
			if bx.X+bx.W > r.X+r.W {
				bx.X = x - t.Space(2) - bx.W
			}
			p.Fill(bx, t.Background.Alpha(0.85), t.Radius)
			p.RichText(bx.X+t.Space(1), bx.Y+1, 0, sp)
		}
		if focus >= 0 {
			for _, n := range near {
				label(n.i, false)
			}
			x, y := pos(focus)
			d := dot + 3
			p.Fill(ui.Rect{X: x - d, Y: y - d, W: 2 * d, H: 2 * d}, Color(t, index[pts[focus].Group]), d)
			p.Stroke(ui.Rect{X: x - d, Y: y - d, W: 2 * d, H: 2 * d}, t.Background, d, 2)
			label(focus, true)
		} else if searching {
			for i, pt := range pts {
				if match(pt) && !s.hidden[pt.Group] {
					label(i, false)
				}
			}
		}
	})
	return hot
}

// chip creates the chip of group g, which hides or shows it.
func chip(c *ui.Context, s *State, pts []Point, g string, col ui.Color) {
	t := c.Theme()
	n := 0
	for _, p := range pts {
		if p.Group == g {
			n++
		}
	}
	on := !s.hidden[g]
	b := ui.ButtonBase(c).Row().Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(2)).
		Radius(t.Space(3)).Border(1, t.Border).Checked(on).Label(g)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		s.hidden[g] = on
	}
	b.Children(func() {
		dot := ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1))
		if on {
			dot.Background(col)
		} else {
			dot.Border(1.5, col)
		}
		txt := ui.Text(c, g).FontSize(t.FontSize - 2)
		if !on {
			txt.TextColor(t.TextMuted)
		}
		ui.Text(c, strconv.Itoa(n)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum")
	})
}

// panel creates the item at focus, its group, and its neighbors, each of
// which a click chooses.
func panel(c *ui.Context, s *State, pts []Point, focus int, index map[string]int, k int) {
	t := c.Theme()
	p := pts[focus]
	ui.Column(c).Gap(t.Space(0.5)).Children(func() {
		ui.Text(c, p.Label).Bold().FontSize(t.FontSize + 2)
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(Color(t, index[p.Group]))
			ui.Text(c, p.Group).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
	})
	by := "Nearest on the plot"
	if len(p.Vec) > 0 {
		by = "Nearest by cosine similarity"
	}
	ui.Text(c, by).FontSize(t.FontSize - 2).FontWeight(600).TextColor(t.TextMuted)
	ui.Column(c).Children(func() {
		for _, n := range neighbors(s, pts, focus, k) {
			q := pts[n.i]
			b := ui.ButtonBase(c.Key(n.i)).Row().Gap(t.Space(2)).AlignItems(ui.Center).
				Padding(t.Space(1), t.Space(1.5)).Radius(t.Radius).Label(q.Label)
			if b.Hovered() {
				b.Background(t.Text.Alpha(0.06))
			}
			if b.Clicked() {
				s.chosen = n.i + 1
			}
			b.Children(func() {
				ui.Box(c).Size(t.Space(1.5), t.Space(1.5)).Radius(t.Space(0.75)).Background(Color(t, index[q.Group]))
				ui.Text(c, q.Label).Grow(1).Shrink(1).SingleLine()
				sim := fmt.Sprintf("%.3f", n.sim)
				if len(p.Vec) == 0 {
					sim = fmt.Sprintf("%.2f", -n.sim)
				}
				ui.Text(c, sim).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
			})
		}
	})
}

// neighbors returns the k items nearest to item i among those shown, the
// nearest first: by cosine similarity of their vectors, or else by the
// distance between them on the plot, negated.
func neighbors(s *State, pts []Point, i, k int) []neighbor {
	a := pts[i]
	var out []neighbor
	for j, b := range pts {
		if j == i || s.hidden[b.Group] {
			continue
		}
		sim := -math.Hypot(a.X-b.X, a.Y-b.Y)
		if len(a.Vec) > 0 && len(a.Vec) == len(b.Vec) {
			sim = cosine(a.Vec, b.Vec)
		}
		out = append(out, neighbor{j, sim})
	}
	slices.SortFunc(out, func(x, y neighbor) int { return cmp.Compare(y.sim, x.sim) })
	return out[:min(k, len(out))]
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}

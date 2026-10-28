// Package arcdiagram draws a network as an arc diagram: its nodes in a
// row along a line, each a dot as large as its number of links and
// colored by its group, its name below it, and each link an arc above the
// line between its two nodes, as wide as its weight. Links within a group
// take the group's color. Hovering a node lights its arcs and the names of
// the nodes it links to, and reads out how many links go out and come in.
//
//	arcdiagram.View(c, &app.arcs, nodes, links)
//
// The nodes keep the order they are given in; ordering them by group
// keeps most arcs short.
package arcdiagram

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Node is a node of the network.
type Node struct {
	Name, Group string
}

// Link links node From to node To, indexes into the nodes.
type Link struct {
	From, To int
	// Weight is how strong the link is; zero counts as one.
	Weight float64
}

// State is what a diagram keeps: the node under the pointer and the room
// its names take. Keep it in the app's state, one for each diagram.
type State struct {
	hover  int
	labels float32
}

var palette = []ui.Color{
	ui.RGB(66, 133, 244), ui.RGB(234, 134, 46), ui.RGB(52, 168, 83), ui.RGB(217, 72, 92),
	ui.RGB(142, 92, 214), ui.RGB(23, 162, 184), ui.RGB(196, 160, 30), ui.RGB(120, 120, 128),
}

// View creates the diagram of nodes and links.
func View(c *ui.Context, s *State, nodes []Node, links []Link) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Clip().Role(ui.RoleImage).Label(fmt.Sprintf("Arc diagram of %d nodes and %d links", len(nodes), len(links)))
	if len(nodes) == 0 {
		return view
	}
	colors := map[string]ui.Color{}
	var groups []string
	for _, n := range nodes {
		if _, ok := colors[n.Group]; !ok {
			colors[n.Group] = palette[len(groups)%len(palette)]
			groups = append(groups, n.Group)
		}
	}
	out, in := make([]int, len(nodes)), make([]int, len(nodes))
	heaviest := 0.0
	for _, l := range links {
		out[l.From]++
		in[l.To]++
		heaviest = max(heaviest, weight(l))
	}
	b := view.LayoutBounds()
	pad := t.Space(4)
	step := (b.W - 2*pad) / float32(max(1, len(nodes)-1))
	xOf := func(i int) float32 { return pad + float32(i)*step }
	base := b.H - s.labels - t.Space(2)
	hover := -1
	if px, py, ok := view.PointerPosition(); ok && py > base-t.Space(6) {
		i := int(math.Round(float64((px - pad) / step)))
		if i >= 0 && i < len(nodes) {
			hover = i
		}
	}
	if hover != s.hover-1 {
		s.hover = hover + 1
		c.Invalidate()
	}
	linked := make([]bool, len(nodes))
	if hover >= 0 {
		for _, l := range links {
			if l.From == hover {
				linked[l.To] = true
			}
			if l.To == hover {
				linked[l.From] = true
			}
		}
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		small := t.FontSize - 2
		widest := float32(0)
		for _, n := range nodes {
			w, _ := p.MeasureText(0, ui.Span{Text: n.Name, Size: small})
			widest = max(widest, w)
		}
		s.labels = widest + t.Space(3)
		base := r.Y + r.H - s.labels - t.Space(2)
		top := r.Y + t.Space(10)
		longest := float32(0)
		for _, l := range links {
			longest = max(longest, float32(abs(l.To-l.From))*step)
		}
		// Arcs are half circles until the longest would rise past the top;
		// then all flatten alike.
		squash := min(1, (base-top)/max(1, longest/2))
		for pass := range 2 {
			for _, l := range links {
				on := hover >= 0 && (l.From == hover || l.To == hover)
				if (pass == 1) != on {
					continue
				}
				x0, x1 := r.X+xOf(l.From), r.X+xOf(l.To)
				rx := (x1 - x0) / 2
				ry := float32(math.Abs(float64(rx))) * squash
				pts := make([][2]float32, 0, 41)
				for k := range 41 {
					a := math.Pi * float64(k) / 40
					pts = append(pts, [2]float32{x0 + rx - rx*float32(math.Cos(a)), base - ry*float32(math.Sin(a))})
				}
				col := t.TextMuted.Alpha(0.35)
				if nodes[l.From].Group == nodes[l.To].Group {
					col = colors[nodes[l.From].Group].Alpha(0.55)
				}
				switch {
				case on:
					col = colors[nodes[hover].Group]
				case hover >= 0:
					col = col.Alpha(0.12)
				}
				p.StrokePolyline(pts, 1+2.5*float32(weight(l)/heaviest), col)
			}
		}
		p.StrokePolyline([][2]float32{{r.X + pad, base}, {r.X + r.W - pad, base}}, 1, t.Border)
		for i, n := range nodes {
			x := r.X + xOf(i)
			rad := 2.5 + 1.2*float32(math.Sqrt(float64(in[i]+out[i])))
			var dot ui.Path
			p.FillPath(dot.Circle(x, base, rad), colors[n.Group])
			col, weight := t.TextMuted, 400
			switch {
			case i == hover:
				col, weight = t.Text, 700
			case linked[i]:
				col = t.Text
			case hover >= 0:
				col = t.TextMuted.Alpha(0.5)
			}
			sp := ui.Span{Text: n.Name, Size: small, Color: col, Weight: weight}
			_, h := p.MeasureText(0, sp)
			p.Transform(ui.Rotation(90), x, base, func() {
				p.RichText(x+t.Space(3), base-h/2, 0, sp)
			})
		}
		x := r.X + pad
		for _, g := range groups {
			sp := ui.Span{Text: g, Size: small, Color: t.TextMuted}
			w, h := p.MeasureText(0, sp)
			sw := t.Space(2.5)
			p.Fill(ui.Rect{X: x, Y: r.Y + t.Space(3) + (h-sw)/2, W: sw, H: sw}, colors[g], sw/2)
			p.RichText(x+sw+t.Space(1.5), r.Y+t.Space(3), 0, sp)
			x += sw + t.Space(1.5) + w + t.Space(4)
		}
		if hover >= 0 {
			n := nodes[hover]
			spans := []ui.Span{
				{Text: n.Name + "  ", Size: t.FontSize - 1, Weight: 600, Color: t.Text},
				{Text: fmt.Sprintf("%d out · %d in", out[hover], in[hover]), Size: t.FontSize - 1, Color: t.TextMuted, Features: "tnum"},
			}
			w, _ := p.MeasureText(0, spans...)
			p.RichText(r.X+r.W-pad-w, r.Y+t.Space(3), 0, spans...)
		}
	})
	return view
}

func weight(l Link) float64 {
	if l.Weight == 0 {
		return 1
	}
	return l.Weight
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

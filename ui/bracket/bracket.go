// Package bracket draws a single-elimination bracket, as sports apps and
// tournament sites show one: a column for each round, each match level
// with the two it follows, lines joining them, and the champion at the
// end. Winners show bold with their scores, losers dimmed, a match being
// played marked Live, and one not played yet with its time.
//
//	ui.ScrollBoth(c).Grow(1).Children(func() {
//		if ev := bracket.View(c, app.rounds); ev.Round >= 0 {
//			app.show(ev.Round, ev.Match)
//		}
//	})
//
// The pointer over a team lights its way through the rounds, and a click
// on a match reports it.
package bracket

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Side is one of the two sides of a match: its name, or "" while not
// known, its seed, or 0, its score, and whether it won.
type Side struct {
	Name  string
	Seed  int
	Score string
	Won   bool
}

// Match is a match: its two sides, whether it is being played, and when
// it will be, shown while neither side has a score.
type Match struct {
	Top, Bottom Side
	Live        bool
	Time        string
}

// Round is a round: its name, as Quarterfinals, and its matches, those
// of the round before in twos feeding each.
type Round struct {
	Name    string
	Matches []Match
}

// Events is a bracket, and the match clicked this frame: its round and
// its place in the round, or -1 for none.
type Events struct {
	ui.Element
	Round, Match int
}

const (
	boxW, rowH = 184, 28
	gapX, gapY = 48, 20
	head       = 28
)

// place is where a match is drawn.
type place struct{ x, y float32 }

func layout(rounds []Round) [][]place {
	out := make([][]place, len(rounds))
	for r, rd := range rounds {
		out[r] = make([]place, len(rd.Matches))
		for m := range rd.Matches {
			x := float32(r) * (boxW + gapX)
			y := head + float32(m)*(2*rowH+gapY)
			if r > 0 && 2*m+1 < len(out[r-1]) {
				y = (out[r-1][2*m].y + out[r-1][2*m+1].y) / 2
			}
			out[r][m] = place{x, y}
		}
	}
	return out
}

// View creates the bracket of rounds, as large as it is drawn.
func View(c *ui.Context, rounds []Round) Events {
	t := c.Theme()
	ev := Events{Round: -1, Match: -1}
	places := layout(rounds)
	var w, h float32
	for _, col := range places {
		for _, p := range col {
			w, h = max(w, p.x+boxW), max(h, p.y+2*rowH)
		}
	}
	champion := ""
	if n := len(rounds); n > 0 && len(rounds[n-1].Matches) == 1 {
		final := rounds[n-1].Matches[0]
		for _, s := range []Side{final.Top, final.Bottom} {
			if s.Won {
				champion = s.Name
			}
		}
		if champion != "" {
			w += gapX + boxW
		}
	}
	box := ui.Box(c).Size(w, h).Label("Bracket")

	// at returns the match and the side under the pointer, if any.
	at := func(x, y float32) (r, m int, side *Side) {
		for r, col := range places {
			for m, p := range col {
				if x >= p.x && x < p.x+boxW && y >= p.y && y < p.y+2*rowH {
					mt := &rounds[r].Matches[m]
					if y < p.y+rowH {
						return r, m, &mt.Top
					}
					return r, m, &mt.Bottom
				}
			}
		}
		return -1, -1, nil
	}
	lit := ""
	hr, hm := -1, -1
	if x, y, ok := box.PointerPosition(); ok {
		var s *Side
		hr, hm, s = at(x, y)
		if s != nil {
			lit = s.Name
			box.Cursor(ui.CursorPointer)
		}
	}
	if box.Clicked() && hr >= 0 {
		ev.Round, ev.Match = hr, hm
	}
	ev.Element = box
	in := func(mt Match) bool { return lit != "" && (mt.Top.Name == lit || mt.Bottom.Name == lit) }

	box.Draw(func(p *ui.Painter, rc ui.Rect) {
		ox, oy := rc.X, rc.Y
		for r, rd := range rounds {
			span := ui.Span{Text: rd.Name, Size: t.FontSize - 1, Weight: 600, Color: t.TextMuted}
			p.RichText(ox+places[r][0].x, oy, 0, span)
		}
		// Lines first, so the boxes cover their ends.
		for r := 1; r < len(rounds); r++ {
			for m := range rounds[r].Matches {
				to := places[r][m]
				for k, from := range []int{2 * m, 2*m + 1} {
					if from >= len(places[r-1]) {
						continue
					}
					f := places[r-1][from]
					color, width := t.Border, float32(1.5)
					if in(rounds[r-1].Matches[from]) && in(rounds[r].Matches[m]) {
						color, width = t.Accent, 2.5
					}
					ty := to.y + rowH/2 + float32(k)*rowH
					mid := f.x + boxW + gapX/2
					p.StrokePolyline([][2]float32{
						{ox + f.x + boxW, oy + f.y + rowH},
						{ox + mid, oy + f.y + rowH},
						{ox + mid, oy + ty},
						{ox + to.x, oy + ty},
					}, width, color)
				}
			}
		}
		for r, rd := range rounds {
			for m, mt := range rd.Matches {
				pl := places[r][m]
				frame := ui.Rect{X: ox + pl.x, Y: oy + pl.y, W: boxW, H: 2 * rowH}
				p.Shadow(frame, 6, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.08))
				p.Fill(frame, t.Background, 6)
				border, bw := t.Border, float32(1)
				switch {
				case in(mt):
					border, bw = t.Accent, 2
				case r == hr && m == hm:
					border = t.TextMuted
				}
				p.Line(frame.X, frame.Y+rowH, frame.X+boxW, frame.Y+rowH, 1, t.Border)
				p.Stroke(frame, border, 6, bw)
				played := mt.Top.Score != "" || mt.Bottom.Score != ""
				for k, s := range []Side{mt.Top, mt.Bottom} {
					side(p, t, s, frame.X, frame.Y+float32(k)*rowH, played, lit != "" && s.Name == lit)
				}
				note := ui.Span{Text: mt.Time, Size: t.FontSize - 2, Color: t.TextMuted}
				if mt.Live {
					note.Text, note.Color, note.Weight = "Live", t.Danger, 600
				}
				if mt.Live || !played && mt.Time != "" {
					nw, nh := p.MeasureText(0, note)
					p.RichText(frame.X+boxW-nw, frame.Y-nh-2, 0, note)
				}
			}
		}
		if champion == "" {
			return
		}
		last := places[len(places)-1][0]
		x, y := ox+last.x+boxW+gapX, oy+last.y+rowH/2
		p.Line(ox+last.x+boxW, y+rowH/2, x, y+rowH/2, 1.5, t.Border)
		cup := ui.Rect{X: x, Y: y, W: boxW, H: rowH}
		p.Fill(cup, t.Warning.Alpha(0.18), 6)
		p.Stroke(cup, t.Warning, 6, 1.5)
		name := ui.Span{Text: champion, Size: t.FontSize, Weight: 700, Color: t.Text}
		label := ui.Span{Text: "Champion", Size: t.FontSize - 2, Color: t.TextMuted}
		lw, lh := p.MeasureText(0, label)
		_, nh := p.MeasureText(0, name)
		p.RichText(cup.X+10, cup.Y+(rowH-nh)/2, 0, name)
		p.RichText(cup.X+boxW-10-lw, cup.Y+(rowH-lh)/2, 0, label)
	})
	return ev
}

// side draws s in the row at (x, y) of a match: its seed, its name and
// its score.
func side(p *ui.Painter, t *ui.Theme, s Side, x, y float32, played, lit bool) {
	ink, weight := t.Text, 400
	switch {
	case s.Name == "":
		ink = t.TextMuted
	case s.Won:
		weight = 700
	case played:
		ink = t.TextMuted
	}
	if lit {
		p.Fill(ui.Rect{X: x + 1, Y: y + 1, W: boxW - 2, H: rowH - 2}, t.Accent.Alpha(0.1), 5)
	}
	if s.Seed > 0 {
		seed := ui.Span{Text: strconv.Itoa(s.Seed), Size: t.FontSize - 3, Color: t.TextMuted}
		_, sh := p.MeasureText(0, seed)
		p.RichText(x+10, y+(rowH-sh)/2, 0, seed)
	}
	name := s.Name
	if name == "" {
		name = "TBD"
	}
	span := ui.Span{Text: name, Size: t.FontSize, Weight: weight, Color: ink}
	_, nh := p.MeasureText(0, span)
	p.RichText(x+30, y+(rowH-nh)/2, 0, span)
	if s.Score != "" {
		score := ui.Span{Text: s.Score, Size: t.FontSize, Weight: weight, Color: ink, Features: "tnum"}
		sw, sh := p.MeasureText(0, score)
		p.RichText(x+boxW-10-sw, y+(rowH-sh)/2, 0, score)
	}
}

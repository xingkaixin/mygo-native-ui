package evals

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// pane shows the chosen cell: its case and column, the thumbs rating it,
// whether it passed, its score, latency and cost, and below them the
// prompt, what is expected, the output, and the output against another
// column's.
func (s *State) pane(c *ui.Context, rated map[Cell]Rating, run Run) {
	t := c.Theme()
	cs := run.Cases[s.chosen.Case]
	r := cs.result(s.chosen.Column)
	name := run.Columns[s.chosen.Column]
	ui.Column(c).Width(t.Space(100)).Shrink(0).Border(1, t.Border).Radius(t.Radius).Clip().
		Role(ui.RoleGroup).Label(cs.ID + ", " + name).Children(func() {
		ui.Row(c).Gap(t.Space(1)).Padding(t.Space(2.5), t.Space(2), t.Space(2.5), t.Space(4)).AlignItems(ui.Center).
			Background(t.Surface).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, cs.ID).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
				ui.Text(c, name).Bold().FontSize(t.FontSize + 1).SingleLine()
			})
			rate(c, rated, s.chosen)
			ui.Box(c).Width(t.Space(1))
			if button(c, "Close", t.TextMuted, func(p *ui.Painter, r ui.Rect, color ui.Color) {
				k := r.W * 0.2
				cross(p, ui.Rect{X: r.X + k, Y: r.Y + k, W: r.W - 2*k, H: r.H - 2*k}, color)
			}).Clicked() {
				s.open = false
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Shrink(1).Children(func() {
			ui.Column(c).Padding(t.Space(4)).Gap(t.Space(5)).Children(func() {
				verdict(c, r)
				section(c, "Prompt", func() {
					ui.Text(c, strings.TrimSpace(cs.Prompt)).Selectable()
				})
				section(c, "Expected", func() {
					ui.Text(c, strings.TrimSpace(cs.Expected)).Selectable()
				})
				section(c, "Output", func() {
					block(c, ui.Text(c, strings.TrimSpace(r.Output)))
				})
				s.compare(c, run, cs, r.Output)
			})
		})
	})
}

// verdict shows whether r passed, and its score, latency and cost where
// it has them.
func verdict(c *ui.Context, r Result) {
	t := c.Theme()
	ui.Row(c).Wrap().GapX(t.Space(5)).GapY(t.Space(1)).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			word, color := "Failed", t.Danger
			if r.Pass {
				word, color = "Passed", t.Success
			}
			mark(c, r.Pass, t.FontSize+2)
			ui.Text(c, word).Bold().TextColor(color)
		})
		stat := func(name, value string) {
			ui.Row(c).Gap(t.Space(1.5)).Children(func() {
				ui.Text(c, name).TextColor(t.TextMuted)
				ui.Text(c, value).FontFeatures("tnum")
			})
		}
		if r.Scored {
			stat("Score", score(r.Score))
		}
		if r.Latency > 0 {
			stat("Latency", duration(r.Latency))
		}
		if r.Cost > 0 {
			stat("Cost", cost(r.Cost))
		}
	})
}

// section shows title over what body builds.
func section(c *ui.Context, title string, body func()) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
		body()
	})
}

// block sets text in a box of its own, to copy from.
func block(c *ui.Context, text ui.Element) {
	t := c.Theme()
	text.Selectable().Padding(t.Space(2.5), t.Space(3)).Radius(t.Radius).Background(t.Surface)
}

// compare shows output against that of another column for the same case,
// which a drop-down chooses: the words only the other has struck through
// in red, and those only output has lit in green.
func (s *State) compare(c *ui.Context, run Run, cs Case, output string) {
	t := c.Theme()
	var others []string
	for k, name := range run.Columns {
		if k != s.chosen.Column {
			others = append(others, name)
		}
	}
	if len(others) == 0 {
		return
	}
	if !slices.Contains(others, s.against) {
		s.against = others[0]
	}
	old := strings.TrimSpace(cs.result(slices.Index(run.Columns, s.against)).Output)
	output = strings.TrimSpace(output)
	removed, added := removedSpan(t), addedSpan(t)
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Against").FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
			ui.Select(c, &s.against, others).Label("Compare against")
		})
		if old == output {
			ui.Text(c, "The same output").TextColor(t.TextMuted)
			return
		}
		block(c, ui.RichText(c, diff(removed, added, old, output)...))
		removed.Text, added.Text = s.against, run.Columns[s.chosen.Column]
		ui.RichText(c, removed, ui.Span{Text: " only   "}, added, ui.Span{Text: " only"}).
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
}

func removedSpan(t *ui.Theme) ui.Span {
	return ui.Span{Color: t.Danger, Strikethrough: true, Background: t.Danger.Alpha(0.12), Radius: 3}
}

func addedSpan(t *ui.Theme) ui.Span {
	return ui.Span{Color: t.Text, Background: t.Success.Alpha(0.2), Radius: 3}
}

// diff returns the spans of b against a: the words only a has as removed,
// those only b has as added, and the rest plain.
func diff(removed, added ui.Span, a, b string) []ui.Span {
	wa, wb := words(a), words(b)
	var spans []ui.Span
	var same, del, ins strings.Builder
	emit := func(text *strings.Builder, sp ui.Span) {
		if text.Len() > 0 {
			sp.Text = text.String()
			spans = append(spans, sp)
			text.Reset()
		}
	}
	changes := func() {
		both := del.Len() > 0 && ins.Len() > 0
		emit(&del, removed)
		if both {
			// A thin space keeps the words taken out from running into
			// those put in.
			spans = append(spans, ui.Span{Text: "\u2009"})
		}
		emit(&ins, added)
	}
	i, j := 0, 0
	pairs := append(common(wa, wb), [2]int{len(wa), len(wb)})
	for k, p := range pairs {
		if i < p[0] || j < p[1] {
			emit(&same, ui.Span{})
		}
		for ; i < p[0]; i++ {
			del.WriteString(wa[i])
		}
		for ; j < p[1]; j++ {
			ins.WriteString(wb[j])
		}
		if p[0] == len(wa) {
			break
		}
		w := wa[p[0]]
		next := k+1 < len(pairs) && (pairs[k+1][0] > p[0]+1 || pairs[k+1][1] > p[1]+1)
		if del.Len()+ins.Len() > 0 && next && strings.TrimSpace(w) == "" {
			// A space between two changes joins them, so that a phrase
			// changed reads as one.
			del.WriteString(w)
			ins.WriteString(w)
		} else {
			changes()
			same.WriteString(w)
		}
		i, j = p[0]+1, p[1]+1
	}
	changes()
	emit(&same, ui.Span{})
	return spans
}

// words splits s into runs of letters and digits, runs of spaces, and
// each other character.
func words(s string) []string {
	class := func(r rune) int {
		switch {
		case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			return 1
		case unicode.IsSpace(r):
			return 2
		}
		return 0
	}
	var ws []string
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		j := i + n
		if k := class(r); k != 0 {
			for j < len(s) {
				r, n := utf8.DecodeRuneInString(s[j:])
				if class(r) != k {
					break
				}
				j += n
			}
		}
		ws = append(ws, s[i:j])
		i = j
	}
	return ws
}

// common returns the pairs of indexes of a longest common subsequence of
// a and b, in order, or none when they are too long to compare.
func common(a, b []string) [][2]int {
	n, m := len(a), len(b)
	if n == 0 || m == 0 || n*m > 1<<20 {
		return nil
	}
	// The lengths of the suffixes', so that the pairs come walking forward.
	w := m + 1
	dp := make([]int32, (n+1)*w)
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i*w+j] = dp[(i+1)*w+j+1] + 1
			} else {
				dp[i*w+j] = max(dp[(i+1)*w+j], dp[i*w+j+1])
			}
		}
	}
	var pairs [][2]int
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case a[i] == b[j]:
			pairs = append(pairs, [2]int{i, j})
			i, j = i+1, j+1
		case dp[(i+1)*w+j] >= dp[i*w+j+1]:
			i++
		default:
			j++
		}
	}
	return pairs
}

// rate shows the thumbs up and down rating cell, which a click gives it
// or, given, takes back.
func rate(c *ui.Context, rated map[Cell]Rating, cell Cell) {
	t := c.Theme()
	for _, r := range []Rating{Up, Down} {
		on := rated[cell] == r
		color := t.TextMuted
		if on {
			color = ratingColor(t, r)
		}
		b := button(c.Key(r), ratingName(r), color, func(p *ui.Painter, rr ui.Rect, color ui.Color) {
			thumb(p, rr, r, color)
		}).Role(ui.RoleToggleButton).Checked(on)
		if on {
			b.Background(color.Alpha(0.15))
		}
		if b.Clicked() {
			if on {
				delete(rated, cell)
			} else {
				rated[cell] = r
			}
		}
	}
}

// button creates a square button of icon in color, named label, which its
// tooltip shows.
func button(c *ui.Context, label string, color ui.Color, icon func(p *ui.Painter, r ui.Rect, color ui.Color)) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(7), t.Space(7)).Radius(t.Space(1.5)).Label(label).Tooltip(label)
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.06))
	}
	return b.Draw(func(p *ui.Painter, r ui.Rect) {
		k := t.FontSize + 1
		icon(p, ui.Rect{X: r.X + (r.W-k)/2, Y: r.Y + (r.H-k)/2, W: k, H: k}, color)
	})
}

func ratingName(r Rating) string {
	if r == Up {
		return "Thumbs up"
	}
	return "Thumbs down"
}

func ratingColor(t *ui.Theme, r Rating) ui.Color {
	if r == Up {
		return t.Success
	}
	return t.Danger
}

// mark shows a check for passed or a cross for failed, in a square size
// wide.
func mark(c *ui.Context, pass bool, size float32) {
	t := c.Theme()
	ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.28
		if pass {
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, t.Success)
			return
		}
		cross(p, ui.Rect{X: cx - k, Y: cy - k, W: 2 * k, H: 2 * k}, t.Danger)
	})
}

// cross draws an × across r.
func cross(p *ui.Painter, r ui.Rect, color ui.Color) {
	p.StrokePolyline([][2]float32{{r.X, r.Y}, {r.X + r.W, r.Y + r.H}}, 1.75, color)
	p.StrokePolyline([][2]float32{{r.X + r.W, r.Y}, {r.X, r.Y + r.H}}, 1.75, color)
}

// thumb draws a hand giving rating, a thumb up or down, in r.
func thumb(p *ui.Painter, r ui.Rect, rating Rating, color ui.Color) {
	// The hand is drawn on a grid of 16 by 16, the thumb up, and turned
	// over for down.
	at := func(x, y float32) (float32, float32) {
		if rating == Down {
			y = 16 - y
		}
		return r.X + x*r.W/16, r.Y + y*r.H/16
	}
	var path ui.Path
	move := func(x, y float32) { path.MoveTo(at(x, y)) }
	line := func(x, y float32) { path.LineTo(at(x, y)) }
	quad := func(cx, cy, x, y float32) {
		ax, ay := at(cx, cy)
		bx, by := at(x, y)
		path.QuadTo(ax, ay, bx, by)
	}
	move(1.5, 7)
	line(4.5, 7)
	line(4.5, 14.5)
	line(1.5, 14.5)
	path.Close()
	move(4.5, 7)
	line(7.4, 1.6)
	quad(9.9, 1.6, 9.4, 4.2)
	line(8.9, 6)
	line(13, 6)
	quad(14.9, 6.1, 14.4, 8)
	line(13.4, 13.2)
	quad(13.1, 14.5, 11.8, 14.5)
	line(4.5, 14.5)
	p.StrokePath(&path, max(1, r.W/11), color)
}

// cost writes v in US dollars with two significant digits below a dime,
// "$0.0012", and cents above.
func cost(v float64) string {
	d := 2
	for x := v; x < 0.1 && d < 8; x *= 10 {
		d++
	}
	return "$" + strconv.FormatFloat(v, 'f', d, 64)
}

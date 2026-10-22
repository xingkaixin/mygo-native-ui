// Package evals shows a run of evals, as LLM eval tools show one: the
// outputs of a few models, or variants of a prompt, side by side over the
// same test cases.
//
// The grid has a row a case, its ID and the start of its prompt, and a
// column a model, each cell a check or a cross for whether the output
// passed, its score where it has one, how long it took, and the first
// lines of the output. Above each column are its pass rate, mean score,
// median latency, and the thumbs up and down its outputs were given. The
// headers and the cases stay in place as the grid scrolls either way. All,
// Failing and Disagree show every case, those any column fails, and those
// where the columns differ in pass or fail, each with its count.
//
// A click on a cell, or the arrow keys, choose it and open a pane beside
// the grid: the prompt, what is expected, the output, and the output
// against another column's for the same case, the words only the other
// has struck through and those only this one has lit. Thumbs up and down
// in the pane rate the output, and Escape closes it.
//
//	evals.View(c, &app.evals, &app.ratings, run).Grow(1).AlignSelf(ui.Stretch)
//
// The app keeps the ratings, by cell.
package evals

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Run is a run of evals: the columns it compares, as models or variants
// of a prompt, by their names, each different, and its cases.
type Run struct {
	Columns []string
	Cases   []Case
}

// Case is a test case: its ID, as "refund-03", the prompt, what is
// expected, as the answer or the criteria a grader checks the output by,
// and a result for each column, in their order.
type Case struct {
	ID       string
	Prompt   string
	Expected string
	Results  []Result
}

// result returns the result of column k, failed and empty where cs has
// none.
func (cs Case) result(k int) Result {
	if k < len(cs.Results) {
		return cs.Results[k]
	}
	return Result{}
}

// Result is what a column gave for a case: the output, whether it passed,
// its score from 0 to 1 where Scored, how long it took, and what it cost
// in US dollars, 0 where not known.
type Result struct {
	Output  string
	Pass    bool
	Score   float64
	Scored  bool
	Latency time.Duration
	Cost    float64
}

// Cell is the result of a case for a column, by their indexes in the run.
type Cell struct {
	Case, Column int
}

// Rating is how a person rated an output: Up, Down, or 0 for not rated.
type Rating int8

const (
	Down Rating = -1
	Up   Rating = 1
)

// The filters, in the order they show.
const (
	all = iota
	failing
	disagree
)

// State is what a view keeps: the filter, the cell chosen and whether its
// pane is open, the column its output is compared with, and where the
// grid is scrolled. Keep it in the app's state, one for each run.
type State struct {
	filter  int
	chosen  Cell
	open    bool
	against string
	scroll  ui.ScrollState
	// reveal is how many frames more scroll the chosen cell into view:
	// two, for the grid narrows as the pane opens.
	reveal int
	// first and last are the rows built, to build others when the grid
	// shows them.
	first, last int
}

func (s *State) choose(cell Cell) {
	s.chosen, s.open, s.reveal = cell, true, 2
}

// fails reports whether any column fails cs.
func fails(cs Case) bool {
	return slices.ContainsFunc(cs.Results, func(r Result) bool { return !r.Pass })
}

// disagrees reports whether some columns pass cs and others fail it.
func disagrees(cs Case) bool {
	return fails(cs) && slices.ContainsFunc(cs.Results, func(r Result) bool { return r.Pass })
}

// View creates the filters, the grid of run below them, and, while a cell
// is chosen, its pane beside the grid, whose thumbs set *ratings. Grow and
// stretch it in its parent: the grid scrolls within it.
func View(c *ui.Context, s *State, ratings *map[Cell]Rating, run Run) ui.Element {
	t := c.Theme()
	if *ratings == nil {
		*ratings = map[Cell]Rating{}
	}
	rated := *ratings
	if s.chosen.Case >= len(run.Cases) || s.chosen.Column >= len(run.Columns) {
		s.chosen, s.open = Cell{}, false
	}
	var counts [3]int
	for _, cs := range run.Cases {
		counts[all]++
		if fails(cs) {
			counts[failing]++
		}
		if disagrees(cs) {
			counts[disagree]++
		}
	}
	return ui.Column(c).Gap(t.Space(3)).AlignItems(ui.Stretch).Children(func() {
		ui.Row(c).Children(func() {
			labels := []string{"All", "Failing", "Disagree"}
			for i := range labels {
				labels[i] += " " + strconv.Itoa(counts[i])
			}
			if ui.Segmented(c, &s.filter, labels...).Label("Cases").Changed() {
				s.scroll.Y, s.reveal = 0, 2
			}
		})
		var rows []int
		for i, cs := range run.Cases {
			if s.filter == all || s.filter == failing && fails(cs) || s.filter == disagree && disagrees(cs) {
				rows = append(rows, i)
			}
		}
		body := ui.Row(c).Grow(1).Shrink(1).Gap(t.Space(4)).AlignItems(ui.Stretch)
		if s.open && body.Shortcut(0, ui.KeyEscape) {
			s.open = false
		}
		body.Children(func() {
			s.grid(c, rated, run, rows)
			if s.open {
				s.pane(c, rated, run)
			}
		})
	})
}

// grid creates the grid of the cases in rows, by their indexes in run:
// the headers of the columns above, the cases at the left, and only the
// rows in view built.
func (s *State) grid(c *ui.Context, rated map[Cell]Rating, run Run, rows []int) {
	t := c.Theme()
	cols := len(run.Columns)
	caseW, colW, rowH, headH := t.Space(56), t.Space(62), t.Space(17), t.Space(22)
	width, height := caseW+float32(cols)*colW, float32(max(len(rows), 1))*rowH
	line := t.Border.Alpha(0.6)
	// The grid is as wide as its columns, so that the pane is beside them,
	// and narrows to a column beside the cases, which then scroll sideways.
	box := ui.Column(c).Grow(1).Shrink(1).MinWidth(min(width, caseW+colW) + 2).MaxWidth(width + 2).AlignItems(ui.Stretch)
	box.Children(func() {
		frame := ui.Column(c).Grow(1).Shrink(1).MaxHeight(headH+height+2).
			Border(1, t.Border).Radius(t.Radius).Clip().Focusable().Label("Eval results")
		s.keys(frame, rows, cols)
		at := slices.Index(rows, s.chosen.Case)
		frame.Children(func() {
			var strip ui.Element
			ui.Row(c).Height(headH).Shrink(0).AlignItems(ui.Stretch).Background(t.Surface).Draw(func(p *ui.Painter, r ui.Rect) {
				p.Line(r.X, r.Y+r.H-1, r.X+r.W, r.Y+r.H-1, 1, t.Border)
			}).Children(func() {
				ui.Column(c).Width(caseW).Shrink(0).Justify(ui.End).Padding(t.Space(2.5), t.Space(3)).
					Draw(func(p *ui.Painter, r ui.Rect) {
						p.Line(r.X+r.W-1, r.Y, r.X+r.W-1, r.Y+r.H, 1, t.Border)
					}).Children(func() {
					ui.Text(c, "Case").FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
				})
				strip = ui.Row(c).Grow(1).Shrink(1).Clip().Children(func() {
					for k, name := range run.Columns {
						header(c.Key(k), name, columnStats(run, rated, k), colW, k < cols-1, s.open && at >= 0 && s.chosen.Column == k)
					}
				})
			})
			sc := ui.ScrollBoth(c).Grow(1).Shrink(1)
			view := sc.LayoutBounds()
			if s.reveal > 0 {
				// Scrolled before the grid is built, the cases at the left move
				// with the cells in the same frame.
				if s.open && at >= 0 && view.W > 0 {
					s.scroll.X = nearest(s.scroll.X, float32(s.chosen.Column)*colW, colW, view.W-caseW)
					s.scroll.Y = nearest(s.scroll.Y, float32(at)*rowH, rowH, view.H)
				}
				if s.reveal--; s.reveal > 0 {
					c.Invalidate()
				}
			}
			sc.TrackScroll(&s.scroll)
			strip.ScrollWith(sc)
			s.first = max(0, int(s.scroll.Y/rowH)-1)
			s.last = max(s.first, min(len(rows), int((s.scroll.Y+view.H)/rowH)+2))
			sc.Children(func() {
				content := ui.Box(c).Size(width, height)
				content.Draw(func(p *ui.Painter, r ui.Rect) {
					vis := p.Visible()
					top, bottom := int((vis.Y-r.Y)/rowH), min(len(rows), int((vis.Y+vis.H-r.Y)/rowH)+1)
					if top < s.first || bottom > s.last {
						c.Invalidate()
					}
					for i := s.first; i < s.last && i < len(rows)-1; i++ {
						y := r.Y + float32(i+1)*rowH
						p.Line(r.X, y-1, r.X+r.W, y-1, 1, line)
					}
					for k := 1; k < cols && len(rows) > 0; k++ {
						x := r.X + caseW + float32(k)*colW
						p.Line(x-1, r.Y, x-1, r.Y+r.H, 1, line)
					}
				})
				content.Children(func() {
					if len(rows) == 0 {
						empty := [...]string{"No cases", "No case fails", "The columns agree on every case"}[s.filter]
						ui.Row(c).Absolute().Left(s.scroll.X).Top(0).Height(rowH).Padding(0, t.Space(3)).
							AlignItems(ui.Center).Children(func() {
							ui.Text(c, empty).TextColor(t.TextMuted)
						})
						return
					}
					for i := s.first; i < s.last; i++ {
						cs := run.Cases[rows[i]]
						for k, name := range run.Columns {
							cell := Cell{rows[i], k}
							e := resultCell(c.Key(cell), cs.result(k), rated[cell], cell == s.chosen && s.open, cs.ID+", "+name).
								Left(caseW+float32(k)*colW).Top(float32(i)*rowH).Size(colW, rowH)
							if e.Clicked() {
								s.choose(cell)
								frame.Focus()
								c.Invalidate()
							}
						}
					}
					// The cases stay at the left as the cells scroll under them.
					for i := s.first; i < s.last; i++ {
						caseHeader(c.Key(rows[i]), run.Cases[rows[i]], s.open && i == at).
							Left(s.scroll.X).Top(float32(i)*rowH).Size(caseW, rowH)
					}
				})
			})
		})
	})
}

// nearest returns the offset of a view size long, from off, that shows
// what spans [pos, pos+size) as near as it can.
func nearest(off, pos, size, view float32) float32 {
	switch {
	case pos < off:
		return pos
	case pos+size > off+view:
		return min(pos, pos+size-view)
	}
	return off
}

// keys moves the chosen cell among rows and cols columns as the arrow
// keys say, opening the pane at the cell chosen if it was closed.
func (s *State) keys(frame ui.Element, rows []int, cols int) {
	dr, dc := 0, 0
	switch {
	case frame.Shortcut(0, ui.KeyUp):
		dr = -1
	case frame.Shortcut(0, ui.KeyDown):
		dr = 1
	case frame.Shortcut(0, ui.KeyLeft):
		dc = -1
	case frame.Shortcut(0, ui.KeyRight):
		dc = 1
	default:
		return
	}
	if len(rows) == 0 || cols == 0 {
		return
	}
	at, col := slices.Index(rows, s.chosen.Case), s.chosen.Column
	switch {
	case at < 0:
		at = 0
	case s.open:
		at, col = max(0, min(len(rows)-1, at+dr)), max(0, min(cols-1, col+dc))
	}
	s.choose(Cell{rows[at], col})
}

// header shows the header of a column, name: its pass rate, mean score,
// median latency and ratings in st, w wide, with a line at its end when
// one follows it, lit while one of its cells is chosen.
func header(c *ui.Context, name string, st stats, w float32, follows, lit bool) {
	t := c.Theme()
	ui.Column(c).Width(w).Shrink(0).Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(1.5)).Label(name).
		Draw(func(p *ui.Painter, r ui.Rect) {
			if lit {
				p.Fill(r, t.Accent.Alpha(0.1), 0)
			}
			if follows {
				p.Line(r.X+r.W-1, r.Y, r.X+r.W-1, r.Y+r.H, 1, t.Border.Alpha(0.6))
			}
		}).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, name).Bold().SingleLine().Grow(1).Shrink(1)
			tally(c, Up, st.up)
			tally(c, Down, st.down)
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, percent(st.passed, st.total)).Bold().FontSize(t.FontSize + 3).FontFeatures("tnum")
			ui.Text(c, "pass").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			ui.Box(c).Height(t.Space(1)).Grow(1).Radius(t.Space(0.5)).Background(t.Danger.Alpha(0.7)).
				Draw(func(p *ui.Painter, r ui.Rect) {
					if st.total > 0 {
						p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * float32(st.passed) / float32(st.total), H: r.H}, t.Success, r.H/2)
					}
				})
			ui.Text(c, strconv.Itoa(st.passed)+"/"+strconv.Itoa(st.total)).FontSize(t.FontSize - 1).
				TextColor(t.TextMuted).FontFeatures("tnum")
		})
		var spans []ui.Span
		add := func(value, word string) {
			if len(spans) > 0 {
				spans = append(spans, ui.Span{Text: " · ", Color: t.TextMuted})
			}
			spans = append(spans, ui.Span{Text: value, Features: "tnum"}, ui.Span{Text: " " + word, Color: t.TextMuted})
		}
		if st.scored > 0 {
			add(score(st.score), "mean score")
		}
		if st.median > 0 {
			add(duration(st.median), "median")
		}
		ui.RichText(c, spans...).FontSize(t.FontSize - 1).SingleLine()
	})
}

// tally shows the thumb of rating and how many outputs were given it,
// faint while none were.
func tally(c *ui.Context, rating Rating, n int) {
	t := c.Theme()
	color := t.TextMuted.Alpha(0.5)
	if n > 0 {
		color = ratingColor(t, rating)
	}
	ui.Row(c).Gap(t.Space(0.5)).AlignItems(ui.Center).Label(strconv.Itoa(n) + " " + ratingName(rating)).Children(func() {
		ui.Box(c).Size(t.FontSize, t.FontSize).Draw(func(p *ui.Painter, r ui.Rect) { thumb(p, r, rating, color) })
		e := ui.Text(c, strconv.Itoa(n)).FontSize(t.FontSize - 1).FontFeatures("tnum")
		if n == 0 {
			e.TextColor(t.TextMuted)
		}
	})
}

// caseHeader creates the head of a case's row: its ID and the start of its
// prompt, lit while one of its cells is chosen.
func caseHeader(c *ui.Context, cs Case, lit bool) ui.Element {
	t := c.Theme()
	return ui.Column(c).Absolute().Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(1)).Background(t.Background).
		Draw(func(p *ui.Painter, r ui.Rect) {
			if lit {
				p.Fill(r, t.Accent.Alpha(0.1), 0)
			}
			p.Line(r.X, r.Y+r.H-1, r.X+r.W, r.Y+r.H-1, 1, t.Border.Alpha(0.6))
			p.Line(r.X+r.W-1, r.Y, r.X+r.W-1, r.Y+r.H, 1, t.Border)
		}).Children(func() {
		ui.Text(c, cs.ID).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
		ui.Text(c, strings.Join(strings.Fields(cs.Prompt), " ")).FontSize(t.FontSize - 1).MaxLines(2)
	})
}

// resultCell creates the cell of r: a check or a cross, the score or Pass or
// Fail, the thumb it was rated, how long it took, and the first lines of
// its output; failed cells are tinted, and the chosen one outlined.
func resultCell(c *ui.Context, r Result, rating Rating, chosen bool, label string) ui.Element {
	t := c.Theme()
	verdict, color := "Fail", t.Danger
	if r.Pass {
		verdict, color = "Pass", t.Success
	}
	e := ui.Column(c).Absolute().Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(1)).Cursor(ui.CursorPointer).
		Label(label + ", " + verdict)
	tint, base := float32(0), t.Text
	if !r.Pass {
		tint, base = 0.06, t.Danger
	}
	if e.Hovered() {
		tint += 0.04
	}
	if tint > 0 {
		e.Background(base.Alpha(tint))
	}
	if chosen {
		e.DrawOver(func(p *ui.Painter, rr ui.Rect) {
			p.Stroke(ui.Rect{X: rr.X + 1, Y: rr.Y + 1, W: rr.W - 3, H: rr.H - 3}, t.Accent, 0, 2)
		})
	}
	if r.Scored {
		verdict = score(r.Score)
	}
	return e.Children(func() {
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			mark(c, r.Pass, t.FontSize)
			ui.Text(c, verdict).FontSize(t.FontSize - 1).FontWeight(600).TextColor(color).FontFeatures("tnum")
			ui.Box(c).Grow(1)
			if rating != 0 {
				ui.Box(c).Size(t.FontSize, t.FontSize).Draw(func(p *ui.Painter, rr ui.Rect) {
					thumb(p, rr, rating, ratingColor(t, rating))
				})
			}
			if r.Latency > 0 {
				ui.Text(c, duration(r.Latency)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum")
			}
		})
		ui.Text(c, strings.TrimSpace(r.Output)).FontSize(t.FontSize - 1).MaxLines(2)
	})
}

// stats sums up a column: how many of its cases passed, its mean score
// over those scored, its median latency, and the thumbs up and down given
// its outputs.
type stats struct {
	passed, total, scored int
	score                 float64
	median                time.Duration
	up, down              int
}

func columnStats(run Run, rated map[Cell]Rating, k int) stats {
	var st stats
	var took []time.Duration
	for i, cs := range run.Cases {
		r := cs.result(k)
		st.total++
		if r.Pass {
			st.passed++
		}
		if r.Scored {
			st.score += r.Score
			st.scored++
		}
		if r.Latency > 0 {
			took = append(took, r.Latency)
		}
		switch rated[Cell{i, k}] {
		case Up:
			st.up++
		case Down:
			st.down++
		}
	}
	if st.scored > 0 {
		st.score /= float64(st.scored)
	}
	if n := len(took); n > 0 {
		slices.Sort(took)
		st.median = (took[(n-1)/2] + took[n/2]) / 2
	}
	return st
}

// percent writes n of total as a whole percentage, "80%".
func percent(n, total int) string {
	if total == 0 {
		return "–"
	}
	return strconv.Itoa((200*n+total)/(2*total)) + "%"
}

// score writes a score from 0 to 1 with two decimals, "0.85".
func score(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// duration writes d as "840ms", or from a second as "1.24s".
func duration(d time.Duration) string {
	if d < time.Second {
		return strconv.Itoa(int(d.Milliseconds())) + "ms"
	}
	return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + "s"
}

// Package dag shows jobs that need one another, as CI services show a
// pipeline: the jobs in columns from left to right, each after the jobs
// it needs, ordered so that few links cross; each a card of its status,
// name and detail, such as how long it took, with links from the jobs it
// needs, lit for the job under the pointer and the one chosen. A click
// chooses a job, which Chosen returns.
//
//	dag.View(c, &app.pipeline, app.jobs).Grow(1)
//	if i, ok := app.pipeline.Chosen(); ok {
//		... // show the log of app.jobs[i]
//	}
package dag

import (
	"cmp"
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Status is how a job is doing.
type Status uint8

const (
	Queued Status = iota
	Running
	Passed
	Failed
	Skipped
)

// Job is a job of a graph: its name, how it is doing, a line about it,
// such as how long it took, and the indexes of the jobs it needs.
type Job struct {
	Name   string
	Status Status
	Detail string
	Needs  []int
}

// State is what a view keeps: the job chosen. Keep it in the app's state,
// one for each view.
type State struct {
	// chosen is the index of the job chosen, plus 1, so that the zero
	// State chooses none.
	chosen int
}

// Chosen returns the index of the job chosen.
func (s *State) Chosen() (int, bool) { return s.chosen - 1, s.chosen > 0 }

// layout returns the column of each job, one after the last of those it
// needs, and the jobs of each column, top to bottom, ordered by where
// those they link to are, as Sugiyama's method orders them, so that few
// links cross.
func layout(jobs []Job) ([]int, [][]int) {
	col := make([]int, len(jobs))
	state := make([]uint8, len(jobs))
	var place func(i int) int
	place = func(i int) int {
		switch state[i] {
		case 1:
			// A job that needs itself, by a cycle, goes first.
			return -1
		case 2:
			return col[i]
		}
		state[i] = 1
		c := 0
		for _, k := range jobs[i].Needs {
			if k >= 0 && k < len(jobs) && k != i {
				c = max(c, place(k)+1)
			}
		}
		col[i], state[i] = c, 2
		return c
	}
	n := 0
	for i := range jobs {
		n = max(n, place(i)+1)
	}
	cols := make([][]int, n)
	for i, c := range col {
		cols[c] = append(cols[c], i)
	}
	at := make([]float64, len(jobs))
	for _, ids := range cols {
		for k, i := range ids {
			at[i] = float64(k)
		}
	}
	links := func(i int, before bool) []int {
		if before {
			return jobs[i].Needs
		}
		var out []int
		for j, jb := range jobs {
			if slices.Contains(jb.Needs, i) {
				out = append(out, j)
			}
		}
		return out
	}
	sweep := func(c int, before bool) {
		ids := cols[c]
		key := make(map[int]float64, len(ids))
		for _, i := range ids {
			var sum float64
			var n int
			for _, k := range links(i, before) {
				if k >= 0 && k < len(jobs) {
					sum, n = sum+at[k], n+1
				}
			}
			key[i] = at[i]
			if n > 0 {
				key[i] = sum / float64(n)
			}
		}
		slices.SortStableFunc(ids, func(a, b int) int { return cmp.Compare(key[a], key[b]) })
		for k, i := range ids {
			at[i] = float64(k)
		}
	}
	for range 4 {
		for c := 1; c < n; c++ {
			sweep(c, true)
		}
		for c := n - 2; c >= 0; c-- {
			sweep(c, false)
		}
	}
	return col, cols
}

// View creates the graph of jobs. Give it a size, or Grow; it scrolls
// when it does not fit.
func View(c *ui.Context, s *State, jobs []Job) ui.Element {
	t := c.Theme()
	col, cols := layout(jobs)
	cardW, cardH := t.Space(46), t.Space(13)
	gapX, gapY := t.Space(11), t.Space(4)
	tallest := 0
	for _, ids := range cols {
		tallest = max(tallest, len(ids))
	}
	height := float32(tallest)*(cardH+gapY) - gapY
	pos := make([][2]float32, len(jobs))
	for c, ids := range cols {
		top := (height - (float32(len(ids))*(cardH+gapY) - gapY)) / 2
		for k, i := range ids {
			pos[i] = [2]float32{float32(c) * (cardW + gapX), top + float32(k)*(cardH+gapY)}
		}
	}
	running := false
	for _, j := range jobs {
		running = running || j.Status == Running
	}
	if running {
		c.After(time.Second / 30)
	}
	now := c.Now()
	pad := t.Space(6)
	return ui.ScrollBoth(c).Children(func() {
		box := ui.Box(c).Size(float32(len(cols))*(cardW+gapX)-gapX+2*pad, height+2*pad)
		hovered := -1
		box.Children(func() {
			for i, j := range jobs {
				card := ui.ButtonBase(c.Key(i)).Absolute().Left(pad+pos[i][0]).Top(pad+pos[i][1]).Size(cardW, cardH).
					Radius(t.Space(2)).Background(t.Background).Border(1, t.Border).Label(j.Name).
					Shadow(0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.08))
				if s.chosen == i+1 {
					card.Border(2, t.Accent)
				}
				if card.Hovered() {
					hovered = i
				}
				if card.Clicked() {
					s.chosen = i + 1
				}
				card.Children(func() {
					ui.Row(c).Fill().Gap(t.Space(2.5)).Padding(0, t.Space(3)).AlignItems(ui.Center).Children(func() {
						icon(c, j.Status, now)
						ui.Column(c).Gap(t.Space(0.5)).Shrink(1).Children(func() {
							ui.Text(c, j.Name).SingleLine()
							if j.Detail != "" {
								ui.Text(c, j.Detail).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted).SingleLine()
							}
						})
					})
				})
			}
		})
		chosen := s.chosen - 1
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			for i, j := range jobs {
				for _, k := range j.Needs {
					if k < 0 || k >= len(jobs) || col[k] >= col[i] {
						continue
					}
					x0, y0 := r.X+pad+pos[k][0]+cardW, r.Y+pad+pos[k][1]+cardH/2
					x1, y1 := r.X+pad+pos[i][0], r.Y+pad+pos[i][1]+cardH/2
					color, width := t.Text.Alpha(0.25), float32(1.5)
					if i == hovered || k == hovered || i == chosen || k == chosen {
						color, width = t.Accent, 2
					}
					dx := (x1 - x0) / 2
					var path ui.Path
					path.MoveTo(x0, y0).CubeTo(x0+dx, y0, x1-dx, y1, x1, y1)
					p.StrokePath(&path, width, color)
				}
			}
		})
	})
}

// icon shows status as CI services do: a ring for queued, a turning arc
// for running, a check, a cross and a slash.
func icon(c *ui.Context, status Status, now time.Time) {
	t := c.Theme()
	size := t.Space(4.5)
	ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2-1
		var circle ui.Path
		circle.Circle(cx, cy, rad)
		white := ui.RGB(255, 255, 255)
		switch status {
		case Queued:
			p.StrokePath(&circle, 1.5, t.TextMuted)
		case Running:
			p.StrokePath(&circle, 2, t.Warning.Alpha(0.25))
			turn := float64(now.UnixMilli()%1000) / 1000 * 2 * math.Pi
			var arc ui.Path
			for k := 0; k <= 16; k++ {
				a := turn + float64(k)/16*math.Pi*1.2
				x, y := cx+rad*float32(math.Cos(a)), cy+rad*float32(math.Sin(a))
				if k == 0 {
					arc.MoveTo(x, y)
				} else {
					arc.LineTo(x, y)
				}
			}
			p.StrokePath(&arc, 2, t.Warning)
		case Passed:
			p.FillPath(&circle, t.Success)
			p.StrokePolyline([][2]float32{{cx - rad*0.45, cy}, {cx - rad*0.1, cy + rad*0.35}, {cx + rad*0.5, cy - rad*0.35}}, 1.8, white)
		case Failed:
			p.FillPath(&circle, t.Danger)
			d := rad * 0.38
			p.StrokePolyline([][2]float32{{cx - d, cy - d}, {cx + d, cy + d}}, 1.8, white)
			p.StrokePolyline([][2]float32{{cx + d, cy - d}, {cx - d, cy + d}}, 1.8, white)
		case Skipped:
			p.StrokePath(&circle, 1.5, t.TextMuted)
			d := rad * 0.55
			p.StrokePolyline([][2]float32{{cx + d, cy - d}, {cx - d, cy + d}}, 1.5, t.TextMuted)
		}
	})
}

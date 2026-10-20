// Package gitgraph shows the history of a Git repository as Git clients
// show it: the commits in a list, the newest first, each a dot on the
// lane of its branch, with lines down to its parents, lanes parting where
// branches began and joining where they merged, and the branches and tags
// at a commit as labels before its subject. A click or the arrow keys
// choose a commit, which Chosen returns:
//
//	out, err := exec.Command("git", "log", "--all", "--topo-order", "--format="+gitgraph.Format).Output()
//	app.commits = gitgraph.Parse(string(out)) // once, not every frame
//	...
//	gitgraph.View(c, &app.graph, app.commits).Grow(1)
//	if commit, ok := app.graph.Chosen(); ok {
//		... // show it
//	}
//
// The commits must come children before parents, as git log's
// --topo-order and --date-order put them.
package gitgraph

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Commit is a commit: its hash, those of its parents, the first the
// branch it was made on, the branches and tags pointing at it, as git log
// decorates it ("HEAD -> main", "origin/main", "tag: v1.2.0"), its author,
// when it was made, and its subject.
type Commit struct {
	Hash    string
	Parents []string
	Refs    []string
	Author  string
	Time    time.Time
	Subject string
}

// Format is the format for git log's --format that Parse reads: a line a
// commit, its fields separated by the unit separator, U+001F.
const Format = "%H%x1f%P%x1f%D%x1f%an%x1f%at%x1f%s"

// Parse reads the commits git log printed in Format.
func Parse(log string) []Commit {
	var commits []Commit
	for line := range strings.SplitSeq(log, "\n") {
		f := strings.Split(line, "\x1f")
		if len(f) < 6 {
			continue
		}
		at, _ := strconv.ParseInt(f[4], 10, 64)
		c := Commit{Hash: f[0], Parents: strings.Fields(f[1]), Author: f[3], Time: time.Unix(at, 0), Subject: f[5]}
		if f[2] != "" {
			c.Refs = strings.Split(f[2], ", ")
		}
		commits = append(commits, c)
	}
	return commits
}

// State is what a view keeps: its place in the list, the commit chosen,
// and the lanes laid out for the commits it was given. Keep it in the
// app's state, one for each view.
type State struct {
	list    ui.ListState
	row     int
	commits []Commit
	rows    []row
	lanes   int
}

// Chosen returns the commit chosen, after View, or false with none.
func (s *State) Chosen() (Commit, bool) {
	if s.row < 0 || s.row >= len(s.commits) {
		return Commit{}, false
	}
	return s.commits[s.row], true
}

// row is a commit laid out: its lane, the color of the lane, and the lines
// across its row.
type row struct {
	lane, color int
	lines       []line
}

// line is a line across a row: from lane from at the row's top to lane to
// at its bottom, or for one into or out of the commit's dot, half the
// row from or to it.
type line struct {
	from, to int
	half     half
	color    int
}

type half uint8

const (
	through half = iota
	into
	out
)

// layout lays commits out on lanes, children before parents: a commit
// takes the leftmost of the lanes that wait for it, the others joining it
// there, or a free one; then the lane waits for its first parent, and its
// other parents each take a lane of their own, unless one waits for them
// already. It returns the rows and how many lanes they take at most.
func layout(commits []Commit) ([]row, int) {
	var waits []string
	var colors []int
	next, most := 0, 0
	free := func() int {
		for l, h := range waits {
			if h == "" {
				return l
			}
		}
		waits, colors = append(waits, ""), append(colors, 0)
		return len(waits) - 1
	}
	rows := make([]row, len(commits))
	for i, c := range commits {
		lane := -1
		r := &rows[i]
		for l, h := range waits {
			switch {
			case h == "":
			case h == c.Hash:
				r.lines = append(r.lines, line{from: l, to: l, half: into, color: colors[l]})
				if lane < 0 {
					lane = l
				} else {
					waits[l] = ""
				}
			default:
				r.lines = append(r.lines, line{from: l, to: l, color: colors[l]})
			}
		}
		if lane < 0 {
			lane = free()
			colors[lane], next = next, next+1
		}
		for k := range r.lines {
			if r.lines[k].half == into {
				r.lines[k].to = lane
			}
		}
		r.lane, r.color = lane, colors[lane]
		waits[lane] = ""
		for k, p := range c.Parents {
			l := lane
			if k > 0 {
				l = slices.Index(waits, p)
				if l < 0 {
					l = free()
					colors[l], next = next, next+1
				}
			}
			waits[l] = p
			r.lines = append(r.lines, line{from: lane, to: l, half: out, color: colors[l]})
		}
		for len(waits) > 0 && waits[len(waits)-1] == "" {
			waits, colors = waits[:len(waits)-1], colors[:len(colors)-1]
		}
		most = max(most, len(waits), lane+1)
	}
	return rows, most
}

// laneWidth is how wide a lane is, and maxLanes how many show at most.
const (
	laneWidth = 14
	maxLanes  = 16
)

// View creates the list of commits, the newest first, as git log gave
// them. Give it a height, or Grow.
func View(c *ui.Context, s *State, commits []Commit) ui.Element {
	t := c.Theme()
	if s.list.Selected == nil {
		s.row = -1
		s.list.Selected = &s.row
	}
	if !same(s.commits, commits) {
		s.rows, s.lanes = layout(commits)
	}
	s.commits = commits
	s.list.Label = func(i int) string { return commits[i].Subject }
	rowH := t.FontSize + t.Space(3.5)
	graphW := float32(min(s.lanes, maxLanes))*laneWidth + t.Space(2)
	now := c.Now()
	return ui.List(c, &s.list, len(commits), func(i int) {
		cm := commits[i]
		chosen := i == s.row
		ui.Row(c).Height(rowH).Gap(t.Space(2)).PaddingEnd(t.Space(3)).AlignItems(ui.Center).Children(func() {
			graph(c, s.rows[i], graphW, rowH, chosen)
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Grow(1).Shrink(1).Children(func() {
				for k, ref := range cm.Refs {
					label(c.Key(k), ref, chosen)
				}
				ui.Text(c, cm.Subject).SingleLine().Shrink(1)
			})
			muted := func(e ui.Element) {
				if !chosen {
					e.TextColor(t.TextMuted)
				}
			}
			muted(ui.Text(c, cm.Author).FontSize(t.FontSize - 1).SingleLine().Width(t.Space(30)).Shrink(1))
			muted(ui.Text(c, ago(now, cm.Time)).FontSize(t.FontSize - 1).FontFeatures("tnum").SingleLine().
				Width(t.Space(22)).TextAlign(ui.End))
			muted(ui.Text(c, short(cm.Hash)).Font("monospace").FontSize(t.FontSize - 1).SingleLine().Width(t.Space(16)))
		})
	})
}

// same reports whether a and b are the same commits, as the app passes
// them again from frame to frame.
func same(a, b []Commit) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// graph shows the lanes of r, w wide and h tall, its dot ringed on a
// chosen row.
func graph(c *ui.Context, r row, w, h float32, chosen bool) {
	t := c.Theme()
	ui.Box(c).Size(w, h).Draw(func(p *ui.Painter, b ui.Rect) {
		x := func(lane int) float32 { return b.X + t.Space(2) + float32(lane)*laneWidth + laneWidth/2 }
		top, mid, bottom := b.Y, b.Y+b.H/2, b.Y+b.H
		for _, l := range r.lines {
			if l.from >= maxLanes || l.to >= maxLanes {
				continue
			}
			color := lane(l.color, t.Dark)
			switch l.half {
			case through:
				p.Line(x(l.from), top, x(l.to), bottom, 1.5, color)
			case into:
				curve(p, x(l.from), top, x(l.to), mid, color)
			case out:
				curve(p, x(l.from), mid, x(l.to), bottom, color)
			}
		}
		if r.lane >= maxLanes {
			return
		}
		var dot ui.Path
		dot.Circle(x(r.lane), mid, 4)
		p.FillPath(&dot, lane(r.color, t.Dark))
		ring := t.Background
		if chosen {
			ring = t.AccentText
		}
		p.StrokePath(&dot, 1.5, ring)
	})
}

// curve draws a line from x0, y0 down to x1, y1, bending from the one
// lane to the other.
func curve(p *ui.Painter, x0, y0, x1, y1 float32, color ui.Color) {
	if x0 == x1 {
		p.Line(x0, y0, x1, y1, 1.5, color)
		return
	}
	var path ui.Path
	mid := (y0 + y1) / 2
	path.MoveTo(x0, y0).CubeTo(x0, mid, x1, mid, x1, y1)
	p.StrokePath(&path, 1.5, color)
}

// lane returns the color of the lanes of color i.
func lane(i int, dark bool) ui.Color {
	l := float32(0.62)
	if dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.15, float32((250+i*67)%360))
}

// label shows ref, a branch or a tag, as a chip: "HEAD -> main" as main,
// bold, tags with their tag mark.
func label(c *ui.Context, ref string, chosen bool) {
	t := c.Theme()
	text, color, bold := ref, t.Accent, false
	if name, ok := strings.CutPrefix(ref, "HEAD -> "); ok {
		text, bold = name, true
	} else if name, ok := strings.CutPrefix(ref, "tag: "); ok {
		text, color = "◆ "+name, t.Warning
	}
	chip := ui.Text(c, text).FontSize(t.FontSize-2).SingleLine().Shrink(0).
		Padding(t.Space(0.25), t.Space(1.5)).Radius(t.Space(1)).Border(1, color.Alpha(0.5))
	if bold {
		chip.Bold()
	}
	if chosen {
		chip.Border(1, t.AccentText.Alpha(0.6))
		return
	}
	chip.TextColor(color).Background(color.Alpha(0.1))
}

// short returns the first 7 characters of hash, as Git abbreviates it.
func short(hash string) string { return hash[:min(7, len(hash))] }

// ago returns how long before now t was, "5 minutes ago", or from a week
// on, its date.
func ago(now, t time.Time) string {
	d := now.Sub(t)
	unit := func(n int, word string) string {
		if n == 1 {
			return "1 " + word + " ago"
		}
		return strconv.Itoa(n) + " " + word + "s ago"
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return unit(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return unit(int(d.Hours()), "hour")
	case d < 7*24*time.Hour:
		return unit(int(d.Hours()/24), "day")
	}
	return t.Format("Jan 2, 2006")
}

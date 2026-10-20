// Package testresults shows a run of tests, as coding agents and IDEs
// show one: a header of how many passed, failed and were skipped and how
// long they took, a bar of the shares, and the suites, each a row a click
// opens to its tests, and the tests that printed something a row a click
// opens to it. Suites and tests that fail open as they do, as does a
// run's only suite, unless the user closed them.
//
//	testresults.View(c, &app.tests, []testresults.Suite{
//		{Name: "example.com/notes", Tests: []testresults.Test{
//			{Name: "TestSave", Status: testresults.Passed, Duration: 3 * time.Millisecond},
//			{Name: "TestLoad", Status: testresults.Failed, Output: "load_test.go:12: got nil"},
//		}},
//	})
//
// The app adds tests as they start, Running, and sets their status as
// they end.
package testresults

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Status is how a test goes.
type Status uint8

const (
	Running Status = iota
	Passed
	Failed
	Skipped
)

// Test is a test of a run: its name, how it went, how long it took, and
// what it printed, as why it failed or was skipped.
type Test struct {
	Name     string
	Status   Status
	Duration time.Duration
	Output   string
}

// Suite is the tests of a package or a file, by its name.
type Suite struct {
	Name  string
	Tests []Test
}

// State is what a view keeps: the suites and tests the user opened or
// closed. Keep it in the app's state, one for each run.
type State struct {
	open map[string]bool
}

// opened reports whether the row of key is open: as the user left it, or
// else as def.
func (s *State) opened(key string, def bool) bool {
	if open, ok := s.open[key]; ok {
		return open
	}
	return def
}

// counts is how many tests of a run, or a suite, went each way.
type counts [4]int

func (n counts) total() int { return n[Running] + n[Passed] + n[Failed] + n[Skipped] }

// status returns how tests counted n go together: running while any
// runs, failed with any failed, skipped with all skipped, else passed.
func (n counts) status() Status {
	switch {
	case n[Running] > 0:
		return Running
	case n[Failed] > 0:
		return Failed
	case n[Skipped] == n.total() && n.total() > 0:
		return Skipped
	}
	return Passed
}

// View creates the view of suites. It grows as the suites open: put it
// in a Scroll where they may not fit.
func View(c *ui.Context, s *State, suites []Suite) ui.Element {
	t := c.Theme()
	if s.open == nil {
		s.open = map[string]bool{}
	}
	var all counts
	var took time.Duration
	for _, su := range suites {
		for _, te := range su.Tests {
			all[te.Status]++
			took += te.Duration
		}
	}
	return ui.Column(c).Radius(t.Space(2.5)).Border(1, t.Border).Clip().Role(ui.RoleGroup).Label("Test results").Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
			icon(c, all.status(), t.FontSize+2)
			summary(c, all)
			if took > 0 {
				ui.Text(c, duration(took)).TextColor(t.TextMuted).FontFeatures("tnum")
			}
		})
		bar(c, all)
		ui.Column(c).Padding(t.Space(1)).Children(func() {
			for i, su := range suites {
				suite(c.Key(i), s, su, len(suites) == 1)
			}
		})
	})
}

// summary shows "8 passed · 1 failed · 2 skipped", leaving out those 0,
// growing in its row.
func summary(c *ui.Context, n counts) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(1.5)).Grow(1).Shrink(1).Wrap().Children(func() {
		var parts []struct {
			text  string
			color ui.Color
		}
		add := func(s Status, word string, color ui.Color) {
			if n[s] > 0 {
				parts = append(parts, struct {
					text  string
					color ui.Color
				}{strconv.Itoa(n[s]) + " " + word, color})
			}
		}
		add(Passed, "passed", t.Success)
		add(Failed, "failed", t.Danger)
		add(Skipped, "skipped", t.TextMuted)
		add(Running, "running", t.TextMuted)
		if len(parts) == 0 {
			ui.Text(c, "No tests").TextColor(t.TextMuted)
		}
		for i, p := range parts {
			if i > 0 {
				ui.Text(c, "·").TextColor(t.TextMuted)
			}
			e := ui.Text(c, p.text).FontFeatures("tnum").TextColor(p.color)
			if i == 0 {
				e.Bold()
			}
		}
	})
}

// bar shows the shares of n, passed, failed, skipped and running, along
// the width of the view.
func bar(c *ui.Context, n counts) {
	t := c.Theme()
	ui.Box(c).Height(3).Background(t.Text.Alpha(0.06)).Draw(func(p *ui.Painter, r ui.Rect) {
		total := n.total()
		if total == 0 {
			return
		}
		x := r.X
		for _, part := range []struct {
			s     Status
			color ui.Color
		}{{Passed, t.Success}, {Failed, t.Danger}, {Skipped, t.TextMuted.Alpha(0.5)}} {
			w := r.W * float32(n[part.s]) / float32(total)
			p.Fill(ui.Rect{X: x, Y: r.Y, W: w, H: r.H}, part.color, 0)
			x += w
		}
	})
}

// suite shows su: a row of its name, how many of its tests passed and
// how long they took, which a click opens to the tests; only is whether
// it is the run's only suite.
func suite(c *ui.Context, s *State, su Suite, only bool) {
	t := c.Theme()
	var n counts
	var took time.Duration
	for _, te := range su.Tests {
		n[te.Status]++
		took += te.Duration
	}
	open := s.opened(su.Name, only || n[Failed] > 0)
	ui.Column(c).Children(func() {
		head := row(c, open, true, su.Name)
		if head.Clicked() {
			s.open[su.Name] = !open
		}
		head.Children(func() {
			icon(c, n.status(), t.FontSize+2)
			ui.Text(c, su.Name).SingleLine().Grow(1).Shrink(1)
			ui.Text(c, fmt.Sprintf("%d/%d", n[Passed], n.total()-n[Skipped])).FontSize(t.FontSize - 1).
				TextColor(t.TextMuted).FontFeatures("tnum")
			if took > 0 {
				ui.Text(c, duration(took)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").
					Width(t.Space(14)).TextAlign(ui.End)
			}
		})
		if !open {
			return
		}
		for i, te := range su.Tests {
			test(c.Key(i), s, su.Name+"\x00"+te.Name, te)
		}
	})
}

// test shows te, by key in s, and what it printed while open.
func test(c *ui.Context, s *State, key string, te Test) {
	t := c.Theme()
	out := strings.TrimRight(te.Output, "\n")
	open := out != "" && s.opened(key, te.Status == Failed)
	head := row(c, open, out != "", te.Name).PaddingStart(t.Space(6))
	if head.Clicked() && out != "" {
		s.open[key] = !open
	}
	head.Children(func() {
		icon(c, te.Status, t.FontSize)
		name := ui.Text(c, te.Name).Font("monospace").FontSize(t.FontSize - 1).SingleLine().Grow(1).Shrink(1)
		if te.Status == Skipped {
			name.TextColor(t.TextMuted)
		}
		if te.Duration > 0 {
			ui.Text(c, duration(te.Duration)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").
				Width(t.Space(14)).TextAlign(ui.End)
		}
	})
	if !open {
		return
	}
	color := t.Text
	if te.Status == Failed {
		color = t.Danger
	}
	ui.Scroll(c).MaxHeight(t.Space(50)).Margin(0, t.Space(2), t.Space(1.5), t.Space(11)).Radius(t.Space(1.5)).
		Background(t.Surface).Children(func() {
		ui.Text(c, out).Font("monospace").FontSize(t.FontSize - 1).TextColor(color).Selectable().Padding(t.Space(2))
	})
}

// row creates a row of a suite or a test, label, which a click opens and
// closes when it can open, with an arrow by it.
func row(c *ui.Context, open, opens bool, label string) ui.Element {
	t := c.Theme()
	if !opens {
		return ui.Row(c).Gap(t.Space(2)).Padding(t.Space(1.25), t.Space(4), t.Space(1.25), t.Space(2)).AlignItems(ui.Center)
	}
	b := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(2)).Padding(t.Space(1.25), t.Space(2)).Radius(t.Space(1.5)).
		Label(label).Expanded(open)
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.05))
	}
	return b.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !b.Hovered() && !open {
			return
		}
		x, cy, k := r.X+r.W-t.Space(1.5), r.Y+r.H/2, t.Space(0.6)
		points := [][2]float32{{x - k/2, cy - k}, {x + k/2, cy}, {x - k/2, cy + k}}
		if open {
			points = [][2]float32{{x - k, cy - k/2}, {x, cy + k/2}, {x + k, cy - k/2}}
		}
		p.StrokePolyline(points, 1.5, t.TextMuted)
	}).PaddingEnd(t.Space(4))
}

// icon shows status in a square size wide: a spinner running, a check
// passed, a cross failed, and a dash skipped.
func icon(c *ui.Context, status Status, size float32) {
	t := c.Theme()
	if status == Running {
		ui.Row(c).Size(size, size).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Label("Running")
		})
		return
	}
	ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.28
		switch status {
		case Passed:
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, t.Success)
		case Failed:
			p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.75, t.Danger)
			p.StrokePolyline([][2]float32{{cx + k, cy - k}, {cx - k, cy + k}}, 1.75, t.Danger)
		case Skipped:
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx + k, cy}}, 1.75, t.TextMuted)
		}
	})
}

// duration formats d as go test does, "0.84s", or under a second as
// "12ms".
func duration(d time.Duration) string {
	if d < time.Second {
		return strconv.Itoa(int(d.Milliseconds())) + "ms"
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

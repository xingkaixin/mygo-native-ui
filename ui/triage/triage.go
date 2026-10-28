// Package triage sorts items into labels one at a time, as a queue of
// tickets is triaged or the examples of a classifier are labeled. A strip
// along the top has a cell for each item, filled once it has a label; a
// click on a cell goes to its item. Under it, the item the app builds, on a
// card, and a button for each label with its number key. A click on a
// label, or its key, gives it to the item and goes on to the next one
// without a label. ← and → step through the items, and Skip, or Space,
// passes one by. Once every item has a label, the card counts how many
// each label has.
//
//	triage.View(c, &app.queue, app.labels, &app.answers, func(i int) {
//		ui.Text(c, app.tickets[i].Subject).Bold()
//	})
package triage

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// State is what a queue keeps: the item shown, or the counts once the
// last item gets its label. Keep it in the app's state, one for each queue.
type State struct {
	at     int
	counts bool
}

var digits = []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}

// View creates the queue of *answers, the label of each item or -1 for
// none, build making what item i shows.
func View(c *ui.Context, s *State, labels []string, answers *[]int, build func(i int)) ui.Element {
	t := c.Theme()
	n := len(*answers)
	s.at = max(0, min(n-1, s.at))
	root := ui.Column(c).Gap(t.Space(4)).Focusable().FocusRing(false).AutoFocus().Label("Triage")
	next := func() {
		for k := 1; k <= n; k++ {
			if i := (s.at + k) % n; (*answers)[i] < 0 {
				s.at = i
				return
			}
		}
		s.counts = true
	}
	give := func(l int) {
		(*answers)[s.at] = l
		root.SetChanged()
		next()
	}
	if n > 0 {
		for i, d := range digits[:min(len(labels), len(digits))] {
			if root.Shortcut(0, d) {
				give(i)
			}
		}
		if root.Shortcut(0, ui.KeyLeft) {
			s.at, s.counts = max(0, s.at-1), false
		}
		if root.Shortcut(0, ui.KeyRight) || root.Shortcut(0, ui.KeySpace) {
			s.at, s.counts = min(n-1, s.at+1), false
		}
	}
	at := s.at
	done := 0
	for _, a := range *answers {
		if a >= 0 {
			done++
		}
	}
	root.Value(fmt.Sprintf("Item %d of %d, %d labeled", at+1, n, done))
	root.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%d of %d", min(n, at+1), n)).FontFeatures("tnum").Bold()
			ui.Box(c).Grow(1)
			ui.Text(c, fmt.Sprintf("%d labeled", done)).FontFeatures("tnum").TextColor(t.TextMuted)
		})
		strip(c, s, *answers)
		card := ui.Column(c).Grow(1).MinHeight(t.Space(40)).Padding(t.Space(5)).Gap(t.Space(2)).Radius(t.Space(3)).
			Background(t.Surface).Border(1, t.Border)
		card.Children(func() {
			if s.counts && done == n {
				ui.Text(c, fmt.Sprintf("All %d labeled", n)).FontSize(t.FontSize + 4).Bold()
				counts := make([]int, len(labels))
				for _, a := range *answers {
					if a < len(labels) {
						counts[a]++
					}
				}
				for i, l := range labels {
					ui.Row(c.Key(i)).Gap(t.Space(3)).Children(func() {
						ui.Text(c, l).Width(t.Space(40))
						ui.Text(c, strconv.Itoa(counts[i])).FontFeatures("tnum").TextColor(t.TextMuted)
					})
				}
				ui.Text(c, "← goes back through them").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				return
			}
			if n > 0 {
				ui.Box(c.Key(at)).FillWidth().Children(func() { build(at) })
			}
		})
		ui.Row(c).Gap(t.Space(2)).Wrap().Role(ui.RoleRadioGroup).Label("Labels").Children(func() {
			for i, l := range labels {
				chosen := n > 0 && !s.counts && (*answers)[at] == i
				b := ui.ButtonBase(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).Radius(t.Space(2)).
					Border(1, t.Border).Role(ui.RoleRadio).Checked(chosen).Label(l).Cursor(ui.CursorPointer)
				switch {
				case chosen:
					b.Background(t.Accent).BorderColor(t.Accent)
				case b.Hovered():
					b.Background(t.SurfaceHover)
				}
				if b.Clicked() && n > 0 {
					give(i)
				}
				b.Children(func() {
					fg, muted := t.Text, t.TextMuted
					if chosen {
						fg, muted = t.AccentText, t.AccentText.Alpha(0.75)
					}
					if i < len(digits) {
						ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(muted)
					}
					ui.Text(c, l).TextColor(fg)
				})
			}
			ui.Box(c).Grow(1)
			if ui.Button(c, "Skip").Disabled(at >= n-1).Clicked() {
				s.at, s.counts = s.at+1, false
			}
		})
	})
	return root
}

// strip creates the row of cells, one for each item.
func strip(c *ui.Context, s *State, answers []int) {
	t := c.Theme()
	n := len(answers)
	e := ui.Box(c).FillWidth().Height(t.Space(3)).Cursor(ui.CursorPointer).Role(ui.RoleSlider).Label("Items").
		Range(1, float64(max(1, n)), float64(s.at+1))
	px, _, _ := e.PointerPosition()
	if e.Clicked() && n > 0 {
		if w := e.Bounds().W; w > 0 {
			s.at, s.counts = max(0, min(n-1, int(px/w*float32(n)))), false
		}
	}
	at := s.at
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		if n == 0 {
			return
		}
		gap := float32(2)
		if r.W/float32(n) < 5 {
			gap = 0
		}
		w := (r.W - gap*float32(n-1)) / float32(n)
		for i, a := range answers {
			x := r.X + float32(i)*(w+gap)
			col := t.Border
			if a >= 0 {
				col = t.Accent.Alpha(0.7)
			}
			cell := ui.Rect{X: x, Y: r.Y + 2, W: w, H: r.H - 4}
			if i == at {
				cell = ui.Rect{X: x, Y: r.Y, W: w, H: r.H}
				col = t.Text
			}
			p.Fill(cell, col, min(2, w/2))
		}
	})
}

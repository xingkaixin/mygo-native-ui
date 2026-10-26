// Package subtitle edits the cues of subtitles against the audio they
// go with, as Aegisub and Subtitle Edit do: a timeline of the audio's
// waveform with a ruler of seconds, each cue a box over the time it shows,
// numbered and with its text, and the playhead; under it the cues in a
// list, with their times, their text to edit, and how many characters a
// second they ask the reader to read, in the warning color above 20.
//
// Dragging a cue moves it, and dragging its edges sets when it starts and
// ends; a click on the timeline moves the playhead, and the wheel scrolls
// it, zooming with Cmd. The cue at the playhead is lit in the list as it
// plays, and the timeline follows the playhead. With a cue chosen, [ and ]
// set its start and its end to the playhead; a button adds a cue at the
// playhead, and another deletes the cue chosen.
//
//	cues, err := subtitle.Parse(srt)
//	peaks := waveform.Peaks(samples, 1<<14)
//	...
//	app.subs.Playhead = app.player.Position()
//	ev := subtitle.View(c, &app.subs, &cues, peaks, app.player.Duration())
//	if at, ok := ev.Seeked(); ok {
//		app.player.Seek(at)
//	}
//
// Parse reads SubRip (.srt) text, and Format writes it.
package subtitle

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/waveform"
)

// Cue is a subtitle: its text, shown from Start to End.
type Cue struct {
	Start, End time.Duration
	Text       string
}

// CPS returns how many characters a second the cue asks the reader to
// read, its line breaks left out.
func (q Cue) CPS() float64 {
	secs := (q.End - q.Start).Seconds()
	if secs <= 0 {
		return 0
	}
	return float64(utf8.RuneCountInString(strings.ReplaceAll(q.Text, "\n", ""))) / secs
}

// ErrSyntax is the error of text that is not SubRip.
var ErrSyntax = errors.New("subtitle: not SubRip text")

// Parse reads cues from SubRip text: blocks of a number, a line of times
// such as "00:00:01,600 --> 00:00:04,200", and text, apart by blank lines.
func Parse(srt string) ([]Cue, error) {
	srt = strings.TrimPrefix(strings.ReplaceAll(srt, "\r\n", "\n"), "\ufeff")
	var cues []Cue
	for block := range strings.SplitSeq(strings.TrimSpace(srt), "\n\n") {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) == 0 || lines[0] == "" {
			continue
		}
		if !strings.Contains(lines[0], "-->") {
			lines = lines[1:]
		}
		if len(lines) == 0 {
			return nil, ErrSyntax
		}
		a, b, ok := strings.Cut(lines[0], "-->")
		if !ok {
			return nil, ErrSyntax
		}
		start, err1 := parseTime(strings.TrimSpace(a))
		end, err2 := parseTime(strings.Fields(strings.TrimSpace(b) + " ")[0])
		if err1 != nil || err2 != nil {
			return nil, ErrSyntax
		}
		cues = append(cues, Cue{Start: start, End: end, Text: strings.Join(lines[1:], "\n")})
	}
	return cues, nil
}

// parseTime reads a time such as "01:02:03,450".
func parseTime(s string) (time.Duration, error) {
	var h, m, sec, ms int
	if _, err := fmt.Sscanf(strings.Replace(s, ".", ",", 1), "%d:%d:%d,%d", &h, &m, &sec, &ms); err != nil {
		return 0, err
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second + time.Duration(ms)*time.Millisecond, nil
}

// Format writes cues as SubRip text, numbered from 1.
func Format(cues []Cue) string {
	var sb strings.Builder
	for i, q := range cues {
		fmt.Fprintf(&sb, "%d\n%s --> %s\n%s\n\n", i+1, stamp(q.Start, ','), stamp(q.End, ','), q.Text)
	}
	return sb.String()
}

// stamp writes d as hours, minutes, seconds and milliseconds, the last
// after sep.
func stamp(d time.Duration, sep byte) string {
	ms := d.Milliseconds()
	return fmt.Sprintf("%02d:%02d:%02d%c%03d", ms/3600000, ms/60000%60, ms/1000%60, sep, ms%1000)
}

// State is what an editor keeps: the cue chosen, how far the timeline is
// zoomed and scrolled, and the drag going on. Playhead, which the app
// moves as the audio plays, is where it plays from. Keep it in the app's
// state, one for each editor.
type State struct {
	Playhead time.Duration

	chosen int
	// scale is how many DIPs a second takes, and from the time at the
	// timeline's left.
	scale float64
	from  time.Duration
	// drag is what a press on the timeline drags, and dx how far, in
	// DIPs; grab is a cue's times as the drag began.
	drag  part
	cue   int
	dx    float64
	grab  Cue
	seen  time.Duration
	refs  []ui.Ref
	width float32
}

// part is what a drag on the timeline moves.
type part uint8

const (
	none part = iota
	seek
	move
	startEdge
	endEdge
)

// Events is what the user did in an editor in a frame.
type Events struct {
	edited   bool
	seeked   bool
	position time.Duration
}

// Edited reports an edit of a cue: its times, its text, a cue added or
// deleted.
func (e Events) Edited() bool { return e.edited }

// Seeked returns where the user moved the playhead to, by a click on the
// timeline or on a cue in the list. Seek the audio there.
func (e Events) Seeked() (time.Duration, bool) { return e.position, e.seeked }

// least is how short a cue gets.
const least = 200 * time.Millisecond

// View creates the editor of *cues, which it edits in place, over the
// waveform peaks of audio length long. Give it a size, or grow it in its
// parent.
func View(c *ui.Context, s *State, cues *[]Cue, peaks []waveform.Peak, length time.Duration) Events {
	t := c.Theme()
	var ev Events
	if s.scale == 0 {
		s.scale, s.chosen = 120, -1
	}
	if len(s.refs) != len(*cues) {
		s.refs = make([]ui.Ref, len(*cues))
	}
	playing := -1
	for i, q := range *cues {
		if s.Playhead >= q.Start && s.Playhead < q.End {
			playing = i
		}
	}
	ui.Column(c).Grow(1).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, stamp(s.Playhead, '.')).Font("monospace").Bold().FontFeatures("tnum")
			ui.Text(c, "/ "+stamp(length, '.')).Font("monospace").TextColor(t.TextMuted)
			ui.Spacer(c)
			if ui.Button(c, "Add at playhead").Clicked() {
				s.add(cues, length)
				ev.edited = true
			}
			has := s.chosen >= 0 && s.chosen < len(*cues)
			if ui.Button(c, "[ Start").Tooltip("Set the start to the playhead ([)").Disabled(!has).Clicked() || has && c.Shortcut(0, ui.KeyBracketLeft) {
				q := &(*cues)[s.chosen]
				q.Start, ev.edited = min(s.Playhead, q.End-least), true
			}
			if ui.Button(c, "End ]").Tooltip("Set the end to the playhead (])").Disabled(!has).Clicked() || has && c.Shortcut(0, ui.KeyBracketRight) {
				q := &(*cues)[s.chosen]
				q.End, ev.edited = max(s.Playhead, q.Start+least), true
			}
			if ui.Button(c, "Delete").Disabled(!has).Clicked() {
				*cues = slices.Delete(*cues, s.chosen, s.chosen+1)
				s.refs = slices.Delete(s.refs, s.chosen, s.chosen+1)
				s.chosen, ev.edited = -1, true
			}
		})
		ui.Divider(c)
		s.timeline(c, cues, peaks, length, &ev)
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			for i := range *cues {
				s.row(c.Key(i), i, &(*cues)[i], i == playing, &ev)
			}
		})
	})
	return ev
}

// add adds a cue of two seconds at the playhead, shorter when the next cue
// starts sooner, unless one shows there.
func (s *State) add(cues *[]Cue, length time.Duration) {
	at := s.Playhead
	i, _ := slices.BinarySearchFunc(*cues, at, func(q Cue, at time.Duration) int { return int(q.Start - at) })
	if i > 0 && (*cues)[i-1].End > at {
		s.chosen = i - 1
		return
	}
	end := at + 2*time.Second
	if i < len(*cues) {
		end = min(end, (*cues)[i].Start)
	}
	if length > 0 {
		end = min(end, length)
	}
	if end-at < least {
		return
	}
	*cues = slices.Insert(*cues, i, Cue{Start: at, End: end})
	s.refs = slices.Insert(s.refs, i, ui.Ref{})
	s.chosen = i
}

// timeline creates the timeline of cues over peaks.
func (s *State) timeline(c *ui.Context, cues *[]Cue, peaks []waveform.Peak, length time.Duration, ev *Events) {
	t := c.Theme()
	box := ui.Box(c).Height(t.Space(36)).FillWidth().Clip().Focusable().FocusRing(false).Label("Timeline")
	w := box.LayoutBounds().W
	s.width = w
	sec := func(d time.Duration) float32 { return float32((d - s.from).Seconds() * s.scale) }
	at := func(x float32) time.Duration {
		return s.from + time.Duration(float64(x)/s.scale*float64(time.Second))
	}
	clampFrom := func() {
		most := length - time.Duration(float64(w)/s.scale*float64(time.Second))
		s.from = max(0, min(s.from, most))
	}
	// The timeline follows the playhead as it plays off its right edge.
	if s.Playhead != s.seen && s.drag == none && w > 0 {
		s.seen = s.Playhead
		if x := sec(s.Playhead); x < 0 || x > w {
			s.from = s.Playhead - time.Duration(float64(w)/s.scale*0.1*float64(time.Second))
			clampFrom()
		}
	}
	box.HandleInput(func(in ui.InputEvent) bool {
		if in.Kind != ui.InputScroll {
			return false
		}
		if in.Mods&ui.Cmd != 0 {
			pivot := at(in.X)
			s.scale = min(2000, max(10, s.scale*math.Exp2(-float64(in.DY)/300)))
			s.from = pivot - time.Duration(float64(in.X)/s.scale*float64(time.Second))
		} else {
			d := in.DX
			if d == 0 {
				d = in.DY
			}
			s.from += time.Duration(float64(d) / s.scale * float64(time.Second))
		}
		clampFrom()
		c.Invalidate()
		return true
	})
	top := t.Space(6)
	edge := t.Space(1.5)
	x, y, over := box.PointerPosition()
	// hit returns the cue and the part of it at x, y.
	hit := func(x, y float32) (int, part) {
		if y < top {
			return -1, seek
		}
		for i := len(*cues) - 1; i >= 0; i-- {
			q := (*cues)[i]
			x0, x1 := sec(q.Start), sec(q.End)
			switch {
			case x >= x0-edge && x <= x0+edge:
				return i, startEdge
			case x >= x1-edge && x <= x1+edge:
				return i, endEdge
			case x > x0 && x < x1:
				return i, move
			}
		}
		return -1, seek
	}
	if over {
		if _, p := hit(x, y); p == startEdge || p == endEdge {
			box.Cursor(ui.CursorResizeEW)
		}
	}
	if box.Pressed() && s.drag == none && over {
		s.cue, s.drag = hit(x, y)
		s.dx = 0
		if s.cue >= 0 {
			s.chosen, s.grab = s.cue, (*cues)[s.cue]
		}
	}
	if s.drag != none {
		dx, _, _ := box.Dragged()
		s.dx += float64(dx)
		d := time.Duration(s.dx / s.scale * float64(time.Second)).Round(10 * time.Millisecond)
		switch s.drag {
		case seek:
			s.Playhead = max(0, min(length, at(x)))
			s.seen = s.Playhead
			ev.seeked, ev.position = true, s.Playhead
		case move:
			q := &(*cues)[s.cue]
			n := s.grab.End - s.grab.Start
			q.Start = max(0, s.grab.Start+d)
			if length > 0 {
				q.Start = min(q.Start, length-n)
			}
			q.End = q.Start + n
			ev.edited = ev.edited || d != 0
		case startEdge:
			q := &(*cues)[s.cue]
			q.Start = max(0, min(s.grab.Start+d, q.End-least))
			ev.edited = ev.edited || d != 0
		case endEdge:
			q := &(*cues)[s.cue]
			q.End = max(q.Start+least, s.grab.End+d)
			if length > 0 {
				q.End = min(q.End, length)
			}
			ev.edited = ev.edited || d != 0
		}
		if !box.Pressed() {
			s.drag = none
		}
	}
	chosen, playhead, from, scale := s.chosen, s.Playhead, s.from, s.scale
	list := *cues
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		px := func(d time.Duration) float32 { return r.X + float32((d-from).Seconds()*scale) }
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: top}, t.Surface, 0)
		step := tickStep(scale)
		small := t.FontSize - 3
		for tk := from.Truncate(step); tk <= from+time.Duration(float64(r.W)/scale*float64(time.Second)); tk += step {
			x := px(tk)
			if x < r.X {
				continue
			}
			p.Line(x, r.Y+top-t.Space(1.5), x, r.Y+top, 1, t.TextMuted.Alpha(0.6))
			label := strconv.Itoa(int(tk.Minutes())) + ":" + fmt.Sprintf("%02d", int(tk.Seconds())%60)
			if step < time.Second {
				label = fmt.Sprintf("%d:%04.1f", int(tk.Minutes()), math.Mod(tk.Seconds(), 60))
			}
			p.Text(x+2, r.Y+1, label, small, t.TextMuted)
		}
		mid := r.Y + top + (r.H-top)/2
		half := (r.H - top) / 2 * 0.85
		if len(peaks) > 0 && length > 0 {
			wave := t.TextMuted.Alpha(0.45)
			for x := float32(0); x < r.W; x++ {
				a, b := at(x), at(x+1)
				i0 := int(float64(a) / float64(length) * float64(len(peaks)))
				i1 := max(i0+1, int(float64(b)/float64(length)*float64(len(peaks))))
				if i0 < 0 || i0 >= len(peaks) {
					continue
				}
				lo, hi := peaks[i0].Lo, peaks[i0].Hi
				for _, pk := range peaks[i0:min(i1, len(peaks))] {
					lo, hi = min(lo, pk.Lo), max(hi, pk.Hi)
				}
				p.Fill(ui.Rect{X: r.X + x, Y: mid - hi*half, W: 1, H: max(1, (hi-lo)*half)}, wave, 0)
			}
		}
		for i, q := range list {
			x0, x1 := px(q.Start), px(q.End)
			if x1 < r.X || x0 > r.X+r.W {
				continue
			}
			band := ui.Rect{X: x0, Y: r.Y + top + t.Space(2), W: x1 - x0, H: r.H - top - t.Space(4)}
			fill, line := t.Accent.Alpha(0.14), t.Accent.Alpha(0.5)
			if i == chosen {
				fill, line = t.Accent.Alpha(0.26), t.Accent
			}
			p.Fill(band, fill, t.Space(1))
			p.Stroke(band, line, t.Space(1), 1)
			p.Fill(ui.Rect{X: x0, Y: band.Y, W: 2, H: band.H}, line, 0)
			p.Fill(ui.Rect{X: x1 - 2, Y: band.Y, W: 2, H: band.H}, line, 0)
			p.Clip(band, t.Space(1), func() {
				p.Text(x0+t.Space(1.5), band.Y+t.Space(1), strconv.Itoa(i+1), small, t.Accent)
				text := strings.ReplaceAll(q.Text, "\n", " / ")
				p.Text(x0+t.Space(1.5), band.Y+band.H-t.FontSize-t.Space(1), text, t.FontSize-2, t.Text)
			})
		}
		hx := px(playhead)
		p.Line(hx, r.Y, hx, r.Y+r.H, 1.5, t.Danger)
	})
}

// tickStep returns the time between the ruler's ticks at scale DIPs a
// second, keeping them 60 DIPs apart at least.
func tickStep(scale float64) time.Duration {
	for _, d := range []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond,
		time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second, time.Minute} {
		if d.Seconds()*scale >= 60 {
			return d
		}
	}
	return 5 * time.Minute
}

// row creates the row of the i-th cue, q, in the list, lit while playing.
func (s *State) row(c *ui.Context, i int, q *Cue, playing bool, ev *Events) {
	t := c.Theme()
	r := ui.Row(c).Padding(t.Space(1.5), t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center)
	switch {
	case i == s.chosen:
		r.Background(t.Selection)
	case playing:
		r.Background(t.Accent.Alpha(0.08))
	}
	if r.FocusWithin() && s.chosen != i {
		s.chosen = i
		c.Invalidate()
	}
	r.Children(func() {
		ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").Width(t.Space(7)).TextAlign(ui.End)
		times := ui.ButtonBase(c).Label("Go to cue " + strconv.Itoa(i+1)).Tooltip("Go to its start")
		times.Children(func() {
			ui.Column(c).PassThrough().Children(func() {
				ui.Text(c, stamp(q.Start, '.')).Font("monospace").FontSize(t.FontSize - 1).PassThrough()
				ui.Text(c, stamp(q.End, '.')).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted).PassThrough()
			})
		})
		if times.Clicked() {
			s.chosen, s.Playhead = i, q.Start
			ev.seeked, ev.position = true, q.Start
		}
		if ui.TextArea(c, &q.Text).Bind(&s.refs[i]).Lines(1, 3).MinHeight(0).Label("Text of cue " + strconv.Itoa(i+1)).
			Grow(1).Shrink(1).MinWidth(0).Changed() {
			ev.edited = true
		}
		cps := q.CPS()
		col := t.TextMuted
		if cps > 20 {
			col = t.Warning
		}
		ui.Text(c, strconv.FormatFloat(cps, 'f', 0, 64)+" cps").FontSize(t.FontSize - 2).TextColor(col).FontFeatures("tnum").
			Width(t.Space(13)).TextAlign(ui.End).Tooltip("Characters a second")
	})
	if playing && s.drag == none && s.chosen < 0 {
		r.ScrollIntoView()
	}
}

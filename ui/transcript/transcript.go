// Package transcript shows a transcript of speech that follows playback,
// as transcription apps and podcast players show it. Each segment shows
// its speaker, by a dot in a color of their own, when it starts, and its
// words, wrapping at the width of the view. The word at the playhead is
// lit, the words spoken before it read in full, and those to come muted.
// Words the transcription was less sure of than Threshold show a dotted
// underline, and their confidence once the pointer rests on them.
//
// A click on a word moves the playhead to its start. Dragging selects
// text across words and segments, and Copy puts it on the clipboard as
// plain text: the words of each segment under its speaker and start.
// While the playhead moves, the view scrolls to keep it in view; once the
// user scrolls away, it stays, and a "Jump to current" button goes back.
//
// The field above the words finds text in them: it lights every match,
// hiding nothing, and counts them. Enter and the down arrow go to the next
// match, Shift+Enter and the up arrow to the one before, round the ends,
// each moving the playhead there, and the count reads "2 of 9".
//
//	ui.Column(c).Grow(1).Children(func() {
//		ev := transcript.View(c, &app.transcript, &app.position, app.segments)
//		if at, ok := ev.Seeked(); ok {
//			app.player.Seek(at)
//		}
//	})
//
// The app keeps the playhead and moves it as it plays. The view moves it
// as the user clicks a word or goes to a match, and reports the seek in
// the frame it happens, for the app to move its player there.
package transcript

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// Word is a word as spoken: its text, with the punctuation the
// transcription gave it, as "roof,"; when it starts and ends; and how sure
// the transcription was of it, from 0 to 1, or 0 when it does not tell.
type Word struct {
	Text       string
	Start, End time.Duration
	Confidence float64
}

// Segment is what one speaker said in a stretch, from Start to End, in
// words. Give segments, and the words in each, in order.
type Segment struct {
	Speaker    string
	Start, End time.Duration
	Words      []Word
}

// Events are what the user did with a view in a frame. The zero Events
// report nothing.
type Events struct {
	seek   time.Duration
	seeked bool
}

// Seeked returns where the user moved the playhead, by a click on a word
// or going to a match, already in *playhead: move the player there in
// this frame.
func (e Events) Seeked() (time.Duration, bool) { return e.seek, e.seeked }

// State is what a view keeps: the segments' words indexed, the text the
// field finds and its matches, the scrolling, and whether the user
// scrolled away from the playhead. Threshold, which the app sets, is the
// confidence, from 0 to 1, below which a word shows a dotted underline.
// Keep it in the app's state, one for each view; the zero State
// underlines none.
type State struct {
	Threshold float64

	segments []Segment
	// colors is the color of each segment's speaker, an index of hues, and
	// first the index of each segment's first word among all the words.
	// texts are the words with the space after them, starts when each
	// starts and until when its light goes out: at the next word's start,
	// or for a segment's last, at the end of it and of the segment. at is
	// where each word starts in its segment's text, the words joined by
	// spaces, in runes.
	colors        []int
	first         []int
	texts         []string
	starts, until []time.Duration
	at            [][]int
	long          bool

	// query is the text the field finds, found the one matches were found
	// for, and match the match gone to, plus 1, so that 0 is none.
	query, found string
	matches      []match
	match        int

	// scroll is the scrolling of the words, y where the view left it, and
	// away whether the user scrolled since the view last followed the
	// playhead.
	scroll ui.ScrollState
	y      float32
	away   bool
	// The boxes of the last frame, in the window: the scrolling area's
	// height, its content, each segment and each word. ids are the
	// segments' paragraphs, which the parts of a copy name.
	viewH       float32
	content     ui.Rect
	segs, words []ui.Rect
	ids         []uint64
	// pressed is a press on the words, which is a click unless it selects
	// text.
	pressed bool
}

// match is a match of the field's text: runes from to to of segment
// seg's text.
type match struct{ seg, from, to int }

// hues are those of the speakers' colors, in the order they first speak.
var hues = []float32{255, 25, 145, 310, 70, 195}

// lineHeight is the height of a line of words, in sizes of their font.
const lineHeight = 1.6

// prepare indexes segments.
func (s *State) prepare(segments []Segment) {
	n := len(segments)
	s.segments, s.long = segments, false
	s.colors, s.first, s.at = make([]int, n), make([]int, n), make([][]int, n)
	s.segs, s.ids = make([]ui.Rect, n), make([]uint64, n)
	s.texts, s.starts, s.until = s.texts[:0], s.starts[:0], s.until[:0]
	speakers := map[string]int{}
	for i, seg := range segments {
		color, ok := speakers[seg.Speaker]
		if !ok {
			color = len(speakers)
			speakers[seg.Speaker] = color
		}
		s.colors[i], s.first[i] = color, len(s.starts)
		s.at[i] = make([]int, len(seg.Words))
		r := 0
		for k, w := range seg.Words {
			s.at[i][k] = r
			r += utf8.RuneCountInString(w.Text) + 1
			text, until := w.Text, max(w.End, seg.End)
			if k+1 < len(seg.Words) {
				text, until = w.Text+" ", seg.Words[k+1].Start
			}
			s.texts = append(s.texts, text)
			s.starts = append(s.starts, w.Start)
			s.until = append(s.until, until)
		}
		s.long = s.long || seg.End >= time.Hour
	}
	s.words = make([]ui.Rect, len(s.starts))
	s.search(s.query)
}

// same reports whether a and b are the same segments, as the app passes
// them again from frame to frame.
func same(a, b []Segment) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// search finds query in the segments' text, case aside.
func (s *State) search(query string) {
	s.found, s.matches, s.match = query, s.matches[:0], 0
	if query == "" {
		return
	}
	q := strings.ToLower(query)
	n := utf8.RuneCountInString(q)
	for i := range s.segments {
		// Lowering keeps the runes, though not always their bytes.
		lower := strings.ToLower(strings.Join(s.texts[s.first[i]:s.first[i]+len(s.at[i])], ""))
		at, runes := 0, 0
		for {
			k := strings.Index(lower[at:], q)
			if k < 0 {
				break
			}
			from := runes + utf8.RuneCountInString(lower[at:at+k])
			s.matches = append(s.matches, match{i, from, from + n})
			at, runes = at+k+len(q), from+n
		}
	}
}

// current returns the index of the last word to start by d among all the
// words, or -1, and whether d is within its light.
func (s *State) current(d time.Duration) (int, bool) {
	g := sort.Search(len(s.starts), func(i int) bool { return s.starts[i] > d }) - 1
	return g, g >= 0 && d < s.until[g]
}

// segment returns the segment of word g among all the words.
func (s *State) segment(g int) int {
	return sort.Search(len(s.first), func(i int) bool { return s.first[i] > g }) - 1
}

// View creates the field finding text in segments and below it the
// segments, scrolling, at *playhead. It fills what it is built in: give
// that a size, or grow it in its parent.
func View(c *ui.Context, s *State, playhead *time.Duration, segments []Segment) Events {
	t := c.Theme()
	if !same(s.segments, segments) {
		s.prepare(segments)
	}
	var ev Events
	seek := func(d time.Duration) {
		*playhead, s.away = d, false
		ev = Events{seek: d, seeked: true}
	}
	// An offset past the end, as the content shrank, moved back to it
	// without the user.
	if s.scroll.Y != s.y && s.y <= s.scroll.MaxY {
		s.away = true
	}
	ui.Column(c).Fill().Gap(t.Space(3)).Children(func() {
		s.find(c, seek)
		g, lit := s.current(*playhead)
		ui.Column(c).Grow(1).Children(func() {
			area := ui.Scroll(c).Grow(1).TrackScroll(&s.scroll).Label("Transcript")
			s.viewH = area.LayoutBounds().H
			area.Children(func() {
				if s.text(c, g, lit, seek) {
					c.Invalidate()
				}
			})
			// The words' boxes are those of the last layout once they are
			// built: the scrolling starts in the next frame.
			if !s.away && s.follow(max(g, 0), t.Space(6)) {
				c.Invalidate()
			}
			if s.away && len(s.starts) > 0 {
				jump := ui.Button(c, "Jump to current").Absolute().Right(t.Space(4)).Bottom(t.Space(4)).
					Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.15))
				if jump.Clicked() {
					s.away = false
					c.Invalidate()
				}
			}
		})
	})
	s.y = s.scroll.Y
	return ev
}

// find creates the field finding text in the segments, the count of its
// matches, and the buttons going to the one before and the next; going
// to one seeks to its word.
func (s *State) find(c *ui.Context, seek func(time.Duration)) {
	t := c.Theme()
	step := func(d int) {
		n := len(s.matches)
		if n == 0 {
			return
		}
		switch {
		case s.match > 0:
			s.match = (s.match-1+d+n)%n + 1
		case d > 0:
			s.match = 1
		default:
			s.match = n
		}
		m := s.matches[s.match-1]
		k := sort.SearchInts(s.at[m.seg], m.from+1) - 1
		seek(s.starts[s.first[m.seg]+k])
	}
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		field := ui.SearchField(c, &s.query).Label("Find in transcript").Grow(1).Shrink(1).MaxWidth(t.Space(72))
		if s.query != s.found {
			s.search(s.query)
		}
		n := len(s.matches)
		if field.Submitted() {
			// The field tells Enter, not the keys held with it, which are
			// still down as the frame follows.
			if c.Modifiers()&ui.Shift != 0 {
				step(-1)
			} else {
				step(1)
			}
		}
		if arrow(c, "Previous match", true, n == 0) {
			step(-1)
		}
		if arrow(c, "Next match", false, n == 0) {
			step(1)
		}
		status := ""
		switch {
		case s.query == "":
		case n == 0:
			status = "No matches"
		case s.match > 0:
			status = strconv.Itoa(s.match) + " of " + strconv.Itoa(n)
		case n == 1:
			status = "1 match"
		default:
			status = strconv.Itoa(n) + " matches"
		}
		ui.Text(c, status).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
	})
}

// arrow creates a button named label showing a chevron pointing up, or
// down, and reports a click on it.
func arrow(c *ui.Context, label string, up, disabled bool) bool {
	t := c.Theme()
	b := ui.Button(c, "").Label(label).Tooltip(label).Disabled(disabled).Padding(t.Space(1.5), t.Space(2))
	b.Children(func() {
		ui.Box(c).Size(t.Space(3), t.Space(3)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, d := r.X+r.W/2, r.Y+r.H/2, r.W/4
			if up {
				p.StrokePolyline([][2]float32{{cx - d, cy + d/2}, {cx, cy - d/2}, {cx + d, cy + d/2}}, 1.5, t.Text)
			} else {
				p.StrokePolyline([][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}, 1.5, t.Text)
			}
		})
	})
	return b.Clicked()
}

// follow scrolls to show word g, unless it shows already: to the top of
// its segment, or to the word where the segment is too tall for both,
// pad clear of the edges. It goes by the boxes of the last frame, as the
// words wrapped then, and reports whether it scrolls.
func (s *State) follow(g int, pad float32) bool {
	if g >= len(s.words) || s.words[g].H == 0 || s.scroll.Scrolling() {
		return false
	}
	top := s.words[g].Y - s.content.Y
	bottom := top + s.words[g].H
	y, h := s.scroll.Y, s.viewH
	if top >= y && bottom <= y+h-pad {
		return false
	}
	to := s.segs[s.segment(g)].Y - s.content.Y - pad
	if bottom > to+h-pad {
		to = top - pad
	}
	if to = max(0, min(to, s.scroll.MaxY)); math.Abs(float64(to-y)) <= 1 {
		return false
	}
	s.scroll.ScrollTo(0, to, ui.Smooth)
	return true
}

// text creates the segments, the playhead at word g among all, lit with
// lit, as selectable text; a click on a word seeks to it. It reports
// whether it seeked, which the words show in the next frame.
func (s *State) text(c *ui.Context, g int, lit bool, seek func(time.Duration)) bool {
	t := c.Theme()
	size := t.FontSize + 2
	m := ui.Font{Family: t.Font, Size: size}.Metrics()
	view := ui.Column(c).Gap(t.Space(6)).Padding(t.Space(2), t.Space(4), t.Space(8), t.Space(1)).Selectable()
	view.CopyData(func(parts []ui.SelectedText) transfer.Data { return transfer.TextData(s.copy(parts)) })
	// The press goes on to select text; whether it was a click shows as it
	// lets go.
	view.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputPointerDown && ev.Button == 0 && ev.Clicks == 1 && ev.Mods&ui.Shift == 0 {
			s.pressed = true
		}
		return false
	})
	s.content = view.LayoutBounds()
	hover, held, next := -1, false, 0
	view.Children(func() {
		for i, seg := range s.segments {
			col := ui.Column(c).Gap(t.Space(1.5))
			s.segs[i] = col.LayoutBounds()
			col.Children(func() {
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Unselectable().Children(func() {
					ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Shrink(0).
						Background(ui.Oklch(0.64, 0.15, hues[s.colors[i]%len(hues)]))
					ui.Text(c, seg.Speaker).Bold().SingleLine().Shrink(1)
					ui.Text(c, clock(seg.Start, s.long)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
				})
				para := ui.RichText(c).FontSize(size).LineHeight(lineHeight).PaddingStart(t.Space(4))
				pr := para.LayoutBounds()
				s.ids[i] = para.ID()
				if _, _, ok := para.Dragged(); ok {
					held = true
				}
				para.Children(func() {
					for k, w := range seg.Words {
						at := s.first[i] + k
						color := t.TextMuted
						if at <= g {
							color = t.Text
						}
						var spans []ui.Span
						spans, next = s.spans(t, i, k, color, lit && at == g, next)
						word := ui.RichText(c, spans...)
						s.words[at] = word.LayoutBounds()
						if word.Hovered() {
							hover = at
						}
						if s.low(w) {
							word.Tooltip("Confidence " + strconv.Itoa(int(math.Round(100*w.Confidence))) + "%")
						}
					}
				})
				para.DrawOver(func(p *ui.Painter, r ui.Rect) {
					// The words' boxes are those of the last frame, which
					// wrapped them in its width.
					if r.W != pr.W {
						c.Invalidate()
						return
					}
					for k, w := range seg.Words {
						if !s.low(w) {
							continue
						}
						b := s.words[s.first[i]+k]
						ww, _ := p.MeasureText(0, ui.Span{Text: strings.TrimRightFunc(w.Text, unicode.IsPunct), Font: t.Font, Size: size})
						y := r.Y + b.Y - pr.Y + (b.H-m.Ascent-m.Descent)/2 + m.Ascent + 2.5
						dots(p, r.X+b.X-pr.X, y, ww, t.TextMuted)
					}
				})
			})
		}
	})
	if !s.pressed || held {
		return false
	}
	s.pressed = false
	if hover < 0 || len(view.SelectedTexts()) > 0 {
		return false
	}
	seek(s.starts[hover])
	return true
}

// low reports whether w shows a dotted underline: the transcription was
// less sure of it than Threshold.
func (s *State) low(w Word) bool { return w.Confidence > 0 && w.Confidence < s.Threshold }

// spans returns the spans of word k of segment i, with the space after
// it, in color: its runes that match the field's text lit, the match gone
// to most, and the word lit with lit. next is the first match that may
// hold the word, and it returns the one for the word after it.
func (s *State) spans(t *ui.Theme, i, k int, color ui.Color, lit bool, next int) ([]ui.Span, int) {
	text := s.texts[s.first[i]+k]
	from := s.at[i][k]
	end := from + utf8.RuneCountInString(text)
	word := from + utf8.RuneCountInString(s.segments[i].Words[k].Text)
	for next < len(s.matches) && (s.matches[next].seg < i || s.matches[next].seg == i && s.matches[next].to <= from) {
		next++
	}
	hit := func(r int) int {
		for j := next; j < len(s.matches) && s.matches[j].seg == i && s.matches[j].from < end; j++ {
			if m := s.matches[j]; r >= m.from && r < m.to {
				return j
			}
		}
		return -1
	}
	// The colors are opaque, so that where the backgrounds of two spans
	// meet, their rounded ends overlapping, shows no seam.
	current := t.Warning.Alpha(0.85).Over(t.Background)
	var out []ui.Span
	var last ui.Color
	start, r := 0, from
	add := func(text string) {
		sp := ui.Span{Text: text, Color: color}
		if last.A > 0 {
			sp.Background, sp.Radius = last, t.Space(1)
		}
		// Light text would not show on the bright match gone to.
		if last == current && t.Dark {
			sp.Color = t.Background
		}
		out = append(out, sp)
	}
	for b := range text {
		bg := ui.Transparent
		switch j := hit(r); {
		case j >= 0 && j == s.match-1:
			bg = current
		case j >= 0:
			bg = t.Warning.Alpha(0.35).Over(t.Background)
		case lit && r < word:
			bg = t.Accent.Alpha(0.3).Over(t.Background)
		}
		if b > 0 && bg != last {
			add(text[start:b])
			start = b
		}
		last = bg
		r++
	}
	add(text[start:])
	return out, next
}

// dots draws a dotted line w long from x at y.
func dots(p *ui.Painter, x, y, w float32, color ui.Color) {
	const d, gap = 1.5, 2
	for dx := float32(0); dx+d <= w; dx += d + gap {
		p.Fill(ui.Rect{X: x + dx, Y: y, W: d, H: d}, color, d/2)
	}
}

// copy returns the parts of the paragraphs selected as Copy puts them on
// the clipboard: the speaker and start of each segment, and below them the
// words selected, a blank line between segments.
func (s *State) copy(parts []ui.SelectedText) string {
	var b strings.Builder
	for _, p := range parts {
		i := slices.Index(s.ids, p.ID)
		runes := []rune(p.Text)
		text := strings.TrimSpace(string(runes[min(p.Start, len(runes)):min(p.End, len(runes))]))
		if i < 0 || text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		seg := s.segments[i]
		b.WriteString(seg.Speaker + "  " + clock(seg.Start, s.long) + "\n" + text)
	}
	return b.String()
}

// clock writes d as 4:05, or as 1:04:05 when long, for a transcript an
// hour long or longer.
func clock(d time.Duration, long bool) string {
	n := int(max(0, d) / time.Second)
	if long {
		return fmt.Sprintf("%d:%02d:%02d", n/3600, n/60%60, n%60)
	}
	return fmt.Sprintf("%d:%02d", n/60, n%60)
}

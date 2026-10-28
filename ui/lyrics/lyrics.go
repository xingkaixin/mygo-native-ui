// Package lyrics shows a song's lyrics in time with it, as music players
// do: the line being sung large and bright, its words lighting up as they
// are sung when the lyrics time them, the lines around it dimmed, and the
// view gliding to keep the sung line a third of the way down. An empty
// line, a break in the singing, shows three dots breathing. Clicking a
// line moves the playhead to it.
//
//	lines, err := lyrics.ParseLRC(lrc)
//	lyrics.View(c, &app.lyrics, lines, &app.playhead)
package lyrics

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Line is a line of a song: when it is sung, its words, and when each
// word is sung when the lyrics time them.
type Line struct {
	At    time.Duration
	Text  string
	Words []Word
}

// Word is a word of a line and when it is sung.
type Word struct {
	At   time.Duration
	Text string
}

// ParseLRC reads lyrics in the LRC format: lines starting with [mm:ss.xx]
// times, and words, in enhanced LRC, each after a <mm:ss.xx> time. Tags
// such as [ar:Artist] are skipped. A line with several times is sung at
// each of them.
func ParseLRC(src string) ([]Line, error) {
	var lines []Line
	for _, raw := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		var times []time.Duration
		rest := strings.TrimSpace(raw)
		for strings.HasPrefix(rest, "[") {
			end := strings.Index(rest, "]")
			if end < 0 {
				break
			}
			at, ok := stamp(rest[1:end])
			if !ok {
				// A tag such as [ti:Title].
				rest = ""
				break
			}
			times = append(times, at)
			rest = rest[end+1:]
		}
		if len(times) == 0 {
			continue
		}
		words, text := parseWords(rest)
		for _, at := range times {
			lines = append(lines, Line{At: at, Text: text, Words: words})
		}
	}
	if len(lines) == 0 {
		return nil, errors.New("lyrics: no timed lines")
	}
	slices.SortStableFunc(lines, func(a, b Line) int { return int(a.At - b.At) })
	return lines, nil
}

// parseWords splits a line into its <mm:ss.xx> timed words, if it has
// any, and returns its text without the times.
func parseWords(s string) ([]Word, string) {
	if !strings.Contains(s, "<") {
		return nil, strings.TrimSpace(s)
	}
	var words []Word
	var text strings.Builder
	for s != "" {
		open := strings.Index(s, "<")
		if open < 0 {
			text.WriteString(s)
			break
		}
		text.WriteString(s[:open])
		end := strings.Index(s[open:], ">")
		if end < 0 {
			break
		}
		at, ok := stamp(s[open+1 : open+end])
		s = s[open+end+1:]
		next := strings.Index(s, "<")
		if next < 0 {
			next = len(s)
		}
		if ok && strings.TrimSpace(s[:next]) != "" {
			words = append(words, Word{At: at, Text: s[:next]})
		}
		text.WriteString(s[:next])
		s = s[next:]
	}
	return words, strings.TrimSpace(text.String())
}

// stamp reads a time written mm:ss.xx.
func stamp(s string) (time.Duration, bool) {
	m, sec, ok := strings.Cut(s, ":")
	if !ok {
		return 0, false
	}
	mins, err1 := strconv.Atoi(m)
	secs, err2 := strconv.ParseFloat(sec, 64)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return time.Duration(mins)*time.Minute + time.Duration(secs*float64(time.Second)), true
}

// State is what a view keeps: where it has scrolled to and where its
// lines were drawn. Keep it in the app's state, one for each view.
type State struct {
	offset float32
	last   time.Time
	tops   []float32
	ready  bool
}

// View creates the lyrics of lines at *playhead, which clicking a line
// moves.
func View(c *ui.Context, s *State, lines []Line, playhead *time.Duration) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Clip().Role(ui.RoleGroup).Label("Lyrics")
	cur := -1
	for i, l := range lines {
		if l.At <= *playhead {
			cur = i
		}
	}
	if view.Clicked() && len(s.tops) == len(lines) {
		if _, py, ok := view.PointerPosition(); ok {
			for i := range lines {
				if py+s.offset >= s.tops[i] && (i+1 == len(lines) || py+s.offset < s.tops[i+1]) {
					*playhead = lines[i].At
				}
			}
		}
	}
	size := t.FontSize + 12
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		pad := t.Space(6)
		width := r.W - 2*pad
		gap := size * 0.7
		s.tops = s.tops[:0]
		y := float32(0)
		heights := make([]float32, len(lines))
		for i, l := range lines {
			s.tops = append(s.tops, y)
			_, h := p.MeasureText(width, ui.Span{Text: measured(l.Text), Size: size, Weight: 700})
			heights[i] = h
			y += h + gap
		}
		// The sung line sits a third of the way down; before the first, the
		// first line does.
		at := max(0, cur)
		target := s.tops[at] + heights[at]/2 - r.H/3
		if !s.ready {
			s.offset, s.ready = target, true
		}
		now := p.Now()
		dt := min(0.1, now.Sub(s.last).Seconds())
		s.last = now
		s.offset += (target - s.offset) * (1 - float32(math.Exp(-dt*8)))
		if math.Abs(float64(target-s.offset)) > 0.5 {
			p.AnimationFrame()
		}
		dim := t.Text.Alpha(0.28)
		for i, l := range lines {
			top := r.Y + s.tops[i] - s.offset
			if top > r.Y+r.H || top+heights[i] < r.Y {
				continue
			}
			if l.Text == "" {
				dots(p, t, r.X+pad, top+heights[i]/2, size, i == cur, p.Now())
				if i == cur {
					p.AnimationFrame()
				}
				continue
			}
			if i != cur || len(l.Words) == 0 {
				col := dim
				if i == cur {
					col = t.Text
				}
				p.RichText(r.X+pad, top, width, ui.Span{Text: l.Text, Size: size, Weight: 700, Color: col})
				continue
			}
			spans := make([]ui.Span, 0, len(l.Words)+1)
			if i := strings.Index(l.Text, strings.TrimSpace(l.Words[0].Text)); i > 0 {
				spans = append(spans, ui.Span{Text: l.Text[:i], Size: size, Weight: 700, Color: t.Text})
			}
			for _, w := range l.Words {
				col := t.Text.Alpha(0.45)
				if w.At <= *playhead {
					col = t.Text
				}
				spans = append(spans, ui.Span{Text: w.Text, Size: size, Weight: 700, Color: col})
			}
			p.RichText(r.X+pad, top, width, spans...)
		}
	})
	return view
}

// measured gives an empty line the height of a line of text.
func measured(s string) string {
	if s == "" {
		return "•"
	}
	return s
}

// dots paints three dots that breathe while lit.
func dots(p *ui.Painter, t *ui.Theme, x, y, size float32, lit bool, now time.Time) {
	for k := range 3 {
		scale, alpha := float32(1), float32(0.28)
		if lit {
			ph := float64(now.UnixMilli())/1000*math.Pi*1.6 - float64(k)*0.7
			scale = 0.8 + 0.25*float32(math.Sin(ph))
			alpha = 0.6 + 0.4*float32(math.Sin(ph))
		}
		rad := size * 0.14 * scale
		var dot ui.Path
		p.FillPath(dot.Circle(x+size*0.2+float32(k)*size*0.5, y, rad), t.Text.Alpha(alpha))
	}
}

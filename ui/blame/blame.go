// Package blame shows who last changed each line of a file, as git blame
// tells it and code hosts show it: the lines with their numbers,
// highlighted as the editor highlights their language, and by the first
// line of each run of lines from one commit, its summary, its author and
// how long ago it was made. A bar by each run shows the commit's age, the
// newest strongest. Resting the pointer on a run's gutter shows a card of
// its commit: its hash, author, date and message. A click on a line
// chooses its commit, lighting all its lines, and sets *chosen to its hash
// in the same frame:
//
//	out, err := exec.Command("git", "blame", "--porcelain", "--", path).Output()
//	app.lines = blame.Parse(string(out)) // once, not every frame
//	...
//	blame.View(c, &app.blame, path, app.lines, &app.chosen).Grow(1)
//	if app.chosen != app.shown {
//		... // show the commit
//	}
//
// It builds only the lines in view, as ui.List does, so files of any
// length scroll as fast as short ones.
package blame

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Commit is a commit that last changed lines of a file: its hash, its
// author, when it was made, in the author's time zone, and the first line
// of its message. Message is the whole message, which git blame does not
// print: the app fills it in, as from git show, and cards show Summary
// while it is empty.
type Commit struct {
	Hash    string
	Author  string
	Time    time.Time
	Summary string
	Message string
}

// Line is a line of the file and the commit that last changed it, which
// the lines from one commit share.
type Line struct {
	Text   string
	Commit *Commit
}

// Parse reads the lines of a file and their commits from what git blame
// --porcelain printed.
func Parse(porcelain string) []Line {
	var lines []Line
	commits := map[string]*Commit{}
	var cur *Commit
	for l := range strings.SplitSeq(porcelain, "\n") {
		if text, ok := strings.CutPrefix(l, "\t"); ok {
			if cur != nil {
				lines = append(lines, Line{Text: strings.TrimSuffix(text, "\r"), Commit: cur})
			}
			continue
		}
		key, value, _ := strings.Cut(l, " ")
		if hash(key) {
			if cur = commits[key]; cur == nil {
				cur = &Commit{Hash: key}
				commits[key] = cur
			}
			continue
		}
		if cur == nil {
			continue
		}
		switch key {
		case "author":
			cur.Author = value
		case "author-time":
			at, _ := strconv.ParseInt(value, 10, 64)
			cur.Time = time.Unix(at, 0)
		case "author-tz":
			if tz, err := time.Parse("-0700", value); err == nil {
				cur.Time = cur.Time.In(tz.Location())
			}
		case "summary":
			cur.Summary = value
		}
	}
	return lines
}

// hash reports whether s is a commit's hash, of SHA-1 or SHA-256.
func hash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !('0' <= r && r <= '9' || 'a' <= r && r <= 'f') {
			return false
		}
	}
	return true
}

// State is what a view keeps: its place in the file, the language of the
// file, the tokens of the lines highlighted so far, and the times of the
// oldest and newest commits. Keep it in the app's state, one for each
// view.
type State struct {
	list           ui.ListState
	lines          []Line
	name           string
	lang           *editor.Language
	tokens         [][]editor.Token
	next           uint64
	oldest, newest time.Time
}

// prepare starts over for the lines of the file name.
func (s *State) prepare(name string, lines []Line) {
	s.lines, s.name, s.lang = lines, name, editor.LanguageFor(name)
	s.tokens, s.next = nil, 0
	s.oldest, s.newest = time.Time{}, time.Time{}
	for _, l := range lines {
		if s.oldest.IsZero() || l.Commit.Time.Before(s.oldest) {
			s.oldest = l.Commit.Time
		}
		if l.Commit.Time.After(s.newest) {
			s.newest = l.Commit.Time
		}
	}
}

// tokensOf returns the tokens of line i, tokenizing the lines up to it that
// are not yet, as each starts in the state the one before ended in.
func (s *State) tokensOf(i int) []editor.Token {
	if s.lang == nil || s.lang.Tokenizer == nil {
		return nil
	}
	for len(s.tokens) <= i {
		var toks []editor.Token
		toks, s.next = s.lang.Tokenizer.Tokenize(s.lines[len(s.tokens)].Text, s.next, nil)
		s.tokens = append(s.tokens, toks)
	}
	return s.tokens[i]
}

// age returns how new cm is among the commits of the file, from 0 for the
// oldest to 1 for the newest.
func (s *State) age(cm *Commit) float32 {
	span := s.newest.Sub(s.oldest)
	if span <= 0 {
		return 1
	}
	return float32(cm.Time.Sub(s.oldest)) / float32(span)
}

// lineHeight is the height of a line of code, in sizes of its font.
const lineHeight = 1.6

// View creates the lines of the file name, as "main.go", whose language
// highlights them, and the commits that last changed them; a click on a
// line sets *chosen to the hash of its commit, which lights its lines. Give
// it a size, or Grow.
func View(c *ui.Context, s *State, name string, lines []Line, chosen *string) ui.Element {
	t := c.Theme()
	if !same(s.lines, lines) || s.name != name {
		s.prepare(name, lines)
	}
	ct := editor.DefaultTheme(t)
	size := t.FontSize - 1
	rowH := size * lineHeight
	numW, _ := c.MeasureText(0, ui.Span{Text: strings.Repeat("0", len(strconv.Itoa(len(lines)))), Font: "monospace", Size: size})
	now := c.Now()
	return ui.List(c, &s.list, len(lines), func(i int) {
		ln := lines[i]
		first := i == 0 || lines[i-1].Commit != ln.Commit
		row := ui.Row(c).MinHeight(rowH).AlignItems(ui.Start)
		if row.Clicked() {
			*chosen = ln.Commit.Hash
		}
		if ln.Commit.Hash == *chosen {
			row.Background(t.Accent.Alpha(0.12))
		}
		if first && i > 0 {
			row.BorderWidth(1, 0, 0, 0).BorderColor(t.Border.Alpha(0.6))
		}
		row.Children(func() {
			gutter(c, s, ln.Commit, first, rowH, now)
			ui.Text(c, strconv.Itoa(i+1)).Font("monospace").FontSize(size).LineHeight(lineHeight).FontFeatures("tnum").
				TextColor(ct.LineNumber).TextAlign(ui.End).Width(numW).Shrink(0).MarginStart(t.Space(3))
			ui.RichText(c, spans(ln.Text, s.tokensOf(i), ct)...).Font("monospace").FontSize(size).LineHeight(lineHeight).
				Grow(1).Shrink(1).Padding(0, t.Space(4))
		})
	})
}

// same reports whether a and b are the same lines, as the app passes them
// again from frame to frame.
func same(a, b []Line) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// gutter shows by a line of cm the bar of its age and, on the first line
// of a run, its summary, author and how long before now it was made; the
// pointer resting on it shows the card of cm.
func gutter(c *ui.Context, s *State, cm *Commit, first bool, rowH float32, now time.Time) {
	t := c.Theme()
	cell := ui.Row(c).Width(t.Space(100)).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Start).Description(cm.Summary)
	cell.Children(func() {
		bar := ui.Box(c).Width(3).Shrink(0).AlignSelf(ui.Stretch).Background(heat(t, s.age(cm)))
		if first {
			bar.Margin(2, 0, 0, 0)
		}
		if !first {
			return
		}
		ui.Row(c).Height(rowH).Grow(1).Shrink(1).Gap(t.Space(2)).Padding(0, t.Space(2.5)).AlignItems(ui.Center).
			FontSize(t.FontSize - 1).Children(func() {
			ui.Text(c, cm.Summary).SingleLine().Grow(1).Shrink(1)
			ui.Text(c, cm.Author).TextColor(t.TextMuted).SingleLine().Width(t.Space(20)).Shrink(1)
			ui.Text(c, ago(now, cm.Time)).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().
				Width(t.Space(24)).Shrink(0).TextAlign(ui.End)
		})
	})
	ui.TooltipBase(c, cell, func(tip ui.Element) {
		tip.AttachTo(cell, ui.AnchorBottomStart, ui.AnchorTopStart).Margin(t.Space(1), 0, 0, t.Space(2)).
			MaxWidth(t.Space(110)).Padding(t.Space(3)).Radius(t.Space(2.5)).
			Background(t.Background).TextColor(t.Text).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGB(0, 0, 0).Alpha(0.14))
		tip.Children(func() { card(c, cm, now) })
	})
}

// card shows cm: its author and date, its message and its hash.
func card(c *ui.Context, cm *Commit, now time.Time) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, cm.Author).Bold().SingleLine().Shrink(1)
			ui.Text(c, cm.Time.Format("Jan 2, 2006 15:04")+" · "+ago(now, cm.Time)).FontSize(t.FontSize - 1).
				TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
		})
		message := strings.TrimSpace(cm.Message)
		if message == "" {
			message = cm.Summary
		}
		ui.Text(c, message)
		ui.Text(c, cm.Hash).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
	})
}

// heat returns the color of the bar of a commit as new as f, from 0 for
// the oldest to 1 for the newest.
func heat(t *ui.Theme, f float32) ui.Color {
	return t.Warning.Alpha(0.25 + 0.75*f)
}

// spans returns text in spans of its tokens' colors, tabs set to every
// fourth column.
func spans(text string, tokens []editor.Token, ct *editor.Theme) []ui.Span {
	if text == "" {
		return []ui.Span{{Text: " "}}
	}
	var out []ui.Span
	col, start := 0, 0
	add := func(end int, st editor.TokenStyle) {
		if end <= start {
			return
		}
		var b strings.Builder
		for _, r := range text[start:end] {
			if r == '\t' {
				n := 4 - col%4
				b.WriteString("    "[:n])
				col += n
				continue
			}
			b.WriteRune(r)
			col++
		}
		sp := ui.Span{Text: b.String(), Color: st.Color, Italic: st.Italic}
		if st.Bold {
			sp.Weight = 700
		}
		out = append(out, sp)
		start = end
	}
	for _, tok := range tokens {
		add(min(tok.End, len(text)), ct.Tokens[tok.Kind])
	}
	add(len(text), editor.TokenStyle{})
	return out
}

// ago returns how long before now t was: "just now", "5 minutes ago",
// "3 months ago".
func ago(now, t time.Time) string {
	const day, year = 24 * time.Hour, 365 * 24 * time.Hour
	d := now.Sub(t)
	unit := func(n time.Duration, word string) string {
		if n <= 1 {
			return "1 " + word + " ago"
		}
		return strconv.Itoa(int(n)) + " " + word + "s ago"
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return unit(d/time.Minute, "minute")
	case d < day:
		return unit(d/time.Hour, "hour")
	case d < 7*day:
		return unit(d/day, "day")
	case d < 30*day:
		return unit(d/(7*day), "week")
	case d < year:
		return unit(d/(year/12), "month")
	}
	return unit(d/year, "year")
}

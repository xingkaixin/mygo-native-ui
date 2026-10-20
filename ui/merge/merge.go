// Package merge resolves the conflicts git leaves in a file, as editors'
// merge views do. Each conflict is a card with the Current lines, ours,
// and the Incoming lines, theirs, side by side, and the base between them
// when git wrote diff3's, the words that differ highlighted; its buttons
// accept Current, Incoming or both, either first, or open the result to
// edit by hand, and Reset brings the conflict back. The unchanged lines
// between conflicts fold to a few around each, and a click shows the rest.
// A bar counts the conflicts left, "2 of 5 conflicts left", and goes to
// the next and previous of them with its buttons, F7 and Shift+F7.
//
//	merge.View(c, &app.merge, app.text, editor.LanguageFor(app.path)).Grow(1)
//	if text, ok := app.merge.Resolved(); ok && ui.Button(c, "Save").Clicked() {
//		os.WriteFile(app.path, []byte(text), 0o644)
//	}
package merge

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// around is how many unchanged lines show on each side of a conflict.
const around = 3

// The sides of a conflict, in the order they show.
const (
	current = iota
	base
	incoming
)

var sideNames = [3]string{"Current", "Base", "Incoming"}

// took is how a conflict was resolved.
type took uint8

const (
	open took = iota
	tookCurrent
	tookIncoming
	tookCurrentFirst
	tookIncomingFirst
	tookEdit
)

var tookNames = [...]string{"", "Accepted Current", "Accepted Incoming", "Accepted Current, then Incoming", "Accepted Incoming, then Current", "Edited"}

// State is what a view keeps: the conflicts of its text and how each was
// resolved, the conflict gone to, and the unchanged lines shown. Keep it
// in the app's state, one for each file. Another text starts over.
type State struct {
	text   string
	lang   *editor.Language
	parsed bool
	eol    string
	// newline is whether the text ends in a line ending.
	newline   bool
	parts     []part
	conflicts []*conflict
	shown     map[int]bool
	at        int
	scroll    bool
	focus     bool
	digits    int
}

// part is a run of unchanged lines, or a conflict.
type part struct {
	same []line
	c    *conflict
}

type conflict struct {
	sides [3]side
	base  bool
	took  took
	edit  *editor.State
}

type side struct {
	label string
	lines []line
}

// line is a line of the text, n its number from 1, with its tokens and the
// bytes that differ from the other sides.
type line struct {
	text string
	n    int
	toks []editor.Token
	mark []bool
}

// Resolved returns the text with each conflict replaced by what the user
// chose for it, after View, or false while conflicts are left.
func (s *State) Resolved() (string, bool) {
	if !s.parsed {
		return "", false
	}
	var b strings.Builder
	for _, p := range s.parts {
		if p.c == nil {
			for _, l := range p.same {
				b.WriteString(l.text + s.eol)
			}
			continue
		}
		if p.c.took == open {
			return "", false
		}
		for _, l := range p.c.result() {
			b.WriteString(l + s.eol)
		}
	}
	out := b.String()
	if !s.newline {
		out = strings.TrimSuffix(out, s.eol)
	}
	return out, true
}

// View creates the view of text, a file with git's conflict markers, its
// code highlighted as lang says, or plain when nil. Give it a height, or
// Grow.
func View(c *ui.Context, s *State, text string, lang *editor.Language) ui.Element {
	if !s.parsed || text != s.text || lang != s.lang {
		s.parse(text, lang)
	}
	t := c.Theme()
	th := editor.DefaultTheme(t)
	left := 0
	for _, k := range s.conflicts {
		if k.took == open {
			left++
		}
	}
	if c.Shortcut(0, ui.KeyF7) {
		s.step(1)
	}
	if c.Shortcut(ui.Shift, ui.KeyF7) {
		s.step(-1)
	}
	return ui.Column(c).Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
			n := len(s.conflicts)
			switch {
			case n == 0:
				ui.Text(c, "No conflicts").TextColor(t.TextMuted).Grow(1)
			case left == 0:
				status(c, true)
				ui.Text(c, "All conflicts resolved").Bold().TextColor(t.Success).Grow(1)
			default:
				noun := "conflicts"
				if n == 1 {
					noun = "conflict"
				}
				ui.Text(c, fmt.Sprintf("%d of %d %s left", left, n, noun)).Bold().FontFeatures("tnum").Grow(1)
			}
			if ui.Button(c, "Previous").Tooltip("Shift+F7").Disabled(left == 0).Clicked() {
				s.step(-1)
			}
			if ui.Button(c, "Next").Tooltip("F7").Disabled(left == 0).Clicked() {
				s.step(1)
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(2), t.Space(3), t.Space(6)).Children(func() {
				n := 0
				for i, p := range s.parts {
					if p.c == nil {
						s.same(c, th, i, p.same)
						continue
					}
					s.card(c, th, n, p.c)
					n++
				}
			})
		})
		s.scroll = false
	})
}

// step goes to the next conflict left after the one gone to, for d 1, or
// the one before, for d -1, coming round at the ends.
func (s *State) step(d int) {
	n := len(s.conflicts)
	for k := 1; k <= n; k++ {
		i := ((s.at+d*k)%n + n) % n
		if s.conflicts[i].took == open {
			s.at, s.scroll = i, true
			return
		}
	}
}

// same shows the unchanged lines of part i, those more than around from a
// conflict folded into a bar that shows them at a click.
func (s *State) same(c *ui.Context, th *editor.Theme, i int, lines []line) {
	t := c.Theme()
	head, tail := 0, 0
	if i > 0 {
		head = around
	}
	if i < len(s.parts)-1 {
		tail = around
	}
	hidden := len(lines) - head - tail
	if s.shown[i] || hidden <= 1 {
		for _, l := range lines {
			s.row(c, th, l, ui.Color{}, ui.Color{})
		}
		return
	}
	for _, l := range lines[:head] {
		s.row(c, th, l, ui.Color{}, ui.Color{})
	}
	noun := "lines"
	if hidden == 1 {
		noun = "line"
	}
	label := fmt.Sprintf("%d unchanged %s", hidden, noun)
	fold := ui.ButtonBase(c).Justify(ui.Start).Margin(t.Space(1), 0).Padding(t.Space(1), 0).Radius(t.Space(1)).
		Background(t.Text.Alpha(0.04)).Label("Show " + label)
	if fold.Hovered() {
		fold.Background(t.Text.Alpha(0.08))
	}
	if fold.Clicked() {
		s.shown[i] = true
	}
	fold.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "⋯").Width(s.gutter(t)).TextAlign(ui.End).TextColor(t.TextMuted)
			ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
	})
	for _, l := range lines[len(lines)-tail:] {
		s.row(c, th, l, ui.Color{}, ui.Color{})
	}
}

// card shows conflict k, the nth: its sides while open, else what it
// took, and the buttons that resolve it.
func (s *State) card(c *ui.Context, th *editor.Theme, n int, k *conflict) {
	t := c.Theme()
	colors := sideColors(t)
	border := t.Border
	if n == s.at {
		border = t.Accent
	}
	card := ui.Column(c).Margin(t.Space(2), 0).Radius(t.Space(2)).Border(1, border).Clip().
		Role(ui.RoleGroup).Label(fmt.Sprintf("Conflict %d", n+1))
	if s.scroll && n == s.at {
		card.ScrollIntoView()
	}
	take := func(how took) {
		k.took, s.at = how, n
	}
	card.Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Background(t.Text.Alpha(0.03)).Children(func() {
			status(c, k.took != open)
			ui.Text(c, fmt.Sprintf("Conflict %d", n+1)).Bold()
			ui.Text(c, tookNames[k.took]).TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1)
			if k.took == open {
				if ui.Button(c, "Accept Current").Clicked() {
					take(tookCurrent)
				}
				if ui.Button(c, "Accept Incoming").Clicked() {
					take(tookIncoming)
				}
				ui.MenuButton(c, "Accept Both", func(m *ui.Menu) {
					if m.Item("Current, then Incoming").Chosen() {
						take(tookCurrentFirst)
					}
					if m.Item("Incoming, then Current").Chosen() {
						take(tookIncomingFirst)
					}
				})
			}
			if k.took != tookEdit && ui.Button(c, "Edit").Clicked() {
				s.startEdit(t, k)
				s.focus = true
				take(tookEdit)
			}
			if k.took != open && ui.Button(c, "Reset").Clicked() {
				k.edit = nil
				take(open)
			}
		})
		ui.Divider(c)
		switch k.took {
		case open:
			ui.Row(c).AlignItems(ui.Stretch).Children(func() {
				for i := range k.sides {
					if i == base && !k.base {
						continue
					}
					if i > current {
						ui.Box(c).Width(1).Background(t.Border)
					}
					sd := k.sides[i]
					ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
						ui.Row(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
							ui.Text(c, sideNames[i]).FontSize(t.FontSize - 1).FontWeight(600).TextColor(colors[i])
							ui.Text(c, sd.label).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Shrink(1)
						})
						for _, l := range sd.lines {
							s.row(c, th, l, colors[i].Alpha(0.08), colors[i].Alpha(0.28))
						}
						if len(sd.lines) == 0 {
							empty(c)
						}
					})
				}
			})
		case tookEdit:
			ui.Row(c).Padding(t.Space(1), 0).Children(func() {
				// The editor sets its text 8 DIPs in when it has no gutter.
				ui.Box(c).Width(s.gutter(t) + t.Space(3) - 8)
				ed := editor.View(c, k.edit).Grow(1)
				// AutoFocus loses to the Edit button just clicked, which then
				// goes, leaving the focus nowhere.
				if s.focus {
					ed.Focus()
					s.focus = false
				}
			})
		default:
			ui.Column(c).Padding(t.Space(1), 0).Children(func() {
				for _, i := range k.order() {
					for _, l := range k.sides[i].lines {
						l.n, l.mark = 0, nil
						s.row(c, th, l, colors[i].Alpha(0.08), ui.Color{})
					}
				}
				if len(k.result()) == 0 {
					empty(c)
				}
			})
		}
	})
}

// startEdit gives k an editor of what it took, or of both sides while
// open.
func (s *State) startEdit(t *ui.Theme, k *conflict) {
	buf := editor.NewBuffer(strings.Join(k.result(), "\n"))
	buf.SetLanguage(s.lang)
	k.edit = &editor.State{
		Buffer:      buf,
		Font:        ui.Font{Family: "monospace", Size: t.FontSize},
		FitContent:  true,
		LineNumbers: editor.LineNumbersOff,
		NoFolding:   true,
		Label:       "Resolved lines",
	}
}

// row shows l, its number in the gutter, on bg, with the bytes marked on
// mark.
func (s *State) row(c *ui.Context, th *editor.Theme, l line, bg, mark ui.Color) {
	t := c.Theme()
	r := ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Start).Font("monospace").FontSize(t.FontSize).LineHeight(1.5)
	if bg != (ui.Color{}) {
		r.Background(bg)
	}
	r.Children(func() {
		num := ""
		if l.n > 0 {
			num = strconv.Itoa(l.n)
		}
		ui.Text(c, num).Width(s.gutter(t)).TextAlign(ui.End).TextColor(th.LineNumber)
		ui.RichText(c, s.spans(th, l, mark)...).Grow(1).Shrink(1).MinWidth(0).Padding(0, t.Space(3), 0, 0)
	})
}

// spans splits l into runs of one token kind and one mark.
func (s *State) spans(th *editor.Theme, l line, mark ui.Color) []ui.Span {
	tab := "    "
	if s.lang != nil && s.lang.TabSize > 0 {
		tab = strings.Repeat(" ", s.lang.TabSize)
	}
	var out []ui.Span
	ti := 0
	for pos := 0; pos < len(l.text); {
		for ti < len(l.toks) && l.toks[ti].End <= pos {
			ti++
		}
		end, kind := len(l.text), editor.TokenText
		if ti < len(l.toks) {
			end, kind = min(l.toks[ti].End, len(l.text)), l.toks[ti].Kind
		}
		marked := l.mark != nil && l.mark[pos]
		if l.mark != nil {
			for e := pos + 1; e < end; e++ {
				if l.mark[e] != marked {
					end = e
					break
				}
			}
		}
		st := th.Tokens[kind]
		sp := ui.Span{Text: strings.ReplaceAll(l.text[pos:end], "\t", tab), Color: th.TokenColor(kind), Italic: st.Italic}
		if st.Bold {
			sp.Weight = 600
		}
		if marked && mark != (ui.Color{}) {
			sp.Background = mark
		}
		out = append(out, sp)
		pos = end
	}
	if len(out) == 0 {
		// A span of nothing lays out no line, which an empty line needs.
		out = append(out, ui.Span{Text: " "})
	}
	return out
}

// gutter is the width of the line numbers.
func (s *State) gutter(t *ui.Theme) float32 {
	return float32(max(s.digits, 2))*t.FontSize*0.62 + t.Space(2)
}

// status shows a dot for a conflict open, or a check for one resolved.
func status(c *ui.Context, resolved bool) {
	t := c.Theme()
	ui.Box(c).Size(t.FontSize, t.FontSize).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.3
		if !resolved {
			var dot ui.Path
			p.FillPath(dot.Circle(cx, cy, r.W*0.25), t.Warning)
			return
		}
		p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, t.Success)
	})
}

// empty shows a side, or a result, of no lines.
func empty(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, "No lines").FontSize(t.FontSize-1).Italic().TextColor(t.TextMuted).Padding(t.Space(1), t.Space(3))
}

// sideColors returns the colors of Current, the base and Incoming.
func sideColors(t *ui.Theme) [3]ui.Color {
	blue := ui.Oklch(0.58, 0.16, 255)
	if t.Dark {
		blue = ui.Oklch(0.72, 0.13, 250)
	}
	return [3]ui.Color{t.Success, t.TextMuted, blue}
}

// order returns the sides whose lines k took, in order, or both while
// open, as an edit starts from.
func (k *conflict) order() []int {
	switch k.took {
	case tookCurrent:
		return []int{current}
	case tookIncoming:
		return []int{incoming}
	case tookIncomingFirst:
		return []int{incoming, current}
	}
	return []int{current, incoming}
}

// result returns the lines k resolves to.
func (k *conflict) result() []string {
	if k.took == tookEdit {
		text := k.edit.Buffer.String()
		if text == "" {
			return nil
		}
		return strings.Split(text, "\n")
	}
	var out []string
	for _, i := range k.order() {
		for _, l := range k.sides[i].lines {
			out = append(out, l.text)
		}
	}
	return out
}

// parse splits text into unchanged lines and conflicts, and tokenizes
// them as lang does.
func (s *State) parse(text string, lang *editor.Language) {
	*s = State{text: text, lang: lang, parsed: true, eol: "\n", shown: map[int]bool{}}
	if text == "" {
		return
	}
	if strings.Contains(text, "\r\n") {
		s.eol = "\r\n"
	}
	raw := strings.Split(text, "\n")
	if s.newline = strings.HasSuffix(text, "\n"); s.newline {
		raw = raw[:len(raw)-1]
	}
	for i := range raw {
		raw[i] = strings.TrimSuffix(raw[i], "\r")
	}
	s.digits = len(strconv.Itoa(len(raw)))
	var tok editor.Tokenizer
	if lang != nil {
		tok = lang.Tokenizer
	}
	lines := func(from, to int, state uint64) ([]line, uint64) {
		out := make([]line, 0, to-from)
		for i := from; i < to; i++ {
			l := line{text: raw[i], n: i + 1}
			if tok != nil {
				l.toks, state = tok.Tokenize(l.text, state, nil)
			}
			out = append(out, l)
		}
		return out, state
	}
	var state uint64
	start := 0
	for i := 0; i < len(raw); i++ {
		label, ok := marker(raw[i], "<<<<<<<")
		if !ok {
			continue
		}
		b, ok := bounds(raw, i)
		if !ok {
			continue
		}
		if start < i {
			var same []line
			same, state = lines(start, i, state)
			s.parts = append(s.parts, part{same: same})
		}
		k := &conflict{base: b[0] >= 0}
		curEnd := b[1]
		if k.base {
			curEnd = b[0]
			k.sides[base].label, _ = marker(raw[b[0]], "|||||||")
			k.sides[base].lines, _ = lines(b[0]+1, b[1], state)
		}
		k.sides[incoming].label, _ = marker(raw[b[2]], ">>>>>>>")
		k.sides[incoming].lines, _ = lines(b[1]+1, b[2], state)
		k.sides[current].label = label
		k.sides[current].lines, state = lines(i+1, curEnd, state)
		k.diff()
		s.parts = append(s.parts, part{c: k})
		s.conflicts = append(s.conflicts, k)
		i, start = b[2], b[2]+1
	}
	if start < len(raw) {
		same, _ := lines(start, len(raw), state)
		s.parts = append(s.parts, part{same: same})
	}
}

// marker reports whether l is the marker m, and returns its label.
func marker(l, m string) (string, bool) {
	if l != m && !strings.HasPrefix(l, m+" ") {
		return "", false
	}
	return strings.TrimSpace(l[len(m):]), true
}

// bounds returns the lines of the base marker, -1 without one, the
// separator and the end marker of the conflict starting at line i, or
// false when it does not end.
func bounds(raw []string, i int) ([3]int, bool) {
	b := [3]int{-1, -1, -1}
	is := func(l, m string) bool { _, ok := marker(l, m); return ok }
	for j := i + 1; j < len(raw); j++ {
		switch l := raw[j]; {
		case is(l, "<<<<<<<"):
			return b, false
		case b[0] < 0 && b[1] < 0 && is(l, "|||||||"):
			b[0] = j
		case b[1] < 0 && is(l, "======="):
			b[1] = j
		case b[1] >= 0 && is(l, ">>>>>>>"):
			b[2] = j
			return b, true
		}
	}
	return b, false
}

// diff marks the words that differ: between Current and Incoming, or with
// a base, between each of them and the base, which shows what either
// changed.
func (k *conflict) diff() {
	cur, inc := k.sides[current].lines, k.sides[incoming].lines
	if !k.base {
		markLines(cur, inc)
		return
	}
	markLines(cur, k.sides[base].lines)
	markLines(inc, k.sides[base].lines)
}

// markLines marks the words of a and b that the other has not, comparing
// the lines that differ word by word.
func markLines(a, b []line) {
	ta, tb := make([]string, len(a)), make([]string, len(b))
	for i, l := range a {
		ta[i] = l.text
	}
	for i, l := range b {
		tb[i] = l.text
	}
	i, j := 0, 0
	for _, p := range append(common(ta, tb), [2]int{len(a), len(b)}) {
		markWords(a[i:p[0]], b[j:p[1]])
		i, j = p[0]+1, p[1]+1
	}
}

// word is a run of letters and digits, of spaces, or another character,
// or the end of a line.
type word struct {
	text             string
	line, start, end int
}

func markWords(a, b []line) {
	wa, wb := words(a), words(b)
	ta, tb := make([]string, len(wa)), make([]string, len(wb))
	for i, w := range wa {
		ta[i] = w.text
	}
	for i, w := range wb {
		tb[i] = w.text
	}
	sa, sb := make([]bool, len(wa)), make([]bool, len(wb))
	for _, p := range common(ta, tb) {
		sa[p[0]], sb[p[1]] = true, true
	}
	paint(a, wa, sa)
	paint(b, wb, sb)
}

// paint marks in lines the words not the same, and the spaces between
// two of them, so that a phrase changed reads as one.
func paint(lines []line, ws []word, same []bool) {
	differs := func(i int) bool { return i >= 0 && i < len(ws) && !same[i] && ws[i].text != "\n" }
	for i, w := range ws {
		if !differs(i) {
			continue
		}
		if strings.TrimSpace(w.text) == "" && !(differs(i-1) && differs(i+1)) {
			continue
		}
		l := &lines[w.line]
		if l.mark == nil {
			l.mark = make([]bool, len(l.text))
		}
		for b := w.start; b < w.end; b++ {
			l.mark[b] = true
		}
	}
	for i := range lines {
		whole(&lines[i])
	}
}

// whole marks all of l but its indentation when most of it is marked, as
// a line rewritten reads better whole than in the scraps it shares.
func whole(l *line) {
	if l.mark == nil {
		return
	}
	marked, all := 0, 0
	for b := range len(l.text) {
		if l.text[b] != ' ' && l.text[b] != '\t' {
			all++
			if l.mark[b] {
				marked++
			}
		}
	}
	if marked*2 <= all {
		return
	}
	start := len(l.text) - len(strings.TrimLeft(l.text, " \t"))
	for b := start; b < len(l.text); b++ {
		l.mark[b] = true
	}
}

func words(lines []line) []word {
	var ws []word
	class := func(r rune) int {
		switch {
		case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			return 1
		case unicode.IsSpace(r):
			return 2
		}
		return 0
	}
	for li, l := range lines {
		for i := 0; i < len(l.text); {
			r, n := utf8.DecodeRuneInString(l.text[i:])
			j := i + n
			if k := class(r); k != 0 {
				for j < len(l.text) {
					r, n := utf8.DecodeRuneInString(l.text[j:])
					if class(r) != k {
						break
					}
					j += n
				}
			}
			ws = append(ws, word{l.text[i:j], li, i, j})
			i = j
		}
		ws = append(ws, word{"\n", li, len(l.text), len(l.text)})
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

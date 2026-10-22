// Package hunks reviews the changes to a file one hunk at a time, as coding
// agents ask about each of their edits, Accept or Reject, and as git
// clients stage changes, Stage or Discard. Each hunk of a unified diff is a
// card of its lines, numbered as in the old file and the new, those removed
// and added tinted, under its header's range and function, with buttons
// that decide it; a decided hunk folds to its header, which shows the
// decision and, at a click, the lines again, dimmed. A click on a changed
// line chooses it, and Shift-click and dragging choose the lines up to it,
// for only those to be accepted. A bar counts the hunks left, "3 of 7 hunks
// left", and accepts or rejects them all. While the view has the keyboard
// focus, j and k or the arrows go to the next and previous hunk, y accepts
// it, or its lines chosen, n rejects it, u undoes the last decision and
// Escape lets the lines chosen go.
//
//	hunks.View(c, &app.hunks, app.diff, &app.decisions, hunks.Stage).Grow(1)
//	if ui.Button(c, "Apply").Clicked() {
//		cmd := exec.Command("git", "apply", "--cached")
//		cmd.Stdin = strings.NewReader(hunks.Patch(app.diff, app.decisions))
//		app.err = cmd.Run()
//	}
package hunks

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Verdict is what the user decided for a hunk.
type Verdict uint8

const (
	Undecided Verdict = iota
	Accepted
	Rejected
)

// Decision is the user's decision on a hunk: its verdict and, for a hunk
// accepted in part, the lines accepted, as indexes into the hunk's Lines
// that editor.ParsePatch reads. No Lines accepts all of them.
type Decision struct {
	Verdict Verdict
	Lines   []int
}

// Labels are the words of a view's buttons and of the decisions it shows.
type Labels struct {
	Accept, Reject     string
	Accepted, Rejected string
}

// Review words the decisions on an agent's edits, Stage those of a git
// client staging changes. The zero Labels are Review's.
var (
	Review = Labels{Accept: "Accept", Reject: "Reject", Accepted: "Accepted", Rejected: "Rejected"}
	Stage  = Labels{Accept: "Stage", Reject: "Discard", Accepted: "Staged", Rejected: "Discarded"}
)

// lineHeight is the height of a line of code, in sizes of its font.
const lineHeight = 1.6

// State is what a view keeps: the hunks of its diff, the hunk gone to, the
// lines chosen, the decided hunks opened and the decisions to undo. Keep it
// in the app's state, one for each view. Another diff starts over.
type State struct {
	diff   string
	parsed bool
	file   editor.FilePatch
	err    error
	toks   [][][]editor.Token
	digits int
	at     int
	scroll bool
	focus  bool
	pick   pick
	opened map[int]bool
	undo   [][]change
}

// pick is the lines chosen: the changed ones of hunk h from line from to
// line to, either first, while on.
type pick struct {
	on          bool
	h, from, to int
}

// change is a decision made, for u to undo: the hunk's and what it was.
type change struct {
	h   int
	was Decision
}

// View creates the view of diff, a unified diff of one file as git diff
// prints it, its file headers optional, and of the decisions on its hunks,
// which it makes one for each hunk, all Undecided, when *decisions holds
// another number or diff is not the last one. Labels word the buttons.
// Changed reports a decision made or undone. Give it a height, or Grow.
func View(c *ui.Context, s *State, diff string, decisions *[]Decision, labels Labels) ui.Element {
	if !s.parsed || diff != s.diff {
		if s.parsed {
			*decisions = nil
		}
		s.parse(diff)
	}
	if labels == (Labels{}) {
		labels = Review
	}
	n := len(s.file.Hunks)
	if len(*decisions) != n {
		*decisions = make([]Decision, n)
	}
	d := *decisions
	t := c.Theme()
	ct := editor.DefaultTheme(t)
	changed := false
	decide := func(v Decision, hs ...int) {
		if s.decide(d, v, hs...) {
			changed = true
		}
	}
	act := func(h int, v Verdict) {
		switch v {
		case Accepted:
			lines := s.chosen(h)
			if len(lines) == changes(s.file.Hunks[h]) {
				lines = nil
			}
			decide(Decision{Verdict: Accepted, Lines: lines}, h)
			s.next(d, h)
		case Rejected:
			decide(Decision{Verdict: Rejected}, h)
			s.next(d, h)
		default:
			decide(Decision{}, h)
			s.at = h
		}
	}
	root := ui.Column(c).Background(t.Background).Focusable().FocusRing(false).Role(ui.RoleGroup).Label("Hunks")
	if n > 0 {
		for _, k := range []struct {
			key ui.Key
			d   int
		}{{ui.KeyJ, 1}, {ui.KeyDown, 1}, {ui.KeyK, -1}, {ui.KeyUp, -1}} {
			if root.Shortcut(0, k.key) {
				s.at, s.scroll = max(0, min(n-1, s.at+k.d)), true
			}
		}
		if root.Shortcut(0, ui.KeyY) {
			act(s.at, Accepted)
		}
		if root.Shortcut(0, ui.KeyN) {
			act(s.at, Rejected)
		}
		if root.Shortcut(0, ui.KeyU) && s.undoLast(d) {
			changed = true
		}
		if s.pick.on && root.Shortcut(0, ui.KeyEscape) {
			s.pick = pick{}
		}
	}
	root.Children(func() {
		s.bar(c, d, labels, func(v Verdict) {
			var open []int
			for i, x := range d {
				if x.Verdict == Undecided {
					open = append(open, i)
				}
			}
			decide(Decision{Verdict: v}, open...)
			s.focus = true
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(1.5), t.Space(3), t.Space(6)).Children(func() {
				if s.err != nil {
					ui.Text(c, "Not a diff: "+s.err.Error()).TextColor(t.Danger).Padding(t.Space(2), 0)
				}
				for i := range n {
					s.card(c, ct, i, d[i], labels, func(v Verdict) {
						act(i, v)
						s.focus = true
					})
				}
			})
		})
		s.scroll = false
	})
	// A press on a line takes no focus, and a button clicked goes as its
	// hunk folds, taking the focus, and the keys, with it.
	if s.focus {
		root.Focus()
		s.focus = false
	}
	if changed {
		root.SetChanged()
	}
	return root
}

// decide sets the decisions on hunks hs to v, as one step to undo, and
// reports whether any changed.
func (s *State) decide(d []Decision, v Decision, hs ...int) bool {
	var step []change
	for _, h := range hs {
		if d[h].Verdict == v.Verdict && slices.Equal(d[h].Lines, v.Lines) {
			continue
		}
		step = append(step, change{h, d[h]})
		d[h] = v
	}
	if len(step) == 0 {
		return false
	}
	s.undo = append(s.undo, step)
	s.pick = pick{}
	return true
}

// undoLast puts back the decisions of the last step, going to its hunk.
func (s *State) undoLast(d []Decision) bool {
	if len(s.undo) == 0 {
		return false
	}
	step := s.undo[len(s.undo)-1]
	s.undo = s.undo[:len(s.undo)-1]
	for _, ch := range step {
		if ch.h < len(d) {
			d[ch.h] = ch.was
		}
	}
	s.at, s.scroll = step[0].h, true
	return true
}

// next goes to the first hunk left after h, coming round at the end, or
// stays on h when none is.
func (s *State) next(d []Decision, h int) {
	s.at, s.scroll = h, true
	for k := 1; k < len(d); k++ {
		if i := (h + k) % len(d); d[i].Verdict == Undecided {
			s.at = i
			return
		}
	}
}

// chosen returns the changed lines chosen in hunk h, in order, or nil.
func (s *State) chosen(h int) []int {
	if !s.pick.on || s.pick.h != h {
		return nil
	}
	var out []int
	lines := s.file.Hunks[h].Lines
	for j := min(s.pick.from, s.pick.to); j <= max(s.pick.from, s.pick.to) && j < len(lines); j++ {
		if k := lines[j][0]; k == '-' || k == '+' {
			out = append(out, j)
		}
	}
	return out
}

// bar shows how many hunks are left and the buttons deciding them all.
func (s *State) bar(c *ui.Context, d []Decision, labels Labels, all func(Verdict)) {
	t := c.Theme()
	left, accepted, rejected := 0, 0, 0
	for _, x := range d {
		switch x.Verdict {
		case Undecided:
			left++
		case Accepted:
			accepted++
		case Rejected:
			rejected++
		}
	}
	ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
		switch {
		case len(d) == 0:
			ui.Text(c, "No hunks").TextColor(t.TextMuted)
		case left == 0:
			mark(c, Accepted)
			ui.Text(c, "All hunks decided").Bold().TextColor(t.Success)
		default:
			ui.Text(c, fmt.Sprintf("%d of %d %s left", left, len(d), plural(len(d), "hunk"))).Bold().FontFeatures("tnum")
		}
		var tally []string
		if accepted > 0 {
			tally = append(tally, fmt.Sprintf("%d %s", accepted, strings.ToLower(labels.Accepted)))
		}
		if rejected > 0 {
			tally = append(tally, fmt.Sprintf("%d %s", rejected, strings.ToLower(labels.Rejected)))
		}
		ui.Text(c, strings.Join(tally, " · ")).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
		hint := fmt.Sprintf("j k to move · y to %s · n to %s · u to undo", strings.ToLower(labels.Accept), strings.ToLower(labels.Reject))
		ui.Text(c, hint).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1).TextAlign(ui.End)
		if ui.Button(c, labels.Reject+" all").Disabled(left == 0).Clicked() {
			all(Rejected)
		}
		if ui.Button(c, labels.Accept+" all").Disabled(left == 0).Clicked() {
			all(Accepted)
		}
	})
}

// card shows hunk i, decided as dec: its header with the buttons deciding
// it, which call act, and its lines unless it is decided and folded.
func (s *State) card(c *ui.Context, ct *editor.Theme, i int, dec Decision, labels Labels, act func(Verdict)) {
	t := c.Theme()
	h := s.file.Hunks[i]
	size := t.FontSize - 1
	open := dec.Verdict == Undecided
	border := t.Border
	if i == s.at {
		border = t.Accent
	}
	card := ui.Column(c).Margin(t.Space(1.5), 0).Radius(t.Space(2)).Border(1, border).Clip().
		Role(ui.RoleGroup).Label(fmt.Sprintf("Hunk %d", i+1))
	if s.scroll && i == s.at {
		card.ScrollIntoView()
	}
	if card.Clicked() {
		s.at, s.focus = i, true
	}
	added, removed := 0, 0
	for _, l := range h.Lines {
		switch l[0] {
		case '+':
			added++
		case '-':
			removed++
		}
	}
	card.Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2), t.Space(1.5), t.Space(3)).MinHeight(t.Space(10)).
			AlignItems(ui.Center).Background(t.Text.Alpha(0.03)).Children(func() {
			title := func() {
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					mark(c, dec.Verdict)
					ui.Text(c, fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OldStart, h.OldLines, h.NewStart, h.NewLines)).
						Font("monospace").FontSize(size).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().Shrink(0)
					ui.Text(c, h.Section).Font("monospace").FontSize(size).SingleLine().Shrink(1)
				})
			}
			if open {
				ui.Row(c).Grow(1).Shrink(1).MinWidth(0).Children(title)
			} else {
				fold := ui.ButtonBase(c).Grow(1).Shrink(1).MinWidth(0).Justify(ui.Start).
					Expanded(s.opened[i]).Label(fmt.Sprintf("Hunk %d, %s", i+1, verdictLabel(dec, h, labels)))
				if fold.Clicked() {
					s.opened[i] = !s.opened[i]
					s.at, s.focus = i, true
				}
				fold.Children(func() {
					ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Grow(1).Shrink(1).MinWidth(0).Children(func() {
						title()
						ui.Spacer(c)
						chevron(c, s.opened[i], fold.Hovered())
					})
				})
			}
			counts(c, added, removed)
			if open {
				accept := labels.Accept
				if k := len(s.chosen(i)); k > 0 && k < added+removed {
					accept = fmt.Sprintf("%s %d %s", labels.Accept, k, plural(k, "line"))
				}
				if ui.Button(c, labels.Reject).Tooltip("N").Clicked() {
					act(Rejected)
				}
				var b ui.Element
				if i == s.at {
					b = ui.PrimaryButton(c, accept)
				} else {
					b = ui.Button(c, accept)
				}
				if b.Tooltip("Y").Clicked() {
					act(Accepted)
				}
				return
			}
			color := t.Success
			if dec.Verdict == Rejected {
				color = t.Danger
			}
			ui.Text(c, verdictLabel(dec, h, labels)).FontSize(t.FontSize - 1).TextColor(color).SingleLine()
			if ui.Button(c, "Reset").Clicked() {
				act(Undecided)
			}
		})
		if open || s.opened[i] {
			ui.Divider(c)
			s.lines(c, ct, i, dec)
		}
	})
}

// verdictLabel words dec on h: "Accepted", "Accepted 2 of 5 lines".
func verdictLabel(dec Decision, h editor.PatchHunk, labels Labels) string {
	if dec.Verdict == Rejected {
		return labels.Rejected
	}
	if len(dec.Lines) == 0 {
		return labels.Accepted
	}
	all := changes(h)
	return fmt.Sprintf("%s %d of %d %s", labels.Accepted, len(dec.Lines), all, plural(all, "line"))
}

// changes returns how many lines h removes and adds.
func changes(h editor.PatchHunk) int {
	n := 0
	for _, l := range h.Lines {
		if l[0] == '-' || l[0] == '+' {
			n++
		}
	}
	return n
}

// lines shows the lines of hunk i, decided as dec, numbered as in the old
// file and the new. Open, a press on them chooses lines; decided, they show
// dimmed, those not accepted plain.
func (s *State) lines(c *ui.Context, ct *editor.Theme, i int, dec Decision) {
	t := c.Theme()
	h := s.file.Hunks[i]
	size := t.FontSize - 1
	rowH := float32(math.Round(float64(size * lineHeight)))
	numW, _ := c.MeasureText(0, ui.Span{Text: strings.Repeat("0", s.digits), Font: "monospace", Size: size})
	open := dec.Verdict == Undecided
	tint := float32(0.12)
	if t.Dark {
		tint = 0.2
	}
	box := ui.Column(c).Padding(t.Space(1), 0)
	if open {
		pressed := ui.Local(box, "hunks.pressed", func() bool { return false })
		if _, _, ok := box.Dragged(); ok {
			// Captured, the pointer reads in the box's axes past its edges
			// too, which the rows at either end take.
			_, py, _ := box.PointerPosition()
			j := max(0, min(len(h.Lines)-1, int(math.Floor(float64((py-t.Space(1))/rowH)))))
			switch {
			case s.pick.on && s.pick.h == i && (*pressed || c.Modifiers()&ui.Shift != 0):
				s.pick.to = j
			case !*pressed:
				s.pick = pick{on: true, h: i, from: j, to: j}
			}
			if !*pressed {
				s.focus = true
			}
			s.at, *pressed = i, true
		} else {
			*pressed = false
		}
	} else {
		box.Opacity(0.6)
	}
	picked := s.chosen(i)
	box.Children(func() {
		o, n := h.OldStart, h.NewStart
		for j, l := range h.Lines {
			kind, text := l[0], strings.TrimSuffix(l[1:], "\r")
			row := ui.Row(c).Height(rowH).AlignItems(ui.Center).ClipX().Font("monospace").FontSize(size)
			if kind == '\\' {
				row.Children(func() {
					ui.Box(c).Width(3 + 2*numW + t.Space(10))
					ui.Text(c, strings.TrimSpace(text)).Italic().TextColor(t.TextMuted).SingleLine()
				})
				continue
			}
			var color ui.Color
			oldNum, newNum := "", ""
			switch kind {
			case '-':
				color, oldNum = ct.Deleted, strconv.Itoa(o)
				o++
			case '+':
				color, newNum = ct.Added, strconv.Itoa(n)
				n++
			default:
				oldNum, newNum = strconv.Itoa(o), strconv.Itoa(n)
				o, n = o+1, n+1
			}
			changed := kind == '-' || kind == '+'
			taken := !changed || open || dec.Verdict == Accepted && (len(dec.Lines) == 0 || slices.Contains(dec.Lines, j))
			chosen := slices.Contains(picked, j)
			switch {
			case chosen:
				row.Background(color.Alpha(tint + 0.12))
			case changed && taken:
				row.Background(color.Alpha(tint))
			}
			row.Children(func() {
				bar := ui.Box(c).Width(3).AlignSelf(ui.Stretch)
				if chosen {
					bar.Background(t.Accent)
				}
				for _, num := range []string{oldNum, newNum} {
					ui.Text(c, num).Width(numW).MarginStart(t.Space(2)).TextAlign(ui.End).TextColor(ct.LineNumber).FontFeatures("tnum").Shrink(0)
				}
				sign := ui.Text(c, string(kind)).Width(t.Space(4)).MarginStart(t.Space(2)).TextAlign(ui.Center).Shrink(0)
				if changed && taken {
					sign.TextColor(color)
				} else {
					sign.TextColor(t.TextMuted)
				}
				ui.RichText(c, spans(text, s.toks[i][j], ct, !taken, t)...).NoWrap().Grow(1).Shrink(1).MinWidth(0).PaddingEnd(t.Space(3))
			})
		}
	})
}

// counts shows "+added −removed", leaving out those 0.
func counts(c *ui.Context, added, removed int) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(1)).Children(func() {
		if added > 0 {
			ui.Text(c, fmt.Sprintf("+%d", added)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.Success)
		}
		if removed > 0 {
			ui.Text(c, fmt.Sprintf("−%d", removed)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.Danger)
		}
	})
}

// mark shows a dot for a hunk left, a check for one accepted, or a cross
// for one rejected.
func mark(c *ui.Context, v Verdict) {
	t := c.Theme()
	ui.Box(c).Size(t.FontSize, t.FontSize).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.3
		switch v {
		case Undecided:
			var dot ui.Path
			p.FillPath(dot.Circle(cx, cy, r.W*0.25), t.Warning)
		case Accepted:
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, t.Success)
		case Rejected:
			k *= 0.8
			p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.75, t.Danger)
			p.StrokePolyline([][2]float32{{cx + k, cy - k}, {cx - k, cy + k}}, 1.75, t.Danger)
		}
	})
}

// chevron shows that a header opens, pointing right, or down when open,
// stronger while hovered.
func chevron(c *ui.Context, open, hovered bool) {
	t := c.Theme()
	color := t.TextMuted.Alpha(0.7)
	if hovered {
		color = t.Text
	}
	ui.Box(c).Size(t.FontSize, t.FontSize).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.22
		pts := [][2]float32{{cx - k/2, cy - k}, {cx + k/2, cy}, {cx - k/2, cy + k}}
		if open {
			pts = [][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}
		}
		p.StrokePolyline(pts, 1.5, color)
	})
}

// spans returns text in spans of its tokens' colors, or muted, tabs set to
// every fourth column.
func spans(text string, tokens []editor.Token, ct *editor.Theme, muted bool, t *ui.Theme) []ui.Span {
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
		if muted {
			sp.Color = t.TextMuted
		}
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

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// parse reads the first file of diff and tokenizes its hunks' lines as the
// language of its name.
func (s *State) parse(diff string) {
	*s = State{diff: diff, parsed: true, opened: map[int]bool{}}
	files, err := editor.ParsePatch(diff)
	if err != nil {
		s.err = err
		return
	}
	if len(files) == 0 {
		return
	}
	s.file = files[0]
	name := s.file.NewName
	if name == "" {
		name = s.file.OldName
	}
	lang := editor.LanguageFor(name)
	last := 0
	for _, h := range s.file.Hunks {
		last = max(last, h.OldStart+h.OldLines, h.NewStart+h.NewLines)
		toks := make([][]editor.Token, len(h.Lines))
		if lang != nil && lang.Tokenizer != nil {
			var state uint64
			for j, l := range h.Lines {
				if l[0] != '\\' {
					toks[j], state = lang.Tokenizer.Tokenize(strings.TrimSuffix(l[1:], "\r"), state, nil)
				}
			}
		}
		s.toks = append(s.toks, toks)
	}
	s.digits = max(2, len(strconv.Itoa(last)))
}

// Patch returns the patch of what decisions accepted of diff, for git apply
// --cached to take: diff's file headers, then the hunks accepted, of each
// only the lines accepted, a line removed but not accepted kept as a line of
// both and one added but not accepted left out, their headers' ranges
// counted again. It returns "" when nothing is accepted, or diff is not one
// View shows.
func Patch(diff string, decisions []Decision) string {
	files, err := editor.ParsePatch(diff)
	if err != nil || len(files) == 0 {
		return ""
	}
	f := files[0]
	var body strings.Builder
	shift, kept := 0, 0
	for i, h := range f.Hunks {
		if i >= len(decisions) || decisions[i].Verdict != Accepted {
			continue
		}
		lines, oldN, newN := take(h, decisions[i].Lines)
		first := h.OldStart
		if h.OldLines == 0 {
			first++
		}
		start := first + shift
		if newN == 0 {
			start--
		}
		fmt.Fprintf(&body, "@@ -%d,%d +%d,%d @@", h.OldStart, oldN, start, newN)
		if h.Section != "" {
			body.WriteString(" " + h.Section)
		}
		body.WriteString("\n" + strings.Join(lines, "\n") + "\n")
		shift += newN - oldN
		kept += newN
	}
	if body.Len() == 0 {
		return ""
	}
	head := ""
	if i := strings.Index("\n"+diff, "\n@@ "); i > 0 {
		head = diff[:i]
	}
	// A file deleted keeps the lines whose removal was not accepted, so the
	// patch changes it instead.
	if f.NewName == "" && f.OldName != "" && kept > 0 {
		var out []string
		for l := range strings.SplitSeq(strings.TrimSuffix(head, "\n"), "\n") {
			switch {
			case strings.HasPrefix(l, "deleted file mode "):
				continue
			case strings.HasPrefix(l, "+++ "):
				l = "+++ b/" + f.OldName
			}
			out = append(out, l)
		}
		head = strings.Join(out, "\n") + "\n"
	}
	return head + body.String()
}

// take returns the lines of h that the lines accepted, all with none, make,
// and how many of them are of the old file and of the new.
func take(h editor.PatchHunk, accepted []int) (out []string, oldN, newN int) {
	// in is whether the line before a "\ No newline at end of file" is in
	// the patch, which the note follows.
	in := false
	for j, l := range h.Lines {
		on := len(accepted) == 0 || slices.Contains(accepted, j)
		switch l[0] {
		case '\\':
			if in {
				out = append(out, l)
			}
			continue
		case '-':
			if !on {
				l = " " + l[1:]
			}
		case '+':
			if !on {
				in = false
				continue
			}
			// The old file's last line, without a newline, kept as a line of
			// both, needs one in the new file for lines to follow it.
			if k := len(out); k >= 2 && out[k-1][0] == '\\' && out[k-2][0] == ' ' {
				last := out[k-2][1:]
				out = append(out[:k-2], "-"+last, out[k-1], "+"+last)
			}
		}
		in = true
		if l[0] != '+' {
			oldN++
		}
		if l[0] != '-' {
			newN++
		}
		out = append(out, l)
	}
	return out, oldN, newN
}

// Package redline shows edits suggested to prose, as Word's Track Changes
// and Google Docs' suggesting mode show them: what an edit takes out struck
// through in red, and what it puts in underlined in green after it. The
// pointer on an edit outlines it, and a click pins a card of it, who
// suggested it and why, with Accept and Reject. An edit decided shows as
// plain text, the new words or the old. A line above counts the edits
// left, with Undo, Reject all and Accept all.
//
//	if app.pieces == nil {
//		app.pieces = redline.Diff(draft, suggested)
//	}
//	if redline.View(c, &app.review, &app.pieces).Changed {
//		app.text = redline.Text(app.pieces)
//	}
//
// With the text focused, or a card pinned, J and K pin the next and the
// previous edit left, Y accepts the edit pinned and N rejects it, both
// going on to the next, U and Cmd+Z undo the last decision, and Escape
// unpins the card.
package redline

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Status is what became of an edit.
type Status uint8

const (
	Pending Status = iota
	Accepted
	Rejected
)

// Piece is a run of the text: unchanged where Old and New are the same, or
// an edit taking Old out and putting New in, either of which may be empty,
// with who suggested it, why, and what became of it.
type Piece struct {
	Old, New     string
	Author, Note string
	Status       Status
}

// Edit reports whether p is an edit rather than unchanged text.
func (p Piece) Edit() bool { return p.Old != p.New }

// Text returns the text as decided so far: the new words of the edits
// accepted, and the old words of the rest.
func Text(pieces []Piece) string {
	var b strings.Builder
	for _, p := range pieces {
		if p.Status == Accepted {
			b.WriteString(p.New)
		} else {
			b.WriteString(p.Old)
		}
	}
	return b.String()
}

// Diff returns the edits that turn old into new, word by word, or
// character by character in Chinese and Japanese. Edits parted only by
// spaces are one edit, as people read "quick brown" changed to "slow red".
func Diff(old, new string) []Piece {
	a, b := tokens(old), tokens(new)
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	var pieces []Piece
	equal := func(s string) {
		if s == "" {
			return
		}
		if k := len(pieces) - 1; k >= 0 && !pieces[k].Edit() {
			pieces[k].Old += s
			pieces[k].New += s
			return
		}
		pieces = append(pieces, Piece{Old: s, New: s})
	}
	edit := func(o, n string) {
		if k := len(pieces) - 1; k >= 0 && pieces[k].Edit() {
			pieces[k].Old += o
			pieces[k].New += n
			return
		}
		pieces = append(pieces, Piece{Old: o, New: n})
	}
	equal(strings.Join(a[:pre], ""))
	x, y := a[pre:len(a)-suf], b[pre:len(b)-suf]
	// lcs[i*w+j] is the length of the longest common subsequence of x[i:]
	// and y[j:].
	w := len(y) + 1
	lcs := make([]int32, (len(x)+1)*w)
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i*w+j] = lcs[(i+1)*w+j+1] + 1
			} else {
				lcs[i*w+j] = max(lcs[(i+1)*w+j], lcs[i*w+j+1])
			}
		}
	}
	for i, j := 0, 0; i < len(x) || j < len(y); {
		switch {
		case i < len(x) && j < len(y) && x[i] == y[j]:
			equal(x[i])
			i, j = i+1, j+1
		case j == len(y) || i < len(x) && lcs[(i+1)*w+j] >= lcs[i*w+j+1]:
			edit(x[i], "")
			i++
		default:
			edit("", y[j])
			j++
		}
	}
	equal(strings.Join(a[len(a)-suf:], ""))
	for k := 1; k+1 < len(pieces); {
		p := pieces[k]
		if !p.Edit() && strings.TrimSpace(p.Old) == "" && !strings.Contains(p.Old, "\n") && pieces[k-1].Edit() && pieces[k+1].Edit() {
			pieces[k-1].Old += p.Old + pieces[k+1].Old
			pieces[k-1].New += p.New + pieces[k+1].New
			pieces = slices.Delete(pieces, k, k+2)
			continue
		}
		k++
	}
	return pieces
}

// tokens splits s into words, runs of spaces, newlines, Chinese and
// Japanese characters, and other characters, one each.
func tokens(s string) []string {
	var out []string
	for len(s) > 0 {
		r, n := utf8.DecodeRuneInString(s)
		switch {
		case r == '\n' || ideograph(r):
		case unicode.IsSpace(r):
			for n < len(s) {
				r, k := utf8.DecodeRuneInString(s[n:])
				if r == '\n' || !unicode.IsSpace(r) {
					break
				}
				n += k
			}
		case wordy(r):
			for n < len(s) {
				r, k := utf8.DecodeRuneInString(s[n:])
				if r == '\'' || r == '’' {
					// An apostrophe stays in a word, as in "don't", when a
					// letter follows it.
					if next, _ := utf8.DecodeRuneInString(s[n+k:]); !wordy(next) {
						break
					}
				} else if !wordy(r) {
					break
				}
				n += k
			}
		}
		out = append(out, s[:n])
		s = s[n:]
	}
	return out
}

func wordy(r rune) bool {
	return (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)) && !ideograph(r)
}

func ideograph(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
}

// Events is the view, and whether a decision changed the pieces this
// frame.
type Events struct {
	ui.Element
	Changed bool
}

// State is what a view keeps: the text laid out, the edit pinned, and the
// decisions to undo. Keep it in the app's state, one for each view.
type State struct {
	pieces []Piece
	font   ui.Font
	toks   []token
	ascent float32
	textH  float32
	width  float32
	lines  int
	// pinned is the index of the edit pinned plus 1, so that 0 is none,
	// and reveal scrolls it into view in the next frame.
	pinned int
	reveal bool
	undo   [][]change
}

// change is a decision to undo: the edit and its status before.
type change struct {
	piece int
	was   Status
}

type kind uint8

const (
	plain kind = iota
	deleted
	inserted
)

// token is a token of the text as shown, the piece it is of and how it
// shows, shaped, and where it is laid out. brk is whether a line may break
// before it.
type token struct {
	piece          int
	kind           kind
	text           *ui.TextLayout
	w              float32
	space, nl, brk bool
	x              float32
	line           int
}

// lineHeight is the height of a line, in sizes of the font.
const lineHeight = 1.8

// prepare shapes the text of pieces in font, as it shows.
func (s *State) prepare(pieces []Piece, font ui.Font) {
	s.pieces, s.font, s.width = slices.Clone(pieces), font, -1
	s.ascent = font.Metrics().Ascent
	_, s.textH = ui.ShapeText("Mg", font, 0).Size()
	s.toks = s.toks[:0]
	var prev rune
	emit := func(i int, k kind, text string) {
		for _, tk := range tokens(text) {
			r, _ := utf8.DecodeRuneInString(tk)
			t := token{piece: i, kind: k, nl: r == '\n', space: unicode.IsSpace(r) && r != '\n'}
			// A line breaks after spaces and around Chinese and Japanese
			// characters, but not before punctuation.
			t.brk = unicode.IsSpace(prev) || ideograph(r) || (ideograph(prev) && !unicode.IsPunct(r))
			if !t.nl {
				t.text = ui.ShapeText(tk, font, 0)
				t.w, _ = t.text.Size()
			}
			s.toks = append(s.toks, t)
			prev = r
		}
	}
	for i, p := range pieces {
		switch {
		case !p.Edit() || p.Status == Rejected:
			emit(i, plain, p.Old)
		case p.Status == Accepted:
			emit(i, plain, p.New)
		default:
			emit(i, deleted, p.Old)
			emit(i, inserted, p.New)
		}
	}
}

// layout places the tokens on lines width wide, a run that may not break
// moving to the next line whole. Spaces never wrap: one passing the end
// of a line hangs past it.
func (s *State) layout(width float32) {
	s.width = width
	limit := width
	if limit <= 0 {
		limit = float32(math.Inf(1))
	}
	x, line := float32(0), 0
	for i := 0; i < len(s.toks); {
		if s.toks[i].nl {
			s.toks[i].x, s.toks[i].line = x, line
			x, line = 0, line+1
			i++
			continue
		}
		j, w := i+1, s.toks[i].w
		for j < len(s.toks) && !s.toks[j].brk && !s.toks[j].nl {
			w += s.toks[j].w
			j++
		}
		if !s.toks[i].space && x > 0 && x+w > limit {
			x, line = 0, line+1
		}
		for ; i < j; i++ {
			s.toks[i].x, s.toks[i].line = x, line
			x += s.toks[i].w
		}
	}
	s.lines = line + 1
}

// at returns the pending edit at (x, y) in the text, or -1.
func (s *State) at(x, y, lineH float32) int {
	line := int(math.Floor(float64(y / lineH)))
	for _, tk := range s.toks {
		if tk.line == line && tk.kind != plain && x >= tk.x && x < tk.x+tk.w {
			return tk.piece
		}
	}
	return -1
}

// boxes returns the extent of edit i on each line it takes.
func (s *State) boxes(i int) []ui.Rect {
	var out []ui.Rect
	for _, tk := range s.toks {
		if tk.piece != i || tk.nl {
			continue
		}
		if k := len(out) - 1; k >= 0 && out[k].Y == float32(tk.line) {
			out[k].W = tk.x + tk.w - out[k].X
			continue
		}
		out = append(out, ui.Rect{X: tk.x, Y: float32(tk.line), W: tk.w})
	}
	return out
}

// next returns the pending edit after edit i, dir 1, or before it, dir
// -1, going round, or -1 when none is left.
func next(pieces []Piece, i, dir int) int {
	n := len(pieces)
	for k := 1; k <= n; k++ {
		j := ((i+dir*k)%n + n) % n
		if pieces[j].Edit() && pieces[j].Status == Pending {
			return j
		}
	}
	return -1
}

// View creates the line of the edits left and below it the text, wrapped
// to the width the view is given, with the card of the edit pinned.
func View(c *ui.Context, s *State, pieces *[]Piece) Events {
	t := c.Theme()
	ps := *pieces
	font := ui.Font{Family: t.Font, Size: t.FontSize + 2}
	lineH := font.Size * lineHeight
	ev := Events{}
	decide := func(st Status, which ...int) {
		var step []change
		for _, i := range which {
			if i >= 0 && ps[i].Edit() && ps[i].Status != st {
				step = append(step, change{i, ps[i].Status})
				ps[i].Status = st
			}
		}
		if len(step) > 0 {
			s.undo = append(s.undo, step)
			ev.Changed = true
		}
	}
	undo := func() {
		if len(s.undo) == 0 {
			return
		}
		step := s.undo[len(s.undo)-1]
		s.undo = s.undo[:len(s.undo)-1]
		for _, ch := range step {
			ps[ch.piece].Status = ch.was
		}
		if len(step) == 1 {
			s.pinned, s.reveal = step[0].piece+1, true
		}
		ev.Changed = true
	}
	var left []int
	for i, p := range ps {
		if p.Edit() && p.Status == Pending {
			left = append(left, i)
		}
	}
	// act does what key asks, from the text or the card, whichever has
	// the focus.
	act := func(key ui.Key, cmd bool) {
		pinned := s.pinned - 1
		switch {
		case cmd && key == ui.KeyZ, key == ui.KeyU:
			undo()
		case key == ui.KeyJ, key == ui.KeyK:
			dir := 1
			if key == ui.KeyK {
				dir = -1
			}
			from := pinned
			if from < 0 && dir < 0 {
				from = 0
			}
			s.pinned, s.reveal = next(ps, from, dir)+1, true
		case (key == ui.KeyY || key == ui.KeyN) && pinned >= 0 && ps[pinned].Status == Pending:
			st := Accepted
			if key == ui.KeyN {
				st = Rejected
			}
			decide(st, pinned)
			s.pinned, s.reveal = next(ps, pinned, 1)+1, true
		case key == ui.KeyEscape:
			s.pinned = 0
		}
	}
	col := ui.Column(c).Gap(t.Space(3))
	ev.Element = col
	col.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			label := strconv.Itoa(len(left)) + " edits left"
			switch len(left) {
			case 0:
				label = "No edits left"
			case 1:
				label = "1 edit left"
			}
			ui.Text(c, label).TextColor(t.TextMuted).FontFeatures("tnum").Grow(1)
			if ui.Button(c, "Undo").Disabled(len(s.undo) == 0).Clicked() {
				undo()
			}
			if ui.Button(c, "Reject all").Disabled(len(left) == 0).Clicked() {
				decide(Rejected, left...)
				s.pinned = 0
			}
			if ui.PrimaryButton(c, "Accept all").Disabled(len(left) == 0).Clicked() {
				decide(Accepted, left...)
				s.pinned = 0
			}
		})
		s.text(c, ps, font, lineH, act, decide)
	})
	return ev
}

// text creates the text on lines lineH tall, and the card of the edit
// pinned. A click on an edit pins it, or unpins it when pinned; a click
// beside the edits or outside the view unpins.
func (s *State) text(c *ui.Context, ps []Piece, font ui.Font, lineH float32, act func(ui.Key, bool), decide func(Status, ...int)) {
	t := c.Theme()
	box := ui.Box(c).Focusable().Label("Suggested edits")
	for _, k := range []ui.Key{ui.KeyJ, ui.KeyK, ui.KeyY, ui.KeyN, ui.KeyU} {
		if box.Shortcut(0, k) {
			act(k, false)
		}
	}
	if box.Shortcut(ui.Cmd, ui.KeyZ) {
		act(ui.KeyZ, true)
	}
	if !slices.Equal(s.pieces, ps) || s.font != font {
		s.prepare(ps, font)
	}
	if w := box.LayoutBounds().W; w != s.width {
		s.layout(w)
	}
	hot := -1
	if x, y, ok := box.PointerPosition(); ok && box.Hovered() {
		hot = s.at(x, y, lineH)
	}
	if hot >= 0 {
		box.Cursor(ui.CursorPointer)
	}
	switch {
	case box.Clicked():
		if hot < 0 || hot == s.pinned-1 {
			s.pinned = 0
		} else {
			s.pinned = hot + 1
		}
	case box.PressedOutside():
		s.pinned = 0
	}
	if p := s.pinned - 1; p >= 0 && (p >= len(ps) || ps[p].Status != Pending) {
		s.pinned = 0
	}
	box.Height(float32(s.lines) * lineH)
	pinned := s.pinned - 1
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The tokens were laid out in the width of the last frame.
		if r.W != s.width {
			c.Invalidate()
		}
		bh := min(lineH-2, s.textH+4)
		pad, ty := (lineH-bh)/2, (lineH-s.textH)/2
		for _, tk := range s.toks {
			if tk.nl {
				continue
			}
			x, y := r.X+tk.x, r.Y+float32(tk.line)*lineH
			color := t.Text
			switch tk.kind {
			case deleted:
				color = t.Danger
				p.Fill(ui.Rect{X: x, Y: y + pad, W: tk.w, H: bh}, t.Danger.Alpha(0.1), 0)
				p.Fill(ui.Rect{X: x, Y: y + ty + s.ascent*0.66, W: tk.w, H: 1.5}, t.Danger, 0)
			case inserted:
				color = t.Success
				p.Fill(ui.Rect{X: x, Y: y + pad, W: tk.w, H: bh}, t.Success.Alpha(0.12), 0)
				p.Fill(ui.Rect{X: x, Y: y + ty + s.ascent + 2, W: tk.w, H: 1.5}, t.Success, 0)
			}
			if !tk.space {
				p.TextLayout(tk.text, x, y+ty, color)
			}
		}
		outline := func(i int, color ui.Color, width float32) {
			for _, b := range s.boxes(i) {
				p.Stroke(ui.Rect{X: r.X + b.X - 2, Y: r.Y + b.Y*lineH + pad - 1, W: b.W + 4, H: bh + 2}, color, 3, width)
			}
		}
		if hot != pinned {
			outline(hot, t.Text.Alpha(0.4), 1)
		}
		outline(pinned, t.Accent, 1.5)
	})
	if pinned < 0 {
		return
	}
	boxes := s.boxes(pinned)
	if len(boxes) == 0 {
		return
	}
	at := boxes[0]
	box.Children(func() {
		anchor := ui.Box(c).Absolute().PassThrough().Left(at.X).Top(at.Y*lineH).Size(at.W, lineH)
		if s.reveal {
			anchor.ScrollIntoView()
			s.reveal = false
		}
		s.card(c, anchor, ps[pinned], pinned, act, decide)
	})
}

// card shows by anchor the card of edit i, p: who suggested it and why,
// with Accept and Reject.
func (s *State) card(c *ui.Context, anchor ui.Element, p Piece, i int, act func(ui.Key, bool), decide func(Status, ...int)) {
	t := c.Theme()
	ui.Overlay(c, func() {
		tip := ui.Column(c).Absolute().Role(ui.RoleDialog).Label("Edit").Width(t.Space(72)).
			AttachTo(anchor, ui.AnchorBottomStart, ui.AnchorTopStart).Margin(t.Space(1.5), 0, 0, 0).
			Padding(t.Space(3)).Gap(t.Space(2.5)).Radius(t.Space(2.5)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		for _, k := range []ui.Key{ui.KeyJ, ui.KeyK, ui.KeyY, ui.KeyN, ui.KeyU, ui.KeyEscape} {
			if tip.OverlayShortcut(0, k) {
				act(k, false)
				c.Invalidate()
			}
		}
		tip.Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				if p.Author != "" {
					ui.Avatar(c, p.Author, nil).Size(t.Space(5), t.Space(5)).FontSize(t.FontSize - 4)
					ui.Text(c, p.Author).Bold()
				}
				what := "Replace"
				switch {
				case p.Old == "":
					what = "Insert"
				case p.New == "":
					what = "Delete"
				}
				ui.Text(c, what).TextColor(t.TextMuted)
			})
			if p.Note != "" {
				ui.Text(c, p.Note).Wrap()
			}
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				if ui.PrimaryButton(c, "Accept").Clicked() {
					decide(Accepted, i)
					s.pinned = 0
				}
				if ui.Button(c, "Reject").Clicked() {
					decide(Rejected, i)
					s.pinned = 0
				}
			})
		})
	})
}

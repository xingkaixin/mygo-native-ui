// Package logprobs shows a language model's output with how likely the
// model found each token, as OpenAI's playground and logprob viewers show
// it: the tokens run as the text they make, wrapping between tokens and
// breaking at their newlines, each newline a faint ↵, and each token tinted
// by how unsure the model was of it, clear near 100% and deeper toward 0%,
// or, in Entropy mode, by how evenly its top alternatives shared the
// chance. With Boundaries, faint lines part the tokens; those less likely
// than Threshold are underlined.
//
// The pointer on a token shows a card of it: the token, its spaces and
// newlines made visible as ␣ and ↵, its probability and logprob, and its
// top alternatives with bars and their shares, the chosen one checked. A
// click pins the card, which Left and Right then move to the token before
// and after, and Up and Down to the line above and below; Escape or a click
// elsewhere unpins it. A line above sums the text up: the tokens, their
// mean logprob, the perplexity, and the least likely token, which a click
// pins and scrolls to. The app reads the token pinned:
//
//	ev := logprobs.View(c, &app.view, app.tokens)
//	if i, ok := ev.Pinned(); ok {
//		... // app.tokens[i]
//	}
package logprobs

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Token is a token the model chose: its text, as " the" or "\n", the
// natural log of its probability, and the likeliest tokens in its place,
// likeliest first, among which it is when it was one of them, as the
// top_logprobs of OpenAI's API.
type Token struct {
	Text    string
	Logprob float64
	Top     []Alternative
}

// Alternative is a token the model might have chosen, and the natural log
// of its probability.
type Alternative struct {
	Text    string
	Logprob float64
}

// Mode is what tints the tokens.
type Mode uint8

const (
	// Probability tints a token deeper the less likely it was.
	Probability Mode = iota
	// Entropy tints a token deeper the more evenly its top alternatives
	// shared the chance, where the model hesitated whatever it chose.
	Entropy
)

// Events are what a view reports in a frame.
type Events struct {
	pinned int
	ok     bool
}

// Pinned returns the index of the token whose card is pinned, or false
// when none is.
func (e Events) Pinned() (int, bool) { return e.pinned, e.ok }

// State is what a view keeps: the tokens shaped and laid out, their
// probabilities and entropies, and the token pinned. Mode, Threshold and
// Boundaries, which the app sets, choose what tints the tokens, underline
// those less likely than Threshold, from 0 to 1, and part the tokens with
// faint lines. Keep it in the app's state, one for each view; the zero
// State tints by probability and underlines none.
type State struct {
	Mode       Mode
	Threshold  float64
	Boundaries bool

	tokens []Token
	font   ui.Font
	plain  string
	// pieces are the tokens' text in runs, and mark the ↵ ending a run at
	// a newline; textH and ascent are those of a line of the text.
	pieces               []piece
	mark                 *ui.TextLayout
	markW, textH, ascent float32
	// probs is each token's probability, bits the entropy of its
	// alternatives, and spread that entropy as a share of the most the
	// view's longest top alternatives and the chance they leave could
	// have, all equally likely.
	probs, bits, spread []float64
	mean                float64
	lowest              int
	// width is the width the pieces were laid out in, and lines the lines
	// they take.
	width float32
	lines int
	// pinned is the index of the token pinned plus 1, so that 0 is none,
	// and reveal scrolls it into view in the next frame.
	pinned int
	reveal bool
}

// piece is a run of a token's text on one line: the text up to a newline
// or the token's end, shaped, nil when empty, its width, with the ↵ after
// it when a newline ends it, and where it is laid out. lead is the width
// of the spaces a token's first run starts with, which hang before the
// start of a line the token wraps to, as spaces between words do.
type piece struct {
	tok     int
	text    *ui.TextLayout
	w, lead float32
	nl      bool
	x       float32
	line    int
}

// across returns where pc shows across its line, without the spaces that
// hang before the line's start, and how wide.
func (pc piece) across() (x, w float32) {
	x = max(pc.x, 0)
	return x, pc.w - (x - pc.x)
}

// lineHeight is the height of a line of tokens, in sizes of their font.
const lineHeight = 1.75

// prepare starts over for tokens drawn in font.
func (s *State) prepare(tokens []Token, font ui.Font) {
	s.tokens, s.font, s.width, s.pinned, s.reveal = tokens, font, -1, 0, false
	s.mark = ui.ShapeText("↵", font, 0)
	s.markW, s.textH = s.mark.Size()
	s.ascent = font.Metrics().Ascent
	space, _ := ui.ShapeText(" ", font, 0).Size()
	s.pieces = s.pieces[:0]
	s.probs, s.bits, s.spread = make([]float64, len(tokens)), make([]float64, len(tokens)), make([]float64, len(tokens))
	s.lowest, s.mean = 0, 0
	var plain strings.Builder
	top := 0
	for i, tk := range tokens {
		plain.WriteString(tk.Text)
		runs := strings.Split(strings.ReplaceAll(tk.Text, "\t", "    "), "\n")
		for k, run := range runs {
			pc := piece{tok: i, nl: k < len(runs)-1}
			if run != "" {
				pc.text = ui.ShapeText(run, font, 0)
				pc.w, _ = pc.text.Size()
			}
			if k == 0 {
				pc.lead = space * float32(len(run)-len(strings.TrimLeft(run, " ")))
			}
			if pc.nl {
				pc.w += s.markW
			}
			if pc.w > 0 {
				s.pieces = append(s.pieces, pc)
			}
		}
		s.probs[i] = math.Exp(tk.Logprob)
		s.bits[i] = entropy(tk)
		top = max(top, len(tk.Top))
		s.mean += tk.Logprob / float64(len(tokens))
		if tk.Logprob < tokens[s.lowest].Logprob {
			s.lowest = i
		}
	}
	s.plain = plain.String()
	if most := math.Log2(float64(top + 1)); most > 0 {
		for i, b := range s.bits {
			s.spread[i] = min(1, b/most)
		}
	}
}

// entropy returns the entropy of tk's alternatives in bits, tk counted
// among them and the chance they leave as one more.
func entropy(tk Token) (bits float64) {
	sum := 0.0
	add := func(p float64) {
		if p > 0 {
			bits -= p * math.Log2(p)
		}
		sum += p
	}
	in := false
	for _, a := range tk.Top {
		add(math.Exp(a.Logprob))
		in = in || a.Text == tk.Text
	}
	if !in {
		add(math.Exp(tk.Logprob))
	}
	// What the alternatives leave is many tokens, each unlikely; as one,
	// it reads as less spread than it is, but no more than the API tells.
	if rest := 1 - sum; rest > 1e-3 {
		add(rest)
	}
	return bits
}

// layout places the pieces on lines width wide: a token that would pass
// the end of a line goes to the next, its leading spaces hanging before
// it, and a newline ends one. Width 0, as before the view has one, wraps
// nothing.
func (s *State) layout(width float32) {
	s.width = width
	limit := width
	if limit <= 0 {
		limit = float32(math.Inf(1))
	}
	x, line := float32(0), 0
	for i := range s.pieces {
		pc := &s.pieces[i]
		if x > 0 && x+pc.w > limit {
			x, line = -pc.lead, line+1
		}
		pc.x, pc.line = x, line
		x += pc.w
		if pc.nl {
			x, line = 0, line+1
		}
	}
	s.lines = line
	if x > 0 || line == 0 {
		s.lines++
	}
}

// first returns the index of the first piece of token i.
func (s *State) first(i int) (int, bool) {
	return slices.BinarySearchFunc(s.pieces, i, func(pc piece, i int) int { return cmp.Compare(pc.tok, i) })
}

// at returns the token at (x, y) in the view, lines lineH tall, or -1.
func (s *State) at(x, y, lineH float32) int {
	line := int(math.Floor(float64(y / lineH)))
	for _, pc := range s.pieces {
		if pc.line == line && x >= pc.x && x < pc.x+pc.w {
			return pc.tok
		}
	}
	return -1
}

// vertical returns the token on the line above token i, dir -1, or below
// it, 1, nearest it across, or i where there is no such line.
func (s *State) vertical(i, dir int) int {
	k, ok := s.first(i)
	if !ok {
		return i
	}
	from := s.pieces[k]
	cx, line := from.x+from.w/2, from.line+dir
	best, dist := i, float32(math.Inf(1))
	for _, pc := range s.pieces {
		if pc.line != line {
			continue
		}
		if d := max(pc.x-cx, cx-pc.x-pc.w, 0); d < dist {
			best, dist = pc.tok, d
		}
	}
	return best
}

// tint returns the color token i is tinted by s.Mode.
func (s *State) tint(t *ui.Theme, i int) ui.Color {
	color, f := t.Danger, 1-s.probs[i]
	if s.Mode == Entropy {
		color, f = ui.Oklch(0.62, 0.2, 295), s.spread[i]
	}
	// The curve keeps the many tokens the model was nearly sure of clear,
	// so that the few it was not stand out.
	return color.Alpha(float32(0.6 * math.Pow(f, 1.5)))
}

// View creates the line summing the tokens up and below it the tokens,
// wrapped to the width the view is given, with the card of the token
// under the pointer or pinned.
func View(c *ui.Context, s *State, tokens []Token) Events {
	t := c.Theme()
	font := ui.Font{Family: t.Font, Size: t.FontSize + 1}
	if !same(s.tokens, tokens) || s.font != font {
		s.prepare(tokens, font)
	}
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		s.summary(c)
		s.text(c, font.Size*lineHeight)
	})
	return Events{pinned: s.pinned - 1, ok: s.pinned > 0}
}

// same reports whether a and b are the same tokens, as the app passes
// them again from frame to frame.
func same(a, b []Token) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// summary shows how many tokens there are, their mean logprob, the
// perplexity, and the least likely token, which a click pins and scrolls
// to.
func (s *State) summary(c *ui.Context) {
	t := c.Theme()
	n := len(s.tokens)
	ui.Row(c).Wrap().GapX(t.Space(5)).GapY(t.Space(1)).AlignItems(ui.Center).FontSize(t.FontSize - 1).Children(func() {
		count := strconv.Itoa(n) + " tokens"
		if n == 1 {
			count = "1 token"
		}
		ui.Text(c, count).FontFeatures("tnum")
		if n == 0 {
			return
		}
		stat(c, "Mean logprob", number(s.mean, 3))
		stat(c, "Perplexity", strconv.FormatFloat(math.Exp(-s.mean), 'f', 2, 64))
		ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Lowest").TextColor(t.TextMuted)
			lo := s.tokens[s.lowest]
			b := ui.ButtonBase(c).Gap(t.Space(1.5)).Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Space(1.5)).
				Label("Lowest token").Description(strconv.Quote(lo.Text) + ", " + percent(s.probs[s.lowest]))
			if b.Clicked() {
				s.pinned, s.reveal = s.lowest+1, true
			}
			if b.Hovered() {
				b.Background(t.Text.Alpha(0.06))
			}
			b.Children(func() {
				ui.RichText(c, visible(lo.Text, t.TextMuted)...).Font("monospace").SingleLine()
				ui.Text(c, percent(s.probs[s.lowest])).TextColor(t.Danger).FontFeatures("tnum")
			})
		})
	})
}

// stat shows name, muted, and value.
func stat(c *ui.Context, name, value string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(1.5)).Children(func() {
		ui.Text(c, name).TextColor(t.TextMuted)
		ui.Text(c, value).FontFeatures("tnum")
	})
}

// text creates the tokens on lines lineH tall, and the card of the token
// under the pointer or pinned. A click on a token pins it, or unpins it
// when pinned; a click beside the tokens or outside the view unpins.
func (s *State) text(c *ui.Context, lineH float32) {
	t := c.Theme()
	box := ui.Box(c).Focusable().Label(s.plain)
	if w := box.LayoutBounds().W; w != s.width {
		s.layout(w)
	}
	box.Height(float32(s.lines) * lineH)
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
	pinned := s.pinned - 1
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The pieces were laid out in the width of the last frame.
		if r.W != s.width {
			c.Invalidate()
		}
		bh := min(lineH-2, s.textH+4)
		pad, ty := (lineH-bh)/2, (lineH-s.textH)/2
		for k, pc := range s.pieces {
			x, w := pc.across()
			x, y := r.X+x, r.Y+float32(pc.line)*lineH
			p.Fill(ui.Rect{X: x, Y: y + pad, W: w, H: bh}, s.tint(t, pc.tok), 0)
			if s.Boundaries && pc.x > 0 && (k == 0 || s.pieces[k-1].tok != pc.tok) {
				p.Fill(ui.Rect{X: x - 0.5, Y: y + pad, W: 1, H: bh}, t.Text.Alpha(0.3), 0)
			}
			if pc.text != nil {
				p.TextLayout(pc.text, r.X+pc.x, y+ty, t.Text)
			}
			if pc.nl {
				p.TextLayout(s.mark, x+w-s.markW, y+ty, t.TextMuted.Alpha(0.7))
			}
			if s.probs[pc.tok] < s.Threshold {
				p.Fill(ui.Rect{X: x, Y: y + ty + s.ascent + 2, W: w, H: 1.5}, t.Danger, 0)
			}
		}
		outline := func(i int, color ui.Color, width float32) {
			k, ok := s.first(i)
			for ; ok && k < len(s.pieces) && s.pieces[k].tok == i; k++ {
				pc := s.pieces[k]
				x, w := pc.across()
				// A pixel wider each way, so that the line clears the glyphs
				// at the token's ends.
				p.Stroke(ui.Rect{X: r.X + x - 1, Y: r.Y + float32(pc.line)*lineH + pad, W: w + 2, H: bh}, color, 2, width)
			}
		}
		if hot != pinned {
			outline(hot, t.Text.Alpha(0.5), 1)
		}
		outline(pinned, t.Accent, 1.5)
	})
	shown := hot
	if pinned >= 0 {
		shown = pinned
	}
	k, ok := s.first(shown)
	if shown < 0 || !ok {
		return
	}
	pc := s.pieces[k]
	x, w := pc.across()
	box.Children(func() {
		anchor := ui.Box(c).Absolute().PassThrough().Left(x).Top(float32(pc.line)*lineH).Size(w, lineH)
		if s.reveal {
			anchor.ScrollIntoView()
			s.reveal = false
		}
		s.card(c, anchor, shown, pinned >= 0)
	})
}

// card shows by anchor the card of token i: pinned, it takes the keys
// that move and unpin it; following the pointer, it lets the pointer
// through to the tokens under it.
func (s *State) card(c *ui.Context, anchor ui.Element, i int, pinned bool) {
	t := c.Theme()
	tk := s.tokens[i]
	// Every element in the card takes the pointer, unless let through, and
	// would take the hover, and the card, from the tokens under it.
	el := func(e ui.Element) ui.Element {
		if !pinned {
			e.PassThrough()
		}
		return e
	}
	ui.Overlay(c, func() {
		tip := el(ui.Column(c).Absolute().Role(ui.RoleTooltip).Width(t.Space(76)).
			AttachTo(anchor, ui.AnchorBottomStart, ui.AnchorTopStart).Margin(t.Space(1), 0, 0, 0).
			Padding(t.Space(3)).Gap(t.Space(2)).Radius(t.Space(2.5)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18)))
		if pinned {
			s.keys(c, tip, i)
		}
		tip.Children(func() {
			el(ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center)).Children(func() {
				el(ui.RichText(c, visible(tk.Text, t.TextMuted)...)).Font("monospace").SingleLine().Shrink(1).
					Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Space(1)).Background(t.Text.Alpha(0.07))
				el(ui.Box(c)).Grow(1)
				el(ui.Text(c, percent(s.probs[i]))).Bold().FontSize(t.FontSize + 3).FontFeatures("tnum")
			})
			el(ui.Text(c, "Logprob "+number(tk.Logprob, 3)+" · Entropy "+strconv.FormatFloat(s.bits[i], 'f', 2, 64)+" bits")).
				FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
			top := tk.Top
			if !slices.ContainsFunc(top, func(a Alternative) bool { return a.Text == tk.Text }) {
				top = append(slices.Clip(top), Alternative{Text: tk.Text, Logprob: tk.Logprob})
			}
			el(ui.Divider(c))
			for k, a := range top {
				alternative(c.Key(k), a, a.Text == tk.Text, el)
			}
		})
	})
}

// keys moves the pin from token i as the arrow keys say, and takes it out
// on Escape.
func (s *State) keys(c *ui.Context, tip ui.Element, i int) {
	to := i
	switch {
	case tip.OverlayShortcut(0, ui.KeyEscape):
		s.pinned = 0
		c.Invalidate()
		return
	case tip.OverlayShortcut(0, ui.KeyLeft):
		to = i - 1
	case tip.OverlayShortcut(0, ui.KeyRight):
		to = i + 1
	case tip.OverlayShortcut(0, ui.KeyUp):
		to = s.vertical(i, -1)
	case tip.OverlayShortcut(0, ui.KeyDown):
		to = s.vertical(i, 1)
	}
	if to != i && to >= 0 && to < len(s.tokens) {
		s.pinned, s.reveal = to+1, true
		c.Invalidate()
	}
}

// alternative shows a row of a card: a check when a is the token chosen,
// a's text, a bar of its probability, in the accent color when chosen,
// and its share. el lets the pointer through the elements, or not.
func alternative(c *ui.Context, a Alternative, chosen bool, el func(ui.Element) ui.Element) {
	t := c.Theme()
	prob := math.Exp(a.Logprob)
	bar := t.TextMuted.Alpha(0.5)
	if chosen {
		bar = t.Accent
	}
	el(ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center)).Children(func() {
		el(ui.Box(c).Size(t.Space(3), t.Space(3)).Shrink(0)).Draw(func(p *ui.Painter, r ui.Rect) {
			if !chosen {
				return
			}
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.32
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.5, t.Accent)
		})
		spans := visible(a.Text, t.TextMuted)
		if chosen {
			for k := range spans {
				spans[k].Weight = 700
			}
		}
		el(ui.RichText(c, spans...)).Font("monospace").FontSize(t.FontSize - 1).SingleLine().Width(t.Space(28)).Shrink(0)
		el(ui.Box(c).Height(t.Space(1.5)).Grow(1).Radius(t.Space(0.75)).Background(t.Text.Alpha(0.08))).
			Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * float32(prob), H: r.H}, bar, r.H/2)
			})
		el(ui.Text(c, percent(prob))).FontSize(t.FontSize - 1).FontFeatures("tnum").
			Width(t.Space(12)).Shrink(0).TextAlign(ui.End)
	})
}

// visible returns text in spans, its spaces, newlines and tabs as ␣, ↵ and
// ⇥ in muted, so that a token of them shows.
func visible(text string, muted ui.Color) []ui.Span {
	var spans []ui.Span
	last := false
	for _, r := range text {
		sym, ws := string(r), true
		switch r {
		case ' ':
			sym = "␣"
		case '\n':
			sym = "↵"
		case '\t':
			sym = "⇥"
		default:
			ws = false
		}
		if len(spans) > 0 && ws == last {
			spans[len(spans)-1].Text += sym
			continue
		}
		sp := ui.Span{Text: sym}
		if ws {
			sp.Color = muted
		}
		spans, last = append(spans, sp), ws
	}
	return spans
}

// percent writes p, from 0 to 1, as "62.4%", "<0.1%" or "100%".
func percent(p float64) string {
	v := 100 * p
	switch {
	case v < 0.05:
		return "<0.1%"
	case v >= 99.95:
		return "100%"
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

// number writes v with decimals digits after the point, negative with a
// minus sign, "−0.472".
func number(v float64, decimals int) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', decimals, 64), "-", "−", 1)
}

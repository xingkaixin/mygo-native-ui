// Package tokenizer shows a text as a model's tokenizer splits it, as
// OpenAI's Tokenizer page and tiktokenizer show it: how many tokens and
// characters it has, and the text with each token tinted, by turns in
// five colors, or the tokens' IDs. The pointer on a token tells its ID
// and lights every token with the same one. Spaces show as dots and line
// breaks as arrows with Whitespace on. A token of bytes that are only
// part of a character, as a byte-level tokenizer makes of an emoji,
// shows its bytes in hex.
//
// The app tokenizes the text with the model's own tokenizer:
//
//	ids := enc.Encode(app.prompt)
//	tokens := make([]tokenizer.Token, len(ids))
//	for i, id := range ids {
//		tokens[i] = tokenizer.Token{Text: enc.Decode(id), ID: id}
//	}
//	tokenizer.View(c, &app.tokens, tokens)
package tokenizer

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Token is a token: the text it stands for, which may be bytes that are
// part of a character, and its ID.
type Token struct {
	Text string
	ID   int
}

// State is what a view keeps: whether it shows the text or the IDs, and
// whitespace, and the text laid out. Keep it in the app's state, one for
// each view.
type State struct {
	ids        bool
	whitespace bool

	tokens []Token
	shown  bool // the whitespace of the layout
	font   ui.Font
	width  float32
	shape  *ui.TextLayout
	parts  []ui.Span
	starts []int // where each token starts in the layout, in UTF-16, and where the last ends
	rects  [][]ui.Rect
}

// View creates the view of tokens.
func View(c *ui.Context, s *State, tokens []Token) ui.Element {
	t := c.Theme()
	chars := 0
	for _, tk := range tokens {
		chars += utf8.RuneCountInString(tk.Text)
	}
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(6)).AlignItems(ui.End).Children(func() {
			stat(c, "Tokens", len(tokens))
			stat(c, "Characters", chars)
			ui.Spacer(c)
			ui.Checkbox(c, &s.whitespace, "Whitespace")
			view := 0
			if s.ids {
				view = 1
			}
			if ui.Segmented(c, &view, "Text", "IDs").Label("Show").Changed() {
				s.ids = view == 1
			}
		})
		box := ui.Column(c).Padding(t.Space(3)).Radius(t.Radius).Border(1, t.Border).Background(t.Surface)
		box.Children(func() {
			if s.ids {
				ids := make([]string, len(tokens))
				for i, tk := range tokens {
					ids[i] = strconv.Itoa(tk.ID)
				}
				ui.Text(c, "["+strings.Join(ids, ", ")+"]").Font("monospace").Selectable()
				return
			}
			s.text(c, tokens)
		})
	})
}

// stat shows n under its name.
func stat(c *ui.Context, name string, n int) {
	t := c.Theme()
	ui.Column(c).Children(func() {
		ui.Text(c, name).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		ui.Text(c, group(n)).FontSize(t.FontSize + 8).Bold().FontFeatures("tnum")
	})
}

// group writes n with commas: 12,400.
func group(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// marks are what spaces, tabs and line breaks show as with whitespace
// on.
var marks = map[rune]string{' ': "·", '\t': "→", '\n': "↵\n"}

// shown returns the runs of text a token shows: its bytes in hex where
// they are not whole characters, and its spaces, tabs and line breaks as
// marks when whitespace shows, both muted in t's colors.
func shown(text string, whitespace bool, font ui.Font, t *ui.Theme) []ui.Span {
	if !utf8.ValidString(text) {
		var b strings.Builder
		for _, r := range []byte(text) {
			fmt.Fprintf(&b, "<%02X>", r)
		}
		return []ui.Span{{Text: b.String(), Color: t.TextMuted, Size: font.Size - 2}}
	}
	if !whitespace {
		return []ui.Span{{Text: text}}
	}
	var out []ui.Span
	from := 0
	for i, r := range text {
		mark, ok := marks[r]
		if !ok {
			continue
		}
		if from < i {
			out = append(out, ui.Span{Text: text[from:i]})
		}
		out = append(out, ui.Span{Text: mark, Color: t.TextMuted})
		from = i + utf8.RuneLen(r)
	}
	if from < len(text) {
		out = append(out, ui.Span{Text: text[from:]})
	}
	return out
}

// text creates the text of tokens, each tinted.
func (s *State) text(c *ui.Context, tokens []Token) {
	t := c.Theme()
	font := ui.Font{Family: "monospace", Size: t.FontSize}
	if !slices.Equal(tokens, s.tokens) || s.whitespace != s.shown || font != s.font {
		s.prepare(tokens, s.whitespace, font, t)
	}
	box := ui.Box(c).AlignSelf(ui.Stretch).Role(ui.RoleImage).Label(group(len(tokens)) + " tokens")
	if w := box.LayoutBounds().W; w > 0 && w != s.width {
		s.layout(w)
	}
	if s.shape != nil {
		_, h := s.shape.Size()
		box.Height(h)
	}
	hot := -1
	if x, y, ok := box.PointerPosition(); ok && box.Hovered() && s.shape != nil {
		hot = s.tokenAt(ui.Point{X: x, Y: y})
		if hot >= 0 {
			tk := s.tokens[hot]
			box.Tooltip(fmt.Sprintf("%d · %s", tk.ID, strconv.Quote(tk.Text)))
		}
	}
	dark := t.Dark
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if s.shape == nil {
			return
		}
		for i, rects := range s.rects {
			fill := tint(i, dark)
			if hot >= 0 && s.tokens[i].ID == s.tokens[hot].ID {
				fill = t.Accent.Alpha(0.35)
			}
			for _, b := range rects {
				p.Fill(ui.Rect{X: r.X + b.X, Y: r.Y + b.Y + 1, W: b.W, H: b.H - 2}, fill, 2)
				if i == hot {
					p.Stroke(ui.Rect{X: r.X + b.X, Y: r.Y + b.Y + 1, W: b.W, H: b.H - 2}, t.Accent, 2, 1.5)
				}
			}
		}
		p.TextLayout(s.shape, r.X, r.Y, t.Text)
	})
}

// tint returns the color of the i-th token.
func tint(i int, dark bool) ui.Color {
	hue := []float32{250, 145, 60, 330, 195}[i%5]
	if dark {
		return ui.Oklch(0.38, 0.07, hue)
	}
	return ui.Oklch(0.9, 0.06, hue)
}

// prepare starts over for tokens, showing whitespace or not, in font.
func (s *State) prepare(tokens []Token, whitespace bool, font ui.Font, t *ui.Theme) {
	s.tokens, s.shown, s.font, s.width, s.shape = slices.Clone(tokens), whitespace, font, -1, nil
	s.parts = s.parts[:0]
	s.starts = append(s.starts[:0], 0)
	for _, tk := range tokens {
		units := 0
		for _, part := range shown(tk.Text, whitespace, font, t) {
			s.parts = append(s.parts, part)
			for _, r := range part.Text {
				units += utf16.RuneLen(r)
			}
		}
		s.starts = append(s.starts, s.starts[len(s.starts)-1]+units)
	}
}

// layout lays the text out width wide, and finds each token's boxes.
func (s *State) layout(width float32) {
	s.width = width
	s.shape = ui.ShapeRichText(s.parts, s.font, width)
	s.rects = make([][]ui.Rect, len(s.tokens))
	for i := range s.tokens {
		s.rects[i] = s.shape.SelectionRects(ui.TextInputRange{Start: s.starts[i], End: s.starts[i+1]})
	}
}

// tokenAt returns the index of the token at pt, or -1.
func (s *State) tokenAt(pt ui.Point) int {
	for i, rects := range s.rects {
		for _, b := range rects {
			if b.Contains(pt.X, pt.Y) {
				return i
			}
		}
	}
	return -1
}

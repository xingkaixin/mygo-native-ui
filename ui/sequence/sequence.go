// Package sequence draws a sequence diagram from text, as Mermaid's
// sequenceDiagram does: the participants in a row at the top and again at
// the bottom, boxes or, for actors, stick people, with dashed lifelines
// between them, and down them the messages in order, each an arrow with
// its text above it. Activations show as narrow bars on the lifelines,
// notes as tinted boxes, and loops, alternatives and options as frames
// around their messages. The pointer lights the message under it. A
// diagram too big to fit scrolls both ways. The lines that are wrong are
// listed under it, by number with what is wrong, and the diagram shows
// what the others say.
//
//	sequence.View(c, &app.diagram, app.source).Grow(1)
//
// The source has a statement a line; lines starting with %% are
// comments:
//
//	participant A               a participant, in the order declared
//	participant A as Long name  one shown by another name
//	actor U as User             one shown as a stick person
//	A->>B: text                 a call: a line with a filled arrowhead
//	B-->>A: text                a reply: a dashed one
//	A->B: text, A-->B: text     asynchronous: an open arrowhead
//	A->>A: text                 a message to itself
//	A->>+B: text                activates B, until B-->>-A: text
//	Note right of A: text       or left of A, over A, over A,B
//	loop text ... end           a loop, around the lines up to end
//	alt text ... else text ... end, opt text ... end
//
// Participants not declared come in the order they are first named.
package sequence

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// State is what a view keeps: the diagram as it parsed and laid out the
// source it was last given. Keep it in the app's state, one for each view.
type State struct {
	key key
	d   *diagram
}

// key is what a layout is for: the source, and the theme's sizes.
type key struct {
	src         string
	font, space float32
}

type participant struct {
	name, label string
	actor       bool
	// w is the width of the header, x the lifeline.
	w, x float32
}

type kind uint8

const (
	message kind = iota
	note
	open
	divider
	close
)

// event is a row of the diagram: a message, a note, or where a frame
// opens, divides or closes.
type event struct {
	kind kind
	text string
	tw   float32
	// from and to are the participants of a message, or the first and
	// last of those a note is beside or over.
	from, to       int
	dashed, hollow bool
	// fromDepth and toDepth are how many activations are on from as the
	// message leaves or the note is beside it, and on to as it arrives.
	fromDepth, toDepth int
	// side is where a note is: 'l' left of from, 'r' right of it, 'o' over
	// from to to.
	side  byte
	frame int
	// A message goes from x0 to x1, at y, and for one to itself comes back
	// at y2; a note is a box from x0 to x1 and y to y2. lo and hi are how
	// far the row goes left and right.
	x0, x1, y, y2, lo, hi float32
}

// frame is a loop, alt or opt: its word and condition, the line and the
// event it opens at, how wide its tab is, and its box.
type frame struct {
	word, text string
	line       int
	first      int
	tab        float32
	x0, x1     float32
	y0, y1     float32
}

// bar is an activation of a participant, nested depth deep, from the
// message at event from to that at event to, or to the end.
type bar struct {
	part, depth int
	from, to    int
	y0, y1      float32
}

type problem struct {
	line int
	msg  string
}

type diagram struct {
	parts  []participant
	events []event
	frames []frame
	bars   []bar
	errs   []problem
	// ox moves the layout right into the diagram; headH is how tall the
	// headers are, bottom where the bottom ones start.
	ox, w, h, headH, bottom float32
	z                       sizes
}

var arrows = []string{"-->>", "->>", "-->", "->"}

// parse reads the statements of src, skipping those that are wrong.
func parse(src string) *diagram {
	d := &diagram{}
	find := func(name string) int {
		for i, p := range d.parts {
			if p.name == name {
				return i
			}
		}
		d.parts = append(d.parts, participant{name: name, label: name})
		return len(d.parts) - 1
	}
	fail := func(line int, format string, args ...any) {
		d.errs = append(d.errs, problem{line, fmt.Sprintf(format, args...)})
	}
	var frames []int
	active := map[int][]int{}
	for n, raw := range strings.Split(src, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		word, rest := s, ""
		if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
			word, rest = s[:i], strings.TrimSpace(s[i:])
		}
		switch w := strings.ToLower(word); w {
		case "sequencediagram":
		case "participant", "actor":
			name, label, _ := strings.Cut(rest, " as ")
			name, label = strings.TrimSpace(name), strings.TrimSpace(label)
			if name == "" {
				fail(line, "%s needs a name", w)
				continue
			}
			i := find(name)
			if label != "" {
				d.parts[i].label = label
			}
			d.parts[i].actor = w == "actor"
		case "note":
			head, text, _ := strings.Cut(rest, ":")
			f := strings.Fields(head)
			var names string
			e := event{kind: note, text: strings.TrimSpace(text)}
			switch {
			case len(f) > 2 && strings.EqualFold(f[1], "of") && strings.EqualFold(f[0], "right"):
				e.side, names = 'r', strings.Join(f[2:], " ")
			case len(f) > 2 && strings.EqualFold(f[1], "of") && strings.EqualFold(f[0], "left"):
				e.side, names = 'l', strings.Join(f[2:], " ")
			case len(f) > 1 && strings.EqualFold(f[0], "over"):
				e.side, names = 'o', strings.Join(f[1:], " ")
			default:
				fail(line, `a note goes "right of", "left of" or "over" a participant`)
				continue
			}
			list := strings.Split(names, ",")
			for k := range list {
				list[k] = strings.TrimSpace(list[k])
			}
			if len(list) > 2 || len(list) == 2 && e.side != 'o' {
				fail(line, "a note goes beside one participant, or over one or two")
				continue
			}
			if slices.Contains(list, "") {
				fail(line, "a participant's name is missing")
				continue
			}
			e.from = find(list[0])
			e.to = find(list[len(list)-1])
			e.from, e.to = min(e.from, e.to), max(e.from, e.to)
			e.fromDepth = len(active[e.from])
			d.events = append(d.events, e)
		case "loop", "alt", "opt":
			frames = append(frames, len(d.frames))
			d.frames = append(d.frames, frame{word: w, text: rest, line: line, first: len(d.events)})
			d.events = append(d.events, event{kind: open, frame: len(d.frames) - 1})
		case "else":
			if len(frames) == 0 || d.frames[frames[len(frames)-1]].word != "alt" {
				fail(line, "else is outside an alt")
				continue
			}
			d.events = append(d.events, event{kind: divider, text: rest, frame: frames[len(frames)-1]})
		case "end":
			if len(frames) == 0 {
				fail(line, "end closes no loop, alt or opt")
				continue
			}
			d.events = append(d.events, event{kind: close, frame: frames[len(frames)-1]})
			frames = frames[:len(frames)-1]
		default:
			from, arrow, mark, to, text, err := split(s)
			if err != "" {
				fail(line, "%s", err)
				continue
			}
			e := event{kind: message, text: text, from: find(from), to: find(to),
				dashed: strings.HasPrefix(arrow, "--"), hollow: !strings.HasSuffix(arrow, ">>")}
			e.fromDepth = len(active[e.from])
			switch mark {
			case '+':
				active[e.to] = append(active[e.to], len(d.bars))
				d.bars = append(d.bars, bar{part: e.to, depth: len(active[e.to]) - 1, from: len(d.events), to: -1})
			case '-':
				stack := active[e.from]
				if len(stack) == 0 {
					fail(line, "%s is not active", from)
					break
				}
				d.bars[stack[len(stack)-1]].to = len(d.events)
				active[e.from] = stack[:len(stack)-1]
			}
			e.toDepth = len(active[e.to])
			d.events = append(d.events, e)
		}
	}
	for k := len(frames) - 1; k >= 0; k-- {
		f := d.frames[frames[k]]
		fail(f.line, "%s has no end", f.word)
		d.events = append(d.events, event{kind: close, frame: frames[k]})
	}
	slices.SortStableFunc(d.errs, func(a, b problem) int { return cmp.Compare(a.line, b.line) })
	return d
}

// split reads a message, "A->>+B: text", or says what is wrong with it.
func split(s string) (from, arrow string, mark byte, to, text, err string) {
	head, text, _ := strings.Cut(s, ":")
	text = strings.TrimSpace(text)
	i := strings.Index(head, "-")
	if i < 0 {
		return "", "", 0, "", "", fmt.Sprintf("%q is not a participant, message, note or frame", s)
	}
	from, rest := strings.TrimSpace(head[:i]), head[i:]
	for _, a := range arrows {
		if strings.HasPrefix(rest, a) {
			arrow = a
			break
		}
	}
	if arrow == "" {
		return "", "", 0, "", "", fmt.Sprintf("the arrow after %q is not ->>, -->>, -> or -->", from)
	}
	rest = strings.TrimSpace(rest[len(arrow):])
	if rest != "" && (rest[0] == '+' || rest[0] == '-') {
		mark, rest = rest[0], strings.TrimSpace(rest[1:])
	}
	switch {
	case from == "":
		err = "the message has no sender"
	case rest == "":
		err = "the message has no receiver"
	}
	return from, arrow, mark, rest, text, err
}

// sizes are the sizes of the parts of a diagram, from the theme.
type sizes struct {
	th, padX, padY, boxH, figH, barW, loopW, loopH, gapY, tabH, inset, pad float32
}

func measure(c *ui.Context) sizes {
	t := c.Theme()
	_, th := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize})
	return sizes{th: th, padX: t.Space(3), padY: t.Space(1.5), boxH: th + 2*t.Space(1.5), figH: t.Space(8),
		barW: t.Space(2.5), loopW: t.Space(8), loopH: t.Space(5), gapY: t.Space(3.5), tabH: th + 2*t.Space(1),
		inset: t.Space(3), pad: t.Space(5)}
}

// edge returns how far from its lifeline a message meets a participant
// with depth activations, on the right or the left of the top one.
func (z sizes) edge(depth int, right bool) float32 {
	if depth == 0 {
		return 0
	}
	off := float32(depth-1) * z.barW / 2
	if right {
		return off + z.barW/2
	}
	return off - z.barW/2
}

// layout places the participants as far apart as the text between them
// needs, and the rows down the lifelines.
func layout(c *ui.Context, d *diagram) {
	t := c.Theme()
	z := measure(c)
	d.z = z
	text := func(s string, weight int) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: t.FontSize, Weight: weight})
		return w
	}
	d.headH = z.boxH
	for i := range d.parts {
		p := &d.parts[i]
		p.w = max(text(p.label, 600)+2*z.padX, t.Space(16))
		if p.actor {
			d.headH = max(d.headH, z.figH+t.Space(1)+z.th+z.padY)
		}
	}
	n := len(d.parts)
	gaps := make([]float32, max(n-1, 0))
	for i := range gaps {
		gaps[i] = (d.parts[i].w+d.parts[i+1].w)/2 + t.Space(8)
	}
	type need struct {
		a, b int
		v    float32
	}
	var needs []need
	for i := range d.events {
		e := &d.events[i]
		e.tw = text(e.text, 400)
		nw := e.tw + 2*z.padX
		switch {
		case e.kind == message && e.from != e.to:
			needs = append(needs, need{min(e.from, e.to), max(e.from, e.to), e.tw + t.Space(10)})
		case e.kind == message:
			needs = append(needs, need{e.from, e.from + 1, z.edge(e.fromDepth, true) + max(z.loopW, e.tw+t.Space(1)) + t.Space(6)})
		case e.kind == note && e.side == 'r':
			needs = append(needs, need{e.from, e.from + 1, z.edge(e.fromDepth, true) + nw + t.Space(6)})
		case e.kind == note && e.side == 'l':
			needs = append(needs, need{e.from - 1, e.from, nw + t.Space(6)})
		case e.kind == note && e.from == e.to:
			needs = append(needs, need{e.from - 1, e.from, nw/2 + t.Space(4)}, need{e.from, e.from + 1, nw/2 + t.Space(4)})
		case e.kind == note:
			needs = append(needs, need{e.from, e.to, nw - 2*t.Space(6)})
		}
	}
	slices.SortStableFunc(needs, func(a, b need) int { return cmp.Compare(a.b-a.a, b.b-b.a) })
	for _, nd := range needs {
		if nd.a < 0 || nd.b >= n {
			continue
		}
		var sum float32
		for k := nd.a; k < nd.b; k++ {
			sum += gaps[k]
		}
		for k := nd.a; k < nd.b && sum < nd.v; k++ {
			gaps[k] += (nd.v - sum) / float32(nd.b-nd.a)
		}
	}
	for i := 1; i < n; i++ {
		d.parts[i].x = d.parts[i-1].x + gaps[i-1]
	}
	lo, hi := float32(0), float32(0)
	for _, p := range d.parts {
		lo, hi = min(lo, p.x-p.w/2), max(hi, p.x+p.w/2)
	}
	y := z.pad + d.headH + t.Space(4)
	for i := range d.events {
		e := &d.events[i]
		var px float32
		if e.kind == message || e.kind == note {
			px = d.parts[e.from].x
		}
		switch e.kind {
		case message:
			e.y = y + z.th + t.Space(1)
			e.y2 = e.y
			if e.from == e.to {
				e.x0 = px + z.edge(e.fromDepth, true)
				e.x1 = px + z.edge(e.toDepth, true)
				e.y2 = e.y + z.loopH
				e.lo, e.hi = px, e.x0+max(z.loopW, e.tw+t.Space(1))
			} else {
				to := d.parts[e.to].x
				right := to > px
				e.x0, e.x1 = px+z.edge(e.fromDepth, right), to+z.edge(e.toDepth, !right)
				e.lo, e.hi = min(px, to), max(px, to)
			}
			y = e.y2 + z.gapY
			lo, hi = min(lo, e.lo), max(hi, e.hi)
		case note:
			nw := e.tw + 2*z.padX
			switch {
			case e.side == 'r':
				e.x0 = px + z.edge(e.fromDepth, true) + t.Space(2)
				e.x1 = e.x0 + nw
			case e.side == 'l':
				e.x1 = px + min(0, z.edge(e.fromDepth, false)) - t.Space(2)
				e.x0 = e.x1 - nw
			default:
				x1, ext := d.parts[e.to].x, float32(0)
				if e.to > e.from {
					ext = t.Space(6)
				}
				mid := (px + x1) / 2
				e.x0, e.x1 = min(mid-nw/2, px-ext), max(mid+nw/2, x1+ext)
			}
			e.lo, e.hi = e.x0, e.x1
			e.y, e.y2 = y, y+z.th+2*z.padY
			y = e.y2 + z.gapY
			lo, hi = min(lo, e.lo), max(hi, e.hi)
		case open:
			d.frames[e.frame].y0 = y
			y += z.tabH + t.Space(3)
		case divider:
			e.y = y - t.Space(1)
			y = e.y + t.Space(1.5) + z.th + t.Space(3)
		case close:
			f := &d.frames[e.frame]
			f.y1 = y - t.Space(1.5)
			y = f.y1 + t.Space(4)
			flo, fhi := float32(0), float32(0)
			found := false
			for _, in := range d.events[f.first+1 : i] {
				if in.kind == close {
					in.lo, in.hi = d.frames[in.frame].x0, d.frames[in.frame].x1
				} else if in.kind != message && in.kind != note {
					continue
				}
				if !found {
					flo, fhi, found = in.lo, in.hi, true
				}
				flo, fhi = min(flo, in.lo), max(fhi, in.hi)
			}
			if !found && n > 0 {
				flo, fhi = d.parts[0].x, d.parts[n-1].x
			}
			f.tab = text(f.word, 600) + 2*t.Space(2) + t.Space(1.5)
			f.x0, f.x1 = flo-z.inset, fhi+z.inset
			labels := []string{f.text}
			for _, in := range d.events[f.first+1 : i] {
				if in.kind == divider && in.frame == e.frame {
					labels = append(labels, in.text)
				}
			}
			for _, l := range labels {
				f.x1 = max(f.x1, f.x0+f.tab+t.Space(2)+text(cond(l), 400)+t.Space(3))
			}
			lo, hi = min(lo, f.x0), max(hi, f.x1)
		}
	}
	for i := range d.bars {
		b := &d.bars[i]
		b.y0, b.y1 = d.events[b.from].y2, y-t.Space(2)
		if b.to >= 0 {
			b.y1 = d.events[b.to].y
		}
	}
	d.bottom = y + t.Space(2)
	d.ox = z.pad - lo
	d.w, d.h = hi-lo+2*z.pad, d.bottom+d.headH+z.pad
}

// cond returns the condition of a frame as it shows, in brackets.
func cond(text string) string {
	if text == "" {
		return ""
	}
	return "[" + text + "]"
}

// View creates the diagram that src describes. Give it a size, or Grow;
// it scrolls when it does not fit, and the lines that are wrong show
// under it.
func View(c *ui.Context, s *State, src string) ui.Element {
	t := c.Theme()
	k := key{src, t.FontSize, t.Space(1)}
	if s.d == nil || s.key != k {
		s.key, s.d = k, parse(src)
		layout(c, s.d)
	}
	d, z := s.d, s.d.z
	return ui.Column(c).Children(func() {
		ui.ScrollBoth(c).Grow(1).Shrink(1).Children(func() {
			box := ui.Box(c).Size(d.w, d.h)
			hot := -1
			if px, py, ok := box.PointerPosition(); ok {
				px -= d.ox
				for i, e := range d.events {
					if e.kind == message && px >= e.lo-4 && px <= e.hi+4 && py >= e.y-z.th-t.Space(2) && py <= e.y2+t.Space(2) {
						hot = i
					}
				}
			}
			box.Draw(func(p *ui.Painter, r ui.Rect) {
				draw(p, t, z, d, r.X+d.ox, r.Y, hot)
			})
		})
		if len(d.errs) == 0 {
			return
		}
		ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(3)).Children(func() {
			for i, e := range d.errs {
				ui.Text(c.Key(i), fmt.Sprintf("Line %d: %s", e.line, e.msg)).FontSize(t.FontSize - 1).TextColor(t.Danger)
			}
		})
	})
}

// draw paints d with its left at ox and top at oy, lighting message hot.
func draw(p *ui.Painter, t *ui.Theme, z sizes, d *diagram, ox, oy float32, hot int) {
	line := t.TextMuted.Alpha(0.55)
	label := func(x, y float32, s string, weight int, color ui.Color) {
		p.RichText(x, y, 0, ui.Span{Text: s, Size: t.FontSize, Weight: weight, Color: color})
	}
	top, bottom := oy+z.pad+d.headH, oy+d.bottom
	for _, pt := range d.parts {
		dashed(p, ox+pt.x, top, ox+pt.x, bottom, 1, line)
	}
	for _, b := range d.bars {
		x := ox + d.parts[b.part].x + float32(b.depth)*z.barW/2 - z.barW/2
		rect := ui.Rect{X: x, Y: oy + b.y0, W: z.barW, H: max(b.y1-b.y0, t.Space(2))}
		p.Fill(rect, t.Background, 0)
		p.Fill(rect, t.TextMuted.Alpha(0.16), 0)
		p.Stroke(rect, t.TextMuted.Alpha(0.8), 0, 1)
	}
	for _, f := range d.frames {
		x0, x1, y0, y1 := ox+f.x0, ox+f.x1, oy+f.y0, oy+f.y1
		p.Stroke(ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, line, 0, 1)
		notch := t.Space(1.5)
		var tab ui.Path
		tab.MoveTo(x0, y0).LineTo(x0+f.tab, y0).LineTo(x0+f.tab, y0+z.tabH-notch).
			LineTo(x0+f.tab-notch, y0+z.tabH).LineTo(x0, y0+z.tabH).Close()
		p.FillPath(&tab, t.Background)
		p.FillPath(&tab, t.TextMuted.Alpha(0.14))
		p.StrokePath(&tab, 1, line)
		label(x0+t.Space(2), y0+(z.tabH-z.th)/2, f.word, 600, t.Text)
		label(x0+f.tab+t.Space(2), y0+(z.tabH-z.th)/2, cond(f.text), 400, t.TextMuted)
	}
	for _, e := range d.events {
		if e.kind == divider {
			f := d.frames[e.frame]
			dashed(p, ox+f.x0, oy+e.y, ox+f.x1, oy+e.y, 1, line)
			label(ox+f.x0+f.tab+t.Space(2), oy+e.y+t.Space(1.5), cond(e.text), 400, t.TextMuted)
		}
	}
	for _, e := range d.events {
		if e.kind != note {
			continue
		}
		rect := ui.Rect{X: ox + e.x0, Y: oy + e.y, W: e.x1 - e.x0, H: e.y2 - e.y}
		p.Fill(rect, t.Background, t.Space(1))
		p.Fill(rect, t.Warning.Alpha(0.16), t.Space(1))
		p.Stroke(rect, t.Warning.Alpha(0.6), t.Space(1), 1)
		label(rect.X+(rect.W-e.tw)/2, rect.Y+z.padY, e.text, 400, t.Text)
	}
	for i, e := range d.events {
		if e.kind != message {
			continue
		}
		color, width, ink := t.Text.Alpha(0.75), float32(1.25), t.Text
		if i == hot {
			color, width, ink = t.Accent, 2, t.Accent
		}
		y, y2 := oy+e.y, oy+e.y2
		head := t.Space(2.5)
		if e.hollow {
			head = 0
		}
		if e.from == e.to {
			x0, x1, xr := ox+e.x0, ox+e.x1, ox+e.x0+z.loopW
			if e.dashed {
				dashed(p, x0, y, xr, y, width, color)
				dashed(p, xr, y, xr, y2, width, color)
				dashed(p, xr, y2, x1+head/2, y2, width, color)
			} else {
				p.StrokePolyline([][2]float32{{x0, y}, {xr, y}, {xr, y2}, {x1 + head/2, y2}}, width, color)
			}
			arrowhead(p, t, x1, y2, -1, e.hollow, width, color)
			label(x0+t.Space(1.5), y-t.Space(1)-z.th, e.text, 400, ink)
			continue
		}
		x0, x1 := ox+e.x0, ox+e.x1
		dir := float32(1)
		if x1 < x0 {
			dir = -1
		}
		if e.dashed {
			dashed(p, x0, y, x1-dir*head/2, y, width, color)
		} else {
			p.Line(x0, y, x1-dir*head/2, y, width, color)
		}
		arrowhead(p, t, x1, y, dir, e.hollow, width, color)
		label((x0+x1-e.tw)/2, y-t.Space(1)-z.th, e.text, 400, ink)
	}
	for _, pt := range d.parts {
		header(p, t, z, pt, ox+pt.x, oy+z.pad, d.headH, true)
		header(p, t, z, pt, ox+pt.x, oy+d.bottom, d.headH, false)
	}
}

// header draws a participant's header, h tall from y, centered on x: a
// box, at the bottom of the row at the top and the top of the row at the
// bottom, or a stick person over its name.
func header(p *ui.Painter, t *ui.Theme, z sizes, pt participant, x, y, h float32, atTop bool) {
	span := ui.Span{Text: pt.label, Size: t.FontSize, Weight: 600, Color: t.Text}
	tw, _ := p.MeasureText(0, span)
	if pt.actor {
		r := t.Space(1.75)
		ink := t.Text.Alpha(0.85)
		var head ui.Path
		head.Circle(x, y+r+1, r)
		p.StrokePath(&head, 1.5, ink)
		hip := y + z.figH*0.66
		p.Line(x, y+2*r+1, x, hip, 1.5, ink)
		p.Line(x-t.Space(2.5), y+z.figH*0.42, x+t.Space(2.5), y+z.figH*0.42, 1.5, ink)
		p.StrokePolyline([][2]float32{{x - t.Space(2), y + z.figH}, {x, hip}, {x + t.Space(2), y + z.figH}}, 1.5, ink)
		p.RichText(x-tw/2, y+z.figH+t.Space(1), 0, span)
		return
	}
	if atTop {
		y += h - z.boxH
	}
	rect := ui.Rect{X: x - pt.w/2, Y: y, W: pt.w, H: z.boxH}
	p.Fill(rect, t.Surface, t.Space(1.5))
	p.Stroke(rect, t.Border, t.Space(1.5), 1)
	p.RichText(x-tw/2, y+z.padY, 0, span)
}

// arrowhead draws the head of an arrow pointing dir, 1 right or -1 left,
// with its tip at x, y: filled, or open as two strokes.
func arrowhead(p *ui.Painter, t *ui.Theme, x, y, dir float32, hollow bool, width float32, color ui.Color) {
	l, h := t.Space(2.5), t.Space(1.25)
	if hollow {
		p.StrokePolyline([][2]float32{{x - dir*l, y - h}, {x, y}, {x - dir*l, y + h}}, width, color)
		return
	}
	var tri ui.Path
	tri.MoveTo(x, y).LineTo(x-dir*l, y-h).LineTo(x-dir*l, y+h).Close()
	p.FillPath(&tri, color)
}

// dashed draws a horizontal or vertical line from x0, y0 to x1, y1 in
// dashes.
func dashed(p *ui.Painter, x0, y0, x1, y1, width float32, color ui.Color) {
	const on, off = 5, 4
	n := max(x1-x0, x0-x1, y1-y0, y0-y1)
	if n == 0 {
		return
	}
	dx, dy := (x1-x0)/n, (y1-y0)/n
	for s := float32(0); s < n; s += on + off {
		e := min(s+on, n)
		p.Line(x0+dx*s, y0+dy*s, x0+dx*e, y0+dy*e, width, color)
	}
}

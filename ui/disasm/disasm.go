// Package disasm shows machine code as instructions, as debuggers and
// disassemblers list it: a row for each instruction with its address, its
// bytes, its mnemonic colored by what it does and its operands, and a
// comment after; a symbol's name above the instruction it starts at. In
// the gutter on the left, an arrow goes from each jump to the instruction
// it jumps to, when that is listed, the arrows nested so that they do not
// cross; the pointer on a jump, or on the instruction it goes to, lights
// its arrow. A click chooses an instruction, and a double click on a jump
// or a call goes to its target, which Escape comes back from.
//
//	app.code.PC = regs.PC
//	disasm.View(c, &app.code, insts).Grow(1)
package disasm

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Inst is an instruction: its address, its bytes, its mnemonic and its
// operands as text. Target is the address a jump or a call goes to, or 0.
// Label names a symbol starting at the instruction, and Comment says what
// the instruction does.
type Inst struct {
	Addr    uint64
	Bytes   []byte
	Op      string
	Args    string
	Target  uint64
	Label   string
	Comment string
}

// State is what a view keeps: where it is scrolled, the instruction
// chosen and the ones gone to from. PC, which the app sets, is the
// address of the instruction a program stopped at, which shows marked, or
// 0. Keep it in the app's state, one for each view.
type State struct {
	PC uint64

	chosen uint64
	back   []uint64
	scroll ui.ScrollState
	hot    uint64
	// seen is the PC shown last, to scroll to a new one.
	seen uint64
}

// Chosen returns the address of the instruction chosen, or false with
// none.
func (s *State) Chosen() (uint64, bool) { return s.chosen, s.chosen != 0 }

// row is a row of the listing: an instruction, with the label row above
// it when it has a label.
type row struct {
	inst  int
	label bool
}

// arrow is a jump from row from to row to, in lane lane of the gutter.
type arrow struct {
	from, to, lane int
	src, dst       uint64
}

// View creates the listing of insts, in the order of their addresses.
// Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, insts []Inst) ui.Element {
	t := c.Theme()
	size := t.FontSize - 1
	rowH := size + t.Space(2.5)
	cw, _ := c.MeasureText(0, ui.Span{Text: "0", Font: "monospace", Size: size})
	var rows []row
	at := map[uint64]int{}
	wide, long := 0, 0
	for i, in := range insts {
		if in.Label != "" {
			rows = append(rows, row{i, true})
		}
		at[in.Addr] = len(rows)
		rows = append(rows, row{i, false})
		wide = max(wide, len(in.Bytes))
		long = max(long, len([]rune(in.Args)))
	}
	arrows := lay(insts, rows, at)
	lanes := 0
	for _, a := range arrows {
		lanes = max(lanes, a.lane+1)
	}
	digits := 1
	if len(insts) > 0 {
		digits = len(fmt.Sprintf("%x", insts[len(insts)-1].Addr))
	}
	lane := t.Space(2.5)
	gutter := float32(lanes)*lane + t.Space(5)
	cols := [5]float32{gutter, gutter + float32(digits+2)*cw, 0, 0, 0}
	cols[2] = cols[1] + float32(min(wide, 8)*3+1)*cw
	cols[3] = cols[2] + 8*cw
	cols[4] = cols[3] + float32(min(long, 40)+2)*cw

	go2 := func(addr uint64) {
		r, ok := at[addr]
		if !ok {
			return
		}
		s.chosen = addr
		_, h := c.Size()
		s.scroll.ScrollTo(0, max(0, float32(r)*rowH-h/3), ui.Smooth)
	}
	if s.PC != s.seen {
		s.seen = s.PC
		if s.PC != 0 {
			go2(s.PC)
			s.chosen = 0
		}
	}

	view := ui.Scroll(c).TrackScroll(&s.scroll).Focusable().FocusRing(false).Label("Disassembly")
	if view.Focused() && c.Shortcut(0, ui.KeyEscape) && len(s.back) > 0 {
		from := s.back[len(s.back)-1]
		s.back = s.back[:len(s.back)-1]
		go2(from)
	}
	view.Children(func() {
		box := ui.Box(c).FillWidth().Height(float32(len(rows))*rowH + t.Space(2)).Cursor(ui.CursorDefault)
		hot := uint64(0)
		if x, y, ok := box.PointerPosition(); ok && box.Hovered() {
			if r := int(y / rowH); r >= 0 && r < len(rows) && !rows[r].label {
				in := insts[rows[r].inst]
				hot = in.Addr
				if x < gutter {
					for _, a := range arrows {
						ax := gutter - t.Space(3) - float32(a.lane)*lane
						if x > ax-lane/2 && x < ax+lane/2 && r >= min(a.from, a.to) && r <= max(a.from, a.to) {
							hot = a.src
						}
					}
				}
				if box.Clicked() {
					s.chosen = in.Addr
				}
				if box.DoubleClicked() && in.Target != 0 {
					if _, ok := at[in.Target]; ok {
						s.back = append(s.back, in.Addr)
						go2(in.Target)
					}
				}
			}
		}
		if hot != s.hot {
			s.hot = hot
			c.Invalidate()
		}
		chosen, pc := s.chosen, s.PC
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			vis := p.Visible()
			first := max(0, int((vis.Y-r.Y)/rowH))
			last := min(len(rows), int((vis.Y+vis.H-r.Y)/rowH)+1)
			for i := first; i < last; i++ {
				y := r.Y + float32(i)*rowH
				rw := rows[i]
				in := &insts[rw.inst]
				if rw.label {
					p.RichText(r.X+cols[1], y+(rowH-size)/2-1, 0,
						ui.Span{Text: in.Label + ":", Font: "monospace", Size: size, Weight: 700, Color: t.Text})
					continue
				}
				band := ui.Rect{X: r.X, Y: y, W: r.W, H: rowH}
				switch {
				case in.Addr == chosen:
					p.Fill(band, t.Selection, 0)
				case in.Addr == hot:
					p.Fill(band, t.Text.Alpha(0.05), 0)
				}
				if in.Addr == pc {
					p.Fill(band, t.Warning.Alpha(0.18), 0)
					tip := ui.Path{}
					mid := y + rowH/2
					tip.MoveTo(r.X+gutter-t.Space(2.5), mid-4).LineTo(r.X+gutter-t.Space(0.5), mid).LineTo(r.X+gutter-t.Space(2.5), mid+4).Close()
					p.FillPath(&tip, t.Warning)
				}
				ty := y + (rowH-size)/2 - 1
				mono := func(x float32, s string, col ui.Color, weight int) {
					p.RichText(r.X+x, ty, 0, ui.Span{Text: s, Font: "monospace", Size: size, Color: col, Weight: weight})
				}
				mono(cols[1]-float32(digits+1)*cw, fmt.Sprintf("%0*x", digits, in.Addr), t.TextMuted, 0)
				mono(cols[1], bytes(in.Bytes), t.TextMuted.Alpha(0.8), 0)
				mono(cols[2], in.Op, opColor(t, in.Op), 600)
				mono(cols[3], in.Args, t.Text, 0)
				if in.Comment != "" {
					mono(cols[4], "; "+in.Comment, t.TextMuted, 0)
				}
			}
			for _, a := range arrows {
				if max(a.from, a.to) < first || min(a.from, a.to) > last {
					continue
				}
				col, width := t.TextMuted.Alpha(0.6), float32(1)
				if a.src == hot || a.dst == hot || a.src == chosen {
					col, width = t.Accent, 2
				}
				x := r.X + gutter - t.Space(3) - float32(a.lane)*lane
				y0 := r.Y + float32(a.from)*rowH + rowH/2
				y1 := r.Y + float32(a.to)*rowH + rowH/2
				end := r.X + gutter - t.Space(1)
				p.StrokePolyline([][2]float32{{end - t.Space(1), y0}, {x, y0}, {x, y1}, {end, y1}}, width, col)
				head := ui.Path{}
				head.MoveTo(end+1, y1).LineTo(end-4, y1-3.5).LineTo(end-4, y1+3.5).Close()
				p.FillPath(&head, col)
			}
		})
	})
	return view
}

// lay returns the arrows of the jumps among insts listed in rows, at in
// their rows by address, each in the innermost lane free over its rows,
// the shorter jumps inside.
func lay(insts []Inst, rows []row, at map[uint64]int) []arrow {
	var arrows []arrow
	for r, rw := range rows {
		in := insts[rw.inst]
		if rw.label || in.Target == 0 || !isJump(in.Op) {
			continue
		}
		if to, ok := at[in.Target]; ok && to != r {
			arrows = append(arrows, arrow{from: r, to: to, src: in.Addr, dst: in.Target})
		}
	}
	span := func(a arrow) int { return max(a.from, a.to) - min(a.from, a.to) }
	slices.SortStableFunc(arrows, func(a, b arrow) int { return span(a) - span(b) })
	var used [][2]int
	for i := range arrows {
		a := &arrows[i]
		lo, hi := min(a.from, a.to), max(a.from, a.to)
		for a.lane = 0; ; a.lane++ {
			free := true
			for j := range i {
				b := arrows[j]
				if b.lane == a.lane && lo <= used[j][1] && used[j][0] <= hi {
					free = false
					break
				}
			}
			if free {
				break
			}
		}
		used = append(used, [2]int{lo, hi})
	}
	return arrows
}

// bytes writes b in hex, a space between bytes, eight at most.
func bytes(b []byte) string {
	var sb strings.Builder
	for i, v := range b {
		if i == 8 {
			sb.WriteString("…")
			break
		}
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%02x", v)
	}
	return sb.String()
}

// isJump reports whether op jumps within a function, which draws an
// arrow, as a call does not.
func isJump(op string) bool {
	op = strings.ToLower(op)
	for _, p := range []string{"j", "b.", "cbz", "cbnz", "tbz", "tbnz", "loop"} {
		if strings.HasPrefix(op, p) {
			return true
		}
	}
	return op == "b"
}

// opColor returns the color of a mnemonic: jumps in the accent, calls and
// returns in their own colors, and the rest as text.
func opColor(t *ui.Theme, op string) ui.Color {
	o := strings.ToLower(op)
	switch {
	case strings.HasPrefix(o, "ret"):
		return t.Danger
	case strings.HasPrefix(o, "call"), o == "bl", o == "blr":
		return ui.Oklch(0.6, 0.18, 300)
	case isJump(o):
		return t.Accent
	case o == "nop", strings.HasPrefix(o, "int3"):
		return t.TextMuted
	}
	return t.Text
}

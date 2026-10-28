// Package debugger is the panels of a debugger, as the Run and Debug view
// of VS Code: a toolbar that continues, pauses and steps, the call stack
// of each thread, the variables of the frame chosen as a tree whose
// children load as it opens, and the breakpoints.
//
//	switch debugger.Toolbar(c, app.paused) {
//	case debugger.StepOver:
//		app.session.Next()
//	}
//	debugger.Stack(c, &app.debug, app.threads)
//	ev := debugger.Variables(c, &app.debug, app.scopes)
//	if ev.Path != nil {
//		app.session.Set(ev.Path, ev.Value)
//	}
//	debugger.Breakpoints(c, &app.breakpoints)
//
// The toolbar takes the keys of VS Code: F5 continues, F6 pauses, F10
// steps over, F11 into and Shift+F11 out, Cmd+Shift+F5 restarts and
// Shift+F5 stops. A click on a frame chooses it. Variables show their
// values in the colors of their kinds, those that changed lit, and their
// types on hover; a double click or Enter on a value without children
// edits it in place, Enter keeping it and Escape giving it up. A click on
// a breakpoint reports it Opened, its box turns it on and off, and the ×
// over it removes it.
package debugger

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Command is a button of the toolbar.
type Command uint8

const (
	None Command = iota
	Continue
	Pause
	StepOver
	StepInto
	StepOut
	Restart
	Stop
)

// Toolbar creates the toolbar of a session, paused or running, and
// returns the command clicked, or whose key was pressed, this frame.
// Continue shows while paused, Pause while running, and the steps only
// while paused.
func Toolbar(c *ui.Context, paused bool) Command {
	t := c.Theme()
	cmd := None
	button := func(k Command, label string, mods ui.Modifiers, key ui.Key, ink ui.Color, enabled bool) {
		b := ui.ButtonBase(c).Size(t.Space(7), t.Space(7)).Radius(t.Space(1.5)).Label(label).Tooltip(label).Disabled(!enabled)
		if b.Hovered() && enabled {
			b.Background(t.Text.Alpha(0.08))
		}
		if !enabled {
			ink = t.TextMuted.Alpha(0.5)
		}
		b.Draw(func(p *ui.Painter, r ui.Rect) { glyph(p, r, k, ink) })
		if enabled && (b.Clicked() || c.Shortcut(mods, key)) {
			cmd = k
		}
	}
	ui.Row(c).Gap(t.Space(0.5)).Padding(t.Space(0.5)).Radius(t.Space(2)).Background(t.Surface).Border(1, t.Border).Children(func() {
		if paused {
			button(Continue, "Continue", 0, ui.KeyF5, t.Accent, true)
		} else {
			button(Pause, "Pause", 0, ui.KeyF6, t.Accent, true)
		}
		button(StepOver, "Step Over", 0, ui.KeyF10, t.Accent, paused)
		button(StepInto, "Step Into", 0, ui.KeyF11, t.Accent, paused)
		button(StepOut, "Step Out", ui.Shift, ui.KeyF11, t.Accent, paused)
		button(Restart, "Restart", ui.Cmd|ui.Shift, ui.KeyF5, t.Success, true)
		button(Stop, "Stop", ui.Shift, ui.KeyF5, t.Danger, true)
	})
	return cmd
}

// glyph draws the icon of k in r.
func glyph(p *ui.Painter, r ui.Rect, k Command, ink ui.Color) {
	x, y := r.X+r.W/2-8, r.Y+r.H/2-8
	var pa ui.Path
	dot := func(cx, cy float32) {
		var d ui.Path
		d.Circle(x+cx, y+cy, 1.8)
		p.FillPath(&d, ink)
	}
	arrow := func(x0, y0, x1, y1 float32) {
		pa.MoveTo(x+x0, y+y0).LineTo(x+x1, y+y1)
		dx, dy := float32(3), float32(3)
		if y1 > y0 {
			dy = -dy
		}
		pa.MoveTo(x+x1-dx, y+y1+dy).LineTo(x+x1, y+y1).LineTo(x+x1+dx, y+y1+dy)
	}
	switch k {
	case Continue:
		p.Fill(ui.Rect{X: x + 3, Y: y + 3, W: 1.6, H: 10}, ink, 0.5)
		pa.MoveTo(x+6.5, y+3).LineTo(x+13.5, y+8).LineTo(x+6.5, y+13).Close()
		p.FillPath(&pa, ink)
		return
	case Pause:
		p.Fill(ui.Rect{X: x + 4, Y: y + 3, W: 2.4, H: 10}, ink, 0.6)
		p.Fill(ui.Rect{X: x + 9.6, Y: y + 3, W: 2.4, H: 10}, ink, 0.6)
		return
	case StepOver:
		pa.MoveTo(x+2.5, y+9).CubeTo(x+3, y+2.5, x+12.5, y+2.5, x+13, y+8.5)
		pa.MoveTo(x+10, y+6.5).LineTo(x+13, y+9).LineTo(x+15, y+6)
		dot(8, 13)
	case StepInto:
		arrow(8, 1.5, 8, 9.5)
		dot(8, 13.5)
	case StepOut:
		arrow(8, 10, 8, 2)
		dot(8, 13.5)
	case Restart:
		pa.Circle(x+8, y+8, 5.5).Trim(0.12, 0.92)
		pa.MoveTo(x+13.8, y+2.8).LineTo(x+13.8, y+6.6).LineTo(x+10, y+6.6)
	case Stop:
		p.Fill(ui.Rect{X: x + 3.5, Y: y + 3.5, W: 9, H: 9}, ink, 1.5)
		return
	}
	p.StrokePath(&pa, 1.6, ink)
}

// Frame is a frame of a call stack: its function, and where it is. A
// frame without a file, as the runtime's, shows dimmed.
type Frame struct {
	Func string
	File string
	Line int
}

// Thread is a thread of the program, or a goroutine: its name, why it is
// paused, as "breakpoint", or "" while it runs, and its frames, the
// innermost first.
type Thread struct {
	Name   string
	Reason string
	Frames []Frame
}

// State is what the panels keep: the frame chosen, the variables open and
// the value being edited. Keep it in the app's state, one for each
// session.
type State struct {
	thread, frame int
	chosen        bool
	tree          ui.OutlineState[string]
	row           int
	seen          map[string]bool
	editing       string
	draft         string
	fresh         bool
	focused       bool
	setup         bool
}

// Frame returns the frame chosen: its thread and its place in the stack.
func (s *State) Frame() (thread, frame int) { return s.thread, s.frame }

func (s *State) init() {
	if !s.setup {
		s.setup, s.row, s.seen = true, -1, map[string]bool{}
		s.tree.List.Selected = &s.row
	}
}

// Stack creates the call stack of threads: each thread's name and why it
// paused, and its frames, the function and where it is.
func Stack(c *ui.Context, s *State, threads []Thread) ui.Element {
	t := c.Theme()
	s.init()
	if !s.chosen || s.thread >= len(threads) || s.frame >= len(threads[s.thread].Frames) {
		s.thread, s.frame = 0, 0
		for i, th := range threads {
			if th.Reason != "" && len(th.Frames) > 0 {
				s.thread, s.chosen = i, true
				break
			}
		}
	}
	return ui.Column(c).Gap(t.Space(0.5)).Children(func() {
		for ti, th := range threads {
			ui.Row(c.Key(ti)).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(2)).Children(func() {
				ui.Text(c, th.Name).Bold().SingleLine().Shrink(1).Grow(1)
				state := "Running"
				if th.Reason != "" {
					state = "Paused on " + th.Reason
				}
				ui.Text(c, state).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
			})
			if th.Reason == "" {
				continue
			}
			for fi, f := range th.Frames {
				chosen := ti == s.thread && fi == s.frame
				row := ui.ButtonBase(c.Key(fmt.Sprint(ti, ".", fi))).Gap(t.Space(2)).Padding(t.Space(1), t.Space(2), t.Space(1), t.Space(5)).
					Radius(t.Space(1)).Justify(ui.Start).AlignItems(ui.Center).Label(f.Func)
				ink, muted := t.Text, t.TextMuted
				switch {
				case chosen:
					row.Background(t.Accent)
					ink, muted = t.AccentText, t.AccentText
				case row.Hovered():
					row.Background(t.Text.Alpha(0.06))
				}
				if f.File == "" {
					ink = muted
				}
				if row.Clicked() {
					s.thread, s.frame, s.chosen = ti, fi, true
				}
				row.Children(func() {
					ui.Text(c, f.Func).TextColor(ink).SingleLine().Shrink(1).Grow(1)
					if f.File != "" {
						ui.Text(c, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)).FontSize(t.FontSize - 1).TextColor(muted).SingleLine()
					}
				})
			}
		}
	})
}

// Scope is a group of variables, as Locals and Globals.
type Scope struct {
	Name string
	Vars []Var
}

// Var is a variable: its name, type and value as the debugger shows them,
// whether it changed since the last step, and its children, as fields and
// elements, which Kids returns when it shows open, or nil for none. Kids
// is called each frame its variable shows open: an app fetching them
// from a debugger keeps what it fetched.
type Var struct {
	Name, Type, Value string
	Changed           bool
	Kids              func() []Var
}

// Events is a panel, and what the user did with it this frame: edited
// the value of the variable at Path, the names from its scope down, to
// Value; or Opened a breakpoint, its index, or -1.
type Events struct {
	ui.Element
	Path   []string
	Value  string
	Opened int
}

// sep joins the names of a variable's key.
const sep = "\x1f"

// Variables creates the tree of scopes, the first open as it comes. Give
// it a height, or Grow.
func Variables(c *ui.Context, s *State, scopes []Scope) Events {
	t := c.Theme()
	s.init()
	ev := Events{Opened: -1}
	if len(scopes) > 0 && !s.seen[scopes[0].Name] {
		s.tree.Open.Add(scopes[0].Name)
	}
	vars := map[string]Var{}
	roots := make([]string, len(scopes))
	for i, sc := range scopes {
		roots[i] = sc.Name
		s.seen[sc.Name] = true
	}
	list := func(parent string, vs []Var) []string {
		out := make([]string, len(vs))
		for i, v := range vs {
			k := parent + sep + v.Name
			vars[k] = v
			out[i] = k
		}
		return out
	}
	children := func(k string) []string {
		if !strings.Contains(k, sep) {
			for _, sc := range scopes {
				if sc.Name == k {
					return list(k, sc.Vars)
				}
			}
			return nil
		}
		if v := vars[k]; v.Kids != nil {
			return list(k, v.Kids())
		}
		return nil
	}
	colors := lightColors
	if t.Dark {
		colors = darkColors
	}
	done := func() { s.editing = "" }
	view := ui.Outline(c, &s.tree, roots, children, func(k string) {
		chosen := s.row >= 0 && s.row < s.tree.Rows() && s.tree.Item(s.row) == k
		v, ok := vars[k]
		if !ok {
			ink := t.TextMuted
			if chosen {
				ink = t.AccentText
			}
			ui.Text(c, k).Bold().FontSize(t.FontSize - 1).TextColor(ink)
			return
		}
		cs := colors
		// The colors of kinds are unreadable on the accent of the row chosen.
		if chosen {
			a := t.AccentText
			cs = syntax{name: a, str: a, number: a, keyword: a, other: a}
		}
		row := ui.Row(c).Gap(0).AlignItems(ui.Center).Grow(1)
		if v.Type != "" {
			row.Tooltip(v.Type)
		}
		row.Children(func() {
			if k == s.editing {
				ui.Text(c, v.Name+": ").TextColor(cs.name).SingleLine()
				in := ui.TextInputBase(c, &s.draft).Grow(1).Padding(0, t.Space(1)).Background(t.Background).TextColor(t.Text).
					Border(1, t.Accent).Radius(2).Label("Value of " + v.Name)
				if s.fresh {
					in.Focus().SetTextSelection(0, len([]rune(s.draft)))
					s.fresh = false
				}
				switch {
				case in.Submitted():
					ev.Path, ev.Value = strings.Split(k, sep), s.draft
					done()
				case in.Shortcut(0, ui.KeyEscape), s.focused && !in.Focused():
					done()
				}
				s.focused = in.Focused()
				return
			}
			value := ui.Span{Text: v.Value, Color: cs.of(v.Value)}
			if v.Changed && !chosen {
				value.Background, value.Radius = t.Warning.Alpha(0.3), 2
			}
			ui.RichText(c, ui.Span{Text: v.Name, Color: cs.name}, ui.Span{Text: ": ", Color: cs.other.Alpha(0.6)}, value).
				SingleLine().Shrink(1)
		})
	}).Font("monospace").FontSize(t.FontSize - 1)
	ev.Element = view
	if view.Submitted() && s.row >= 0 && s.row < s.tree.Rows() {
		k := s.tree.Item(s.row)
		if v, ok := vars[k]; ok && v.Kids == nil {
			s.editing, s.draft, s.fresh, s.focused = k, v.Value, true, false
		}
	}
	return ev
}

// syntax is the colors of names and values, as VS Code colors them.
type syntax struct {
	name, str, number, keyword, other ui.Color
}

var (
	lightColors = syntax{name: ui.Hex("#001080"), str: ui.Hex("#a31515"), number: ui.Hex("#098658"), keyword: ui.Hex("#0000ff"), other: ui.Hex("#1f1f1f")}
	darkColors  = syntax{name: ui.Hex("#9cdcfe"), str: ui.Hex("#ce9178"), number: ui.Hex("#b5cea8"), keyword: ui.Hex("#569cd6"), other: ui.Hex("#d4d4d4")}
)

// of returns the color of a value by its kind: quoted, a number, a
// keyword, or another.
func (cs syntax) of(v string) ui.Color {
	switch {
	case strings.HasPrefix(v, `"`), strings.HasPrefix(v, "'"), strings.HasPrefix(v, "`"):
		return cs.str
	case v == "true", v == "false", v == "nil", v == "null", v == "undefined", v == "None":
		return cs.keyword
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return cs.number
	}
	return cs.other
}

// Breakpoint is a line the program stops at while On, when Cond, if any,
// holds.
type Breakpoint struct {
	File string
	Line int
	On   bool
	Cond string
}

// Breakpoints creates the list of *bps: each one's box, its file and
// folder, its line, and its condition.
func Breakpoints(c *ui.Context, bps *[]Breakpoint) Events {
	t := c.Theme()
	ev := Events{Opened: -1}
	remove := -1
	ev.Element = ui.Column(c).Gap(t.Space(0.5)).Children(func() {
		for i := range *bps {
			b := &(*bps)[i]
			row := ui.Row(c.Key(fmt.Sprint(b.File, ":", b.Line))).Gap(t.Space(2)).AlignItems(ui.Center).
				Padding(t.Space(1), t.Space(2)).Radius(t.Space(1)).Cursor(ui.CursorPointer)
			if row.Hovered() {
				row.Background(t.Text.Alpha(0.06))
			}
			if row.Clicked() {
				ev.Opened = i
			}
			hovered := row.Hovered()
			row.Children(func() {
				ui.Checkbox(c, &b.On, "").Label(fmt.Sprintf("%s:%d", b.File, b.Line))
				on := b.On
				ui.Box(c).Size(10, 10).Draw(func(p *ui.Painter, r ui.Rect) {
					var d ui.Path
					d.Circle(r.X+5, r.Y+5, 4.5)
					if on {
						p.FillPath(&d, t.Danger)
						return
					}
					p.StrokePath(&d, 1.2, t.TextMuted)
				})
				ink := t.Text
				if !b.On {
					ink = t.TextMuted
				}
				ui.Column(c).Grow(1).Shrink(1).Children(func() {
					ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Text(c, path.Base(b.File)).TextColor(ink).SingleLine().Shrink(0)
						if dir := path.Dir(b.File); dir != "." {
							ui.Text(c, dir).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine().Shrink(1)
						}
					})
					if b.Cond != "" {
						ui.Text(c, b.Cond).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
					}
				})
				if hovered {
					x := ui.ButtonBase(c).Size(t.Space(5), t.Space(5)).Radius(t.Space(1)).Label("Remove Breakpoint").Tooltip("Remove")
					if x.Hovered() {
						x.Background(t.Text.Alpha(0.08))
					}
					x.Draw(func(p *ui.Painter, r ui.Rect) {
						cx, cy, d := r.X+r.W/2, r.Y+r.H/2, float32(3.5)
						p.StrokePolyline([][2]float32{{cx - d, cy - d}, {cx + d, cy + d}}, 1.4, t.TextMuted)
						p.StrokePolyline([][2]float32{{cx - d, cy + d}, {cx + d, cy - d}}, 1.4, t.TextMuted)
					})
					if x.Clicked() {
						remove = i
					}
					return
				}
				ui.Text(c, strconv.Itoa(b.Line)).FontSize(t.FontSize-1).TextColor(t.TextMuted).Padding(0, t.Space(1.5)).
					Radius(999).Background(t.Text.Alpha(0.06))
			})
		}
	})
	if remove >= 0 {
		*bps = append((*bps)[:remove], (*bps)[remove+1:]...)
		ev.Opened = -1
	}
	return ev
}

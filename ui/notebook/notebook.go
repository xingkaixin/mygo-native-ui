// Package notebook shows a notebook of cells, as Jupyter and the
// notebooks of VS Code do: code cells with their execution count, an
// editor and what they printed, and Markdown cells shown rendered until
// a double click opens them to edit.
//
//	ev := notebook.View(c, &app.nb, &app.cells)
//	if ev.Run >= 0 {
//		app.run(ev.Run) // sets Running, then Count and Outputs
//	}
//
// The chosen cell has a bar at its left and its tools at its right: move
// up, move down and delete; + Code and + Text after the last cell add a
// cell after the chosen one. Shift+Enter runs a code cell and moves to the next, adding one
// after the last; Cmd+Enter runs it and stays. On a Markdown cell being
// edited either key shows it rendered. Clicking a code cell's count, a ▶
// on hover, runs it too.
//
// The app runs the cells: Run is the cell to run, -1 none, in the frame
// it was asked for. It sets the cell's Running while it runs, which shows
// [*], and its Count and Outputs once it ran.
package notebook

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

// Kind is what a cell holds.
type Kind uint8

const (
	Code Kind = iota
	Markdown
)

// Output is what running a code cell printed, or the error it raised.
type Output struct {
	Text  string
	Error bool
}

// Cell is a cell of a notebook. Make one with NewCell.
type Cell struct {
	Kind Kind
	// Source is the cell's text and its editor's state.
	Source *editor.State
	// Count is the number of the run that ran it last, 0 for none, and
	// Running whether it runs now.
	Count   int
	Running bool
	Outputs []Output

	id      int
	editing bool
}

var ids int

// NewCell returns a cell of kind holding source.
func NewCell(kind Kind, source string) Cell {
	ids++
	st := &editor.State{
		Buffer:          editor.NewBuffer(source),
		FitContent:      true,
		LineNumbers:     editor.LineNumbersOff,
		NoFolding:       true,
		NoActiveLine:    true,
		NoScrollPastEnd: true,
		SoftWrap:        true,
		Bindings: []editor.Binding{
			{Mods: ui.Shift, Key: ui.KeyEnter, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyEnter, Command: editor.NoCommand},
		},
	}
	return Cell{Kind: kind, Source: st, id: ids, editing: source == ""}
}

// Text returns the cell's source.
func (cl *Cell) Text() string { return cl.Source.Buffer.String() }

// State is what a notebook keeps: the language of its code and the cell
// chosen. Keep it in the app's state, one for each notebook.
type State struct {
	// Language is a file name whose language colors code cells, as
	// "cell.py".
	Language string
	selected int
	focus    int
	md       markdown.Options
}

// Events is a notebook, and the cell it asks the app to run.
type Events struct {
	ui.Element
	Run int
}

// action is what a cell asked for in a frame, done once all are built.
type action struct {
	kind     int
	at       int
	cellKind Kind
}

const (
	none = iota
	run
	runNext
	insert
	remove
	moveUp
	moveDown
)

// View creates a view of *cells, adding, moving and deleting them as the
// user asks.
func View(c *ui.Context, s *State, cells *[]Cell) Events {
	t := c.Theme()
	list := ui.Column(c).Gap(t.Space(1))
	ev := Events{Element: list, Run: -1}
	lang := editor.LanguageFor(s.Language)
	s.selected = max(0, min(len(*cells)-1, s.selected))
	var act action
	list.Children(func() {
		for i := range *cells {
			cl := &(*cells)[i]
			if cl.Source.Buffer.Language() == nil && cl.Kind == Code && lang != nil {
				cl.Source.Buffer.SetLanguage(lang)
			}
			if a := cell(c, s, cl, i, len(*cells)); a.kind != none {
				act = a
			}
		}
		adders(c, &act, min(s.selected, len(*cells)-1))
	})

	cs := *cells
	switch act.kind {
	case run, runNext:
		cl := &cs[act.at]
		if cl.Kind == Code {
			ev.Run = act.at
		} else {
			cl.editing = false
		}
		if act.kind == runNext {
			if act.at == len(cs)-1 {
				cs = append(cs, NewCell(Code, ""))
			}
			s.selected = act.at + 1
			s.focus = cs[s.selected].id
		}
	case insert:
		cl := NewCell(act.cellKind, "")
		cs = append(cs[:act.at+1], append([]Cell{cl}, cs[act.at+1:]...)...)
		s.selected, s.focus = act.at+1, cl.id
	case remove:
		cs = append(cs[:act.at], cs[act.at+1:]...)
		s.selected = max(0, min(len(cs)-1, act.at))
	case moveUp, moveDown:
		to := act.at - 1
		if act.kind == moveDown {
			to = act.at + 1
		}
		if to >= 0 && to < len(cs) {
			cs[act.at], cs[to] = cs[to], cs[act.at]
			s.selected = to
		}
	}
	*cells = cs
	return ev
}

// cell builds cell i, and returns what it asked for.
func cell(c *ui.Context, s *State, cl *Cell, i, n int) action {
	t := c.Theme()
	var act action
	chosen := i == s.selected
	box := ui.Row(c.Key(cl.id)).AlignItems(ui.Start).Gap(t.Space(2)).Padding(t.Space(2), t.Space(2), t.Space(2), t.Space(1)+4).Radius(t.Radius)
	if chosen {
		box.Background(t.Surface).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 4, H: r.H}, t.Accent, 2)
		})
	}
	if box.Pressed() || box.FocusWithin() {
		s.selected = i
	}
	switch {
	case box.Shortcut(ui.Shift, ui.KeyEnter):
		act = action{kind: runNext, at: i}
	case box.Shortcut(ui.Cmd, ui.KeyEnter):
		act = action{kind: run, at: i}
	}

	box.Children(func() {
		gutter := ui.Box(c).Width(t.Space(12)).PaddingY(t.Space(1.5)).Shrink(0)
		if cl.Kind == Code {
			gutter.Children(func() {
				if count(c, cl, chosen || box.Hovered()) {
					act = action{kind: run, at: i}
				}
			})
		}
		ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(2)).Children(func() {
			body(c, s, cl)
			if cl.Kind == Code && len(cl.Outputs) > 0 {
				outputs(c, cl.Outputs)
			}
		})
		if a := tools(c, i, n, chosen); a.kind != none {
			act = a
		}
	})
	return act
}

// count shows a code cell's count, as [3]:, or a run button while shown
// is set, and reports whether it was clicked.
func count(c *ui.Context, cl *Cell, shown bool) bool {
	t := c.Theme()
	label := "[ ]:"
	switch {
	case cl.Running:
		label = "[*]:"
	case cl.Count > 0:
		label = "[" + strconv.Itoa(cl.Count) + "]:"
	}
	b := ui.ButtonBase(c).Label("Run cell").Tooltip("Run cell (⇧↩)").Padding(t.Space(1)).Radius(t.Radius)
	if !shown || cl.Running {
		b.Children(func() {
			ui.Text(c, label).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		return b.Clicked()
	}
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Size(t.Space(7), t.Space(7)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.H/5
		tri := new(ui.Path).MoveTo(cx-k*0.8, cy-k).LineTo(cx+k, cy).LineTo(cx-k*0.8, cy+k).Close()
		p.FillPath(tri, t.Accent)
	})
	return b.Clicked()
}

// body builds a cell's editor, or a Markdown cell rendered while it is
// not edited.
func body(c *ui.Context, s *State, cl *Cell) {
	t := c.Theme()
	if cl.Kind == Markdown && !cl.editing {
		md := ui.Column(c).PaddingY(t.Space(1))
		// The text takes clicks to select itself, so the double click that
		// edits the cell is caught as it comes, before them.
		md.HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind != ui.InputPointerDown || ev.Clicks != 2 {
				return false
			}
			cl.editing, s.focus = true, cl.id
			return true
		})
		md.Children(func() {
			markdown.View(c, cl.Text(), &s.md)
		})
		return
	}
	frame := ui.Column(c).Border(1, t.Border).Radius(t.Radius).Background(t.Background).Padding(t.Space(1.5), 0)
	frame.Children(func() {
		ed := editor.View(c, cl.Source).FillWidth()
		if s.focus == cl.id {
			ed.Focus()
			s.focus = 0
		}
	})
}

// outputs shows what a cell printed, errors on red.
func outputs(c *ui.Context, outs []Output) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		for _, o := range outs {
			text := ui.Text(c, o.Text).Font("monospace").FontSize(t.FontSize-1).Selectable().Padding(t.Space(1), t.Space(2))
			if o.Error {
				text.TextColor(t.Danger).Background(t.Danger.Alpha(0.08)).Radius(t.Radius)
			}
		}
	})
}

// tools builds cell i's tools, shown while it is chosen and keeping
// their room while not, and returns what they asked for.
func tools(c *ui.Context, i, n int, chosen bool) action {
	t := c.Theme()
	var act action
	col := ui.Row(c).Gap(t.Space(1)).Shrink(0)
	if !chosen {
		col.Invisible()
	}
	col.Children(func() {
		for _, b := range []struct {
			label, text string
			kind        int
			ok          bool
		}{
			{"Move cell up", "↑", moveUp, i > 0},
			{"Move cell down", "↓", moveDown, i < n-1},
			{"Delete cell", "✕", remove, true},
		} {
			btn := ui.Button(c.Key(b.label), b.text).Label(b.label).Tooltip(b.label).Disabled(!b.ok).KeepFocus()
			if btn.Clicked() {
				act = action{kind: b.kind, at: i}
			}
		}
	})
	return act
}

// adders builds + Code and + Text, which add a cell after cell i.
func adders(c *ui.Context, act *action, i int) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), 0, 0, t.Space(16)).Children(func() {
		if ui.Button(c, "+ Code").KeepFocus().Clicked() {
			*act = action{kind: insert, at: i, cellKind: Code}
		}
		if ui.Button(c, "+ Text").KeepFocus().Clicked() {
			*act = action{kind: insert, at: i, cellKind: Markdown}
		}
	})
}

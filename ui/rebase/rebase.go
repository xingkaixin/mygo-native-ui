// Package rebase edits the todo of an interactive rebase, as git clients
// show `git rebase -i`: a row a commit, oldest first, on a line joining
// them, with its action, short hash and subject. The action is chosen from
// a menu on its chip, or with the letters of git's own todo while the row
// is chosen: P pick, R reword, E edit, S squash, F fixup and D drop. A
// commit squashed or fixed up is set in under the one it melds into, a
// dropped one is struck through, and a reworded one opens its message to
// edit in place, which Enter keeps and Escape gives up. Rows are dragged
// to another place, or moved with Alt+Up and Alt+Down. A line above tells
// how many commits the rebase leaves, and warns of a squash with nothing
// before it to meld into. Give it a height, or Grow.
//
//	rebase.View(c, &app.rebase, &app.steps)
//	if ui.PrimaryButton(c, "Start rebase").Clicked() {
//		app.run(rebase.Todo(app.steps))
//	}
package rebase

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Action is what the rebase does with a commit.
type Action uint8

const (
	Pick Action = iota
	Reword
	Edit
	Squash
	Fixup
	Drop
)

var names = [...]string{"pick", "reword", "edit", "squash", "fixup", "drop"}

func (a Action) String() string { return names[a] }

// Step is a commit of the todo: what to do with it, its hash and subject,
// and with Reword, its new message.
type Step struct {
	Action        Action
	Hash, Subject string
	Message       string
}

// melds reports whether the step melds into the one before it.
func (s Step) melds() bool { return s.Action == Squash || s.Action == Fixup }

// Todo writes steps as git's rebase todo, one "pick 3f2c183 Subject" a
// line. A commit reworded to a new message is picked, then amended by an
// exec line, so that the rebase runs without stopping for an editor.
func Todo(steps []Step) string {
	var b strings.Builder
	for _, s := range steps {
		if s.Action == Reword && s.Message != "" && s.Message != s.Subject {
			quoted := "'" + strings.ReplaceAll(s.Message, "'", `'\''`) + "'"
			b.WriteString("pick " + s.Hash + " " + s.Subject + "\n")
			b.WriteString("exec git commit --amend --only -m " + quoted + "\n")
			continue
		}
		b.WriteString(s.Action.String() + " " + s.Hash + " " + s.Subject + "\n")
	}
	return b.String()
}

// State is what an editor keeps: the list, the row chosen, and the
// message being reworded. Keep it in the app's state, one for each
// editor.
type State struct {
	list    ui.ListState
	chosen  int
	editing string
	draft   string
}

var keys = []struct {
	key    ui.Key
	action Action
}{
	{ui.KeyP, Pick}, {ui.KeyR, Reword}, {ui.KeyE, Edit},
	{ui.KeyS, Squash}, {ui.KeyF, Fixup}, {ui.KeyD, Drop},
}

// colors returns the color of action's chip.
func colors(t *ui.Theme, a Action) ui.Color {
	switch a {
	case Reword:
		return ui.Hex("#2f80ed")
	case Edit:
		return ui.Hex("#e8890c")
	case Squash, Fixup:
		return ui.Hex("#8e5bd6")
	case Drop:
		return t.Danger
	}
	return t.TextMuted
}

// View creates the editor of *steps.
func View(c *ui.Context, s *State, steps *[]Step) ui.Element {
	t := c.Theme()
	st := *steps
	changed := false
	set := func(i int, a Action) {
		if i < 0 || i >= len(st) || st[i].Action == a {
			return
		}
		st[i].Action, changed = a, true
		if a == Reword {
			s.editing, s.draft = st[i].Hash, st[i].Message
			if s.draft == "" {
				s.draft = st[i].Subject
			}
		}
	}
	move := func(rows []int, to int) {
		var moved []Step
		for _, r := range rows {
			moved = append(moved, st[r])
		}
		kept := slices.DeleteFunc(slices.Clone(st), func(x Step) bool {
			return slices.ContainsFunc(moved, func(m Step) bool { return m.Hash == x.Hash })
		})
		before := 0
		for _, r := range rows {
			if r < to {
				before++
			}
		}
		at := max(0, min(len(kept), to-before))
		*steps = slices.Insert(kept, at, moved...)
		st = *steps
		s.chosen, changed = at, true
	}
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		left, dropped, melded := 0, 0, 0
		for _, x := range st {
			switch {
			case x.Action == Drop:
				dropped++
			case x.melds():
				melded++
			default:
				left++
			}
		}
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, plural(len(st), "commit")+" become "+strconv.Itoa(left)).Bold()
			if melded > 0 {
				ui.Text(c, strconv.Itoa(melded)+" melded").TextColor(colors(t, Squash))
			}
			if dropped > 0 {
				ui.Text(c, strconv.Itoa(dropped)+" dropped").TextColor(t.Danger)
			}
			if len(st) > 0 && st[0].melds() {
				ui.Text(c, "The first commit has nothing before it to meld into").TextColor(t.Danger)
			}
		})
		s.list.Selected = &s.chosen
		s.list.Key = func(i int) any { return st[i].Hash }
		s.list.Reorder = move
		ui.List(c, &s.list, len(st), func(i int) {
			if a, ok := s.row(c, st, i); ok {
				set(i, a)
			}
		}).Grow(1).Radius(t.Space(2)).Border(1, t.Border).Focusable().Label("Commits")
		if s.editing == "" {
			for _, k := range keys {
				if s.list.Shortcut(c, 0, k.key) {
					set(s.chosen, k.action)
				}
			}
			switch {
			case s.list.Shortcut(c, ui.Alt, ui.KeyUp) && s.chosen > 0:
				move([]int{s.chosen}, s.chosen-1)
			case s.list.Shortcut(c, ui.Alt, ui.KeyDown) && s.chosen < len(st)-1:
				move([]int{s.chosen}, s.chosen+2)
			}
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// row creates the row of step i, and returns the action chosen from its
// chip's menu.
func (s *State) row(c *ui.Context, st []Step, i int) (Action, bool) {
	t := c.Theme()
	x := &st[i]
	chosen := Action(0)
	ok := false
	col := colors(t, x.Action)
	muted := t.TextMuted
	if i == s.chosen {
		col, muted = t.AccentText, t.AccentText
	}
	ui.Row(c).Gap(t.Space(2)).Padding(0, t.Space(3), 0, 0).Height(t.Space(10)).AlignItems(ui.Center).Children(func() {
		first, last := i == 0, i == len(st)-1
		action, nextMelds := x.Action, !last && st[i+1].melds()
		meld := colors(t, Squash)
		if i == s.chosen {
			meld = col
		}
		ui.Box(c).Width(t.Space(8)).FillHeight().Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy := r.X+r.W/2+t.Space(1), r.Y+r.H/2
			line := muted.Alpha(0.4)
			top, bottom := r.Y, r.Y+r.H
			if first {
				top = cy
			}
			if last {
				bottom = cy
			}
			p.Line(cx, top, cx, bottom, 2, line)
			var dot ui.Path
			switch {
			case action == Drop:
				dot.Circle(cx, cy, 4.5)
				p.FillPath(&dot, t.Background)
				p.StrokePath(&dot, 1.5, col)
			case action == Squash || action == Fixup:
				// The commit hangs off the line to the right, into the one
				// above it.
				var hook ui.Path
				hook.MoveTo(cx, cy-r.H*0.7).QuadTo(cx, cy, cx+t.Space(3), cy)
				p.StrokePath(&hook, 2, col)
				dot.Circle(cx+t.Space(3), cy, 3.5)
				p.FillPath(&dot, col)
			default:
				dot.Circle(cx, cy, 5)
				p.FillPath(&dot, t.Background)
				p.StrokePath(&dot, 2.5, col)
				if nextMelds {
					p.Line(cx, cy, cx, bottom, 2, meld)
				}
			}
		})
		indent := float32(0)
		if x.melds() {
			indent = t.Space(4)
		}
		chip := ui.ButtonBase(c).Width(t.Space(19)).Gap(t.Space(1)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(1.5)).
			Background(col.Alpha(0.12)).MarginStart(indent).Label("Action").Value(x.Action.String())
		chip.Children(func() {
			ui.Text(c, x.Action.String()).Font("monospace").FontSize(t.FontSize - 1).TextColor(col).Grow(1)
			ui.Text(c, "▾").FontSize(t.FontSize - 3).TextColor(col)
		})
		chip.Menu(func(m *ui.Menu) {
			for _, k := range keys {
				if m.Item(k.action.String()).Checked(k.action == x.Action).Shortcut(0, k.key).Chosen() {
					chosen, ok = k.action, true
				}
			}
		})
		ui.Text(c, x.Hash).Font("monospace").FontSize(t.FontSize - 1).TextColor(muted).Shrink(0)
		if s.editing == x.Hash {
			in := ui.TextInput(c, &s.draft).Grow(1).AutoFocus().Label("Message")
			switch {
			case in.Submitted():
				x.Message, s.editing = strings.TrimSpace(s.draft), ""
			case in.Shortcut(0, ui.KeyEscape):
				s.editing = ""
			}
			return
		}
		subject := x.Subject
		if x.Action == Reword && x.Message != "" {
			subject = x.Message
		}
		txt := ui.Text(c, subject).SingleLine().Grow(1).Shrink(1)
		if x.Action == Drop {
			txt.Strikethrough()
		}
		if x.Action == Drop || x.melds() {
			txt.TextColor(muted)
		}
		if x.Action == Reword {
			edit := ui.ButtonBase(c).Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(1.5)).Label("Edit message")
			if edit.Hovered() {
				edit.Background(t.Text.Alpha(0.06))
			}
			if edit.Clicked() {
				s.editing, s.draft = x.Hash, subject
			}
			link := t.Accent
			if i == s.chosen {
				link = t.AccentText
			}
			edit.Children(func() { ui.Text(c, "Edit message").FontSize(t.FontSize - 1).TextColor(link) })
		}
	})
	return chosen, ok
}

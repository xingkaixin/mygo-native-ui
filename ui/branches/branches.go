// Package branches keeps a chat as a tree, as ChatGPT and Claude keep
// one: editing a message the user sent, or retrying a reply, adds a
// version beside it instead of replacing it, and each message keeps the
// version after it last shown, so that going back to a version brings
// back the conversation that followed it. Map shows the whole tree small:
// a dot a message, the versions of one side by side, the conversation
// showing lit, and a click on a dot showing it.
//
//	for _, id := range app.tree.Path() {
//		m := app.tree.Message(id)
//		at, n := app.tree.Version(id)
//		if m.user {
//			if text, ok := message.User(c, at, n, m.text).Edited(); ok {
//				app.tree.Add(app.tree.Branch(id, msg{user: true, text: text}), app.reply(text))
//			}
//		} else if message.Reply(c, at, n, m.text, false, body).Retried() {
//			app.tree.Branch(id, app.retry(m))
//		}
//	}
//	ui.Popover(c, button, &app.mapOpen, func() {
//		if id, ok := branches.Map(c, &app.tree, label).Shown(); ok {
//			app.scrollTo(id)
//		}
//	})
//
// The app keeps the tree, adds each message after the last one showing,
// and a version as the user edits or retries. Version returns what a
// switcher such as message's ‹ 2/3 › arrows needs to change the version
// showing.
package branches

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// Tree is a conversation as a tree of messages of the app's type T: the
// messages after each are versions of one another, of which one shows,
// the one last shown. The zero Tree has no messages. Keep it in the app's
// state.
type Tree[T any] struct {
	// root is the parent of the versions of the first message.
	root  node[T]
	nodes []*node[T]
}

// node is a message, the one it follows, -1 for the first, the versions
// of the message after it, and the one of them showing.
type node[T any] struct {
	msg    T
	parent int
	kids   []int
	at     int
}

func (t *Tree[T]) node(id int) *node[T] {
	if id < 0 {
		return &t.root
	}
	return t.nodes[id]
}

// Add adds msg after message parent, -1 for the first message, as the
// last of the versions there, and shows it. It returns the ID of msg,
// which stays its own.
func (t *Tree[T]) Add(parent int, msg T) int {
	id := len(t.nodes)
	t.nodes = append(t.nodes, &node[T]{msg: msg, parent: parent})
	p := t.node(parent)
	p.kids = append(p.kids, id)
	t.Show(id)
	return id
}

// Branch adds msg as a version of message id, after the others, and
// shows it, as for the user editing the message or retrying a reply. It
// returns the ID of msg.
func (t *Tree[T]) Branch(id int, msg T) int { return t.Add(t.nodes[id].parent, msg) }

// Message returns message id, for the app to read and change.
func (t *Tree[T]) Message(id int) *T { return &t.nodes[id].msg }

// Path returns the IDs of the messages showing, from the first: at each,
// the version last shown.
func (t *Tree[T]) Path() []int {
	var path []int
	for n := &t.root; len(n.kids) > 0; {
		id := n.kids[n.at]
		path = append(path, id)
		n = t.nodes[id]
	}
	return path
}

// Version returns which of the versions of message id shows, from 0, for
// a switcher to change, and how many there are. Changing it shows another
// version, and after it the conversation last shown there.
func (t *Tree[T]) Version(id int) (*int, int) {
	p := t.node(t.nodes[id].parent)
	return &p.at, len(p.kids)
}

// Show shows message id: the versions before it change to those leading
// to it, and after it the conversation follows the versions last shown.
func (t *Tree[T]) Show(id int) {
	for id >= 0 {
		parent := t.nodes[id].parent
		p := t.node(parent)
		p.at = slices.Index(p.kids, id)
		id = parent
	}
}

// Cut drops the messages after message id, in all their versions, as for
// going back to a checkpoint.
func (t *Tree[T]) Cut(id int) {
	n := t.nodes[id]
	n.kids, n.at = nil, 0
}

// Events are what the user did with a map in a frame. The zero Events
// report nothing.
type Events struct {
	shown int
	ok    bool
}

// Shown returns the message the user clicked in the map, which now shows,
// for the app to scroll the conversation to it.
func (e Events) Shown() (int, bool) { return e.shown, e.ok }

// dot is a message on a map: its ID, the dot of the message before it,
// -1 for none, and its column and row.
type dot struct {
	id, up, col, row int
}

// Map creates a map of tree, as large as the tree is; put it in a
// scroll view where it may not fit. Each message is a dot below the one
// before it, from a point at the top where the conversation starts, and
// the versions of a message side by side, the first below the message
// before them; the messages showing are lit. label names a message, as
// its dot's tooltip shows. A click on a dot shows its message, as Show
// does.
func Map[T any](c *ui.Context, tree *Tree[T], label func(m *T) string) Events {
	t := c.Theme()
	var ev Events
	var dots []dot
	cols, rows := 0, 0
	var walk func(id, up, row int)
	walk = func(id, up, row int) {
		i := len(dots)
		dots = append(dots, dot{id, up, cols, row})
		rows = max(rows, row+1)
		kids := tree.nodes[id].kids
		for _, k := range kids {
			walk(k, i, row+1)
		}
		if len(kids) == 0 {
			cols++
		}
	}
	for _, id := range tree.root.kids {
		walk(id, -1, 1)
	}
	lit := map[int]bool{}
	for _, id := range tree.Path() {
		lit[id] = true
	}
	step, pad := t.Space(6), t.Space(2)
	center := func(col, row int) (float32, float32) {
		return pad + (float32(col)+0.5)*step, pad + (float32(row)+0.5)*step
	}
	box := ui.Box(c).Size(2*pad+float32(cols)*step, 2*pad+float32(rows)*step).Role(ui.RoleGroup).Label("Branch map")
	box.Children(func() {
		for _, d := range dots {
			x, y := center(d.col, d.row)
			name := label(tree.Message(d.id))
			b := ui.ButtonBase(c.Key(d.id)).Absolute().Left(x-step/2).Top(y-step/2).Size(step, step).
				Radius(step / 2).Label(name).Tooltip(name)
			if b.Clicked() {
				tree.Show(d.id)
				ev = Events{d.id, true}
			}
			if b.Hovered() {
				b.Background(t.Text.Alpha(0.08))
			}
			on := lit[d.id]
			b.Draw(func(p *ui.Painter, r ui.Rect) {
				var circle ui.Path
				circle.Circle(r.X+r.W/2, r.Y+r.H/2, t.Space(1.25))
				if on {
					p.FillPath(&circle, t.Accent)
					return
				}
				p.FillPath(&circle, t.Background)
				p.StrokePath(&circle, 1.5, t.TextMuted)
			})
		}
	})
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if len(dots) == 0 {
			return
		}
		// Links off the path go first, so that those on it, sharing their
		// start, cover them.
		for _, on := range []bool{false, true} {
			for _, d := range dots {
				if lit[d.id] != on {
					continue
				}
				x0, y0 := center(0, 0)
				if d.up >= 0 {
					x0, y0 = center(dots[d.up].col, dots[d.up].row)
				}
				x1, y1 := center(d.col, d.row)
				var path ui.Path
				path.MoveTo(r.X+x0, r.Y+y0)
				if x1 != x0 {
					mid, k := (y0+y1)/2, step/4
					path.LineTo(r.X+x0, r.Y+mid-k).QuadTo(r.X+x0, r.Y+mid, r.X+x0+k, r.Y+mid).
						LineTo(r.X+x1-k, r.Y+mid).QuadTo(r.X+x1, r.Y+mid, r.X+x1, r.Y+mid+k)
				}
				path.LineTo(r.X+x1, r.Y+y1)
				color, width := t.Text.Alpha(0.25), float32(1.5)
				if on {
					color, width = t.Accent, 2
				}
				p.StrokePath(&path, width, color)
			}
		}
		x, y := center(0, 0)
		var start ui.Path
		start.Circle(r.X+x, r.Y+y, t.Space(0.75))
		p.FillPath(&start, t.Accent)
	})
	return ev
}

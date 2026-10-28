// Package explorer is the file tree of an editor's sidebar, as VS Code's
// Explorer: folders first, a folder holding only one folder shown with it
// on one row, as src/ui/widgets, and each file in the color of how
// version control sees it, with its letter at the end: M modified, A
// added, U untracked, ! in conflict, and ignored ones dimmed. A folder
// with changes inside shows in their color, with a dot.
//
//	ev := explorer.View(c, &app.files, &app.paths, app.status).Grow(1)
//	if ev.Opened != "" {
//		app.open(ev.Opened)
//	}
//
// Rows are chosen as an outline's, which the arrows open and close. A
// double click or Enter on a file reports it Opened, and on a folder
// opens or closes it. F2 renames the row chosen in place, Cmd+Backspace
// or Delete deletes it, and the context menu of a row, or of the space
// below the rows, does those and makes a new file or folder, named in
// place too: Enter keeps a name, Escape gives it up, and a name already
// taken is refused. Dragging a row onto a folder, or onto a file in it,
// moves it there, and onto the space below the rows to the top.
package explorer

import (
	"path"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Status is how version control sees a file or a folder.
type Status uint8

const (
	Unchanged Status = iota
	Modified
	Added
	Untracked
	Conflicted
	Ignored
)

func (st Status) letter() string {
	switch st {
	case Modified:
		return "M"
	case Added:
		return "A"
	case Untracked:
		return "U"
	case Conflicted:
		return "!"
	}
	return ""
}

func (st Status) color(t *ui.Theme) ui.Color {
	switch st {
	case Modified:
		return t.Warning
	case Added, Untracked:
		return t.Success
	case Conflicted:
		return t.Danger
	case Ignored:
		return t.TextMuted
	}
	return t.Text
}

// rank orders statuses by what a folder shows of those inside it.
func (st Status) rank() int {
	switch st {
	case Conflicted:
		return 3
	case Modified:
		return 2
	case Added, Untracked:
		return 1
	}
	return 0
}

// draft is the last part of the key of the row naming a new file or
// folder, which no path has.
const draft = "\x00new"

// State is what an explorer keeps: the folders open, the row chosen, and
// the row being named. Keep it in the app's state, one for each tree.
type State struct {
	tree ui.OutlineState[string]
	row  int
	// edit is the path renamed, or the key of the draft row; in is the
	// folder a new entry goes in, dir whether it is a folder.
	edit    string
	in      string
	dir     bool
	name    string
	focused bool
	fresh   bool
	reveal  string
	ask     int
	setup   bool
}

// Reveal opens the folders down to p and chooses it.
func (s *State) Reveal(p string) { s.reveal = p }

// NewFile names a new file in place, in the folder chosen or in that of
// the file chosen.
func (s *State) NewFile() { s.ask = 1 }

// NewFolder names a new folder in place, as NewFile a file.
func (s *State) NewFolder() { s.ask = 2 }

// Chosen returns the path of the row chosen, after View, or "" with none.
func (s *State) Chosen() string {
	if s.row < 0 || s.row >= s.tree.Rows() {
		return ""
	}
	if k := s.tree.Item(s.row); path.Base(k) != draft {
		return k
	}
	return ""
}

// Events is an explorer, and what the user did with it this frame:
// Opened a file; Created a file, or a folder, ending in "/"; renamed or
// moved a file or folder From a path To another; or Deleted one. The
// explorer has changed the paths already.
type Events struct {
	ui.Element
	Opened   string
	Created  string
	From, To string
	Deleted  string
}

// tree is the paths read: the children of each folder, "" for the top,
// in order, and the folders.
type tree struct {
	kids map[string][]string
	dirs map[string]bool
}

func read(paths []string) tree {
	tr := tree{kids: map[string][]string{"": {}}, dirs: map[string]bool{}}
	seen := map[string]bool{}
	add := func(p string) {
		for !seen[p] {
			seen[p] = true
			dir := path.Dir(p)
			if dir == "." {
				dir = ""
			}
			tr.kids[dir] = append(tr.kids[dir], p)
			if dir == "" {
				return
			}
			tr.dirs[dir] = true
			if tr.kids[dir] == nil {
				tr.kids[dir] = []string{}
			}
			p = dir
		}
	}
	for _, p := range paths {
		if d, ok := strings.CutSuffix(p, "/"); ok {
			tr.dirs[d] = true
			if tr.kids[d] == nil {
				tr.kids[d] = []string{}
			}
			add(d)
			continue
		}
		add(p)
	}
	for _, ks := range tr.kids {
		slices.SortFunc(ks, func(a, b string) int {
			if tr.dirs[a] != tr.dirs[b] {
				if tr.dirs[a] {
					return -1
				}
				return 1
			}
			return strings.Compare(strings.ToLower(path.Base(a)), strings.ToLower(path.Base(b)))
		})
	}
	return tr
}

// fold returns the folder whose row shows dir: the last of the folders
// each holding only the next, from dir down.
func (tr tree) fold(dir string) string {
	for tr.dirs[dir] && len(tr.kids[dir]) == 1 && tr.dirs[tr.kids[dir][0]] {
		dir = tr.kids[dir][0]
	}
	return dir
}

func (tr tree) has(p string) bool {
	return tr.dirs[p] || slices.Contains(tr.kids[parent(p)], p)
}

func parent(p string) string {
	if d := path.Dir(p); d != "." {
		return d
	}
	return ""
}

func join(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

// View creates the tree of *paths: files, with slashes, and empty
// folders, ending in "/". status holds how version control sees them,
// with a folder's status, as Ignored, holding for all inside it. Give it
// a height, or Grow.
func View(c *ui.Context, s *State, paths *[]string, status map[string]Status) Events {
	t := c.Theme()
	if !s.setup {
		s.row, s.setup = -1, true
		s.tree.List.Selected = &s.row
	}
	tr := read(*paths)
	var ev Events

	// label holds the text of each folder's row: the folders it shows,
	// from the one below its parent's row.
	label := map[string]string{}
	var walk func(dir string)
	walk = func(dir string) {
		for _, k := range tr.kids[dir] {
			if !tr.dirs[k] {
				continue
			}
			f := tr.fold(k)
			label[f] = strings.TrimPrefix(f, join(dir, ""))
			walk(f)
		}
	}
	walk("")
	if s.reveal != "" {
		for d := parent(s.reveal); d != ""; d = parent(d) {
			s.tree.Open.Add(tr.fold(d))
		}
	}

	of := func(p string) Status {
		for q := p; q != ""; q = parent(q) {
			if st, ok := status[q]; ok && (q == p || st == Ignored) {
				return st
			}
		}
		return Unchanged
	}
	inside := map[string]Status{}
	for p, st := range status {
		for d := parent(p); d != ""; d = parent(d) {
			if st.rank() > inside[d].rank() {
				inside[d] = st
			}
		}
	}

	list := func(dir string) []string {
		out := make([]string, 0, len(tr.kids[dir])+1)
		if s.edit == join(dir, draft) {
			out = append(out, s.edit)
		}
		for _, p := range tr.kids[dir] {
			if tr.dirs[p] {
				p = tr.fold(p)
			}
			out = append(out, p)
		}
		return out
	}
	children := func(k string) []string {
		if !tr.dirs[k] {
			return nil
		}
		return list(k)
	}

	create := func(dir bool) {
		in := s.Chosen()
		if !tr.dirs[in] {
			in = parent(in)
		}
		if in != "" {
			s.tree.Open.Add(in)
		}
		s.edit, s.in, s.dir, s.name, s.focused, s.fresh = join(in, draft), in, dir, "", false, true
	}
	if s.ask != 0 {
		create(s.ask == 2)
		s.ask = 0
	}
	rename := func(k string) {
		s.edit, s.name, s.focused, s.fresh = k, path.Base(k), false, true
	}
	remove := func(k string) {
		*paths = slices.DeleteFunc(*paths, func(p string) bool {
			return p == k || strings.HasPrefix(p, k+"/")
		})
		s.tree.Open.Remove(k)
		ev.Deleted = k
	}
	move := func(from, to string) bool {
		if from == to || tr.has(to) || strings.HasPrefix(to, from+"/") {
			return false
		}
		for i, p := range *paths {
			if p == from || strings.HasPrefix(p, from+"/") {
				(*paths)[i] = to + p[len(from):]
			}
		}
		if s.tree.Open.Has(from) {
			s.tree.Open.Remove(from)
			s.tree.Open.Add(to)
		}
		ev.From, ev.To, s.reveal = from, to, to
		return true
	}
	// keep keeps the name typed, reporting whether it was taken; done
	// gives it up.
	done := func() { s.edit = "" }
	keep := func() bool {
		name := strings.Trim(strings.TrimSpace(s.name), "/")
		if path.Base(s.edit) == draft {
			p := join(s.in, name)
			if name == "" {
				done()
				return true
			}
			if tr.has(p) {
				return false
			}
			if s.dir {
				p += "/"
			}
			*paths = append(*paths, p)
			ev.Created, s.reveal = p, strings.TrimSuffix(p, "/")
			done()
			return true
		}
		if name == "" || name == path.Base(s.edit) {
			done()
			return true
		}
		if !move(s.edit, join(parent(s.edit), name)) {
			return false
		}
		done()
		return true
	}
	menu := func(k string) func(m *ui.Menu) {
		return func(m *ui.Menu) {
			if m.Item("New File…").Chosen() {
				create(false)
			}
			if m.Item("New Folder…").Chosen() {
				create(true)
			}
			if k == "" {
				return
			}
			m.Separator()
			if m.Item("Rename…").Shortcut(0, ui.KeyF2).Chosen() {
				rename(k)
			}
			if m.Item("Delete").Shortcut(ui.Cmd, ui.KeyBackspace).Chosen() {
				remove(k)
			}
			m.Separator()
			if m.Item("Copy Path").Chosen() {
				c.WriteClipboard(k)
			}
		}
	}

	editor := func(k string) {
		in := ui.TextInputBase(c, &s.name).Grow(1).Padding(1, t.Space(1)).Background(t.Background).TextColor(t.Text).
			Border(1, t.Accent).Radius(2).Label("Name")
		if s.fresh {
			stem := len([]rune(strings.TrimSuffix(s.name, path.Ext(s.name))))
			if stem == 0 {
				stem = len([]rune(s.name))
			}
			in.Focus().SetTextSelection(0, stem)
			s.fresh = false
		}
		taken := false
		switch {
		case in.Submitted():
			taken = !keep()
		case in.Shortcut(0, ui.KeyEscape):
			done()
		case s.focused && !in.Focused():
			if !keep() {
				done()
			}
		}
		s.focused = in.Focused()
		if taken {
			in.Border(1, t.Danger).Error("A file or folder with this name is already there")
		}
	}

	view := ui.Outline(c, &s.tree, list(""), children, func(k string) {
		chosen := s.row >= 0 && s.row < s.tree.Rows() && s.tree.Item(s.row) == k
		row := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Grow(1)
		if k == s.edit {
			muted := t.TextMuted
			if chosen {
				muted = t.AccentText
			}
			row.Children(func() {
				icon(c, path.Base(k) == draft && s.dir || tr.dirs[k], muted)
				editor(k)
			})
			return
		}
		dir := tr.dirs[k]
		row.Drag(k).ContextMenu(menu(k))
		target := k
		if !dir {
			target = parent(k)
		}
		if from, over := ui.DragOver[string](row); over && from != target && parent(from) != target {
			row.Background(t.Accent.Alpha(0.15)).Radius(2)
		}
		if from, ok := ui.Drop[string](row); ok {
			move(from, join(target, path.Base(from)))
		}
		if row.Dragging() {
			row.Opacity(0.5)
		}
		st, ink, muted := of(k), t.Text, t.TextMuted
		mark := st
		if dir && st != Ignored {
			mark = inside[k]
		}
		if mark != Unchanged {
			ink = mark.color(t)
		}
		// Colors of statuses are unreadable on the accent of the row chosen.
		if chosen {
			ink, muted = t.AccentText, t.AccentText
		}
		name := path.Base(k)
		if dir {
			name = label[k]
		}
		row.Children(func() {
			icon(c, dir, muted)
			text := ui.Text(c, name).TextColor(ink).SingleLine().Shrink(1).Grow(1)
			if st == Ignored {
				text.Opacity(0.6)
			}
			switch {
			case dir && mark.rank() > 0:
				ui.Box(c).Size(6, 6).Radius(3).Background(ink).Margin(0, t.Space(1), 0, 0)
			case !dir && st.letter() != "":
				ui.Text(c, st.letter()).TextColor(ink).FontSize(t.FontSize - 1).Bold().Width(t.Space(3)).TextAlign(ui.Center)
			}
		})
	}).ContextMenu(menu(""))
	ev.Element = view
	if from, ok := ui.Drop[string](view); ok {
		move(from, path.Base(from))
	}

	if k := s.Chosen(); k != "" && s.edit == "" {
		switch {
		case view.Shortcut(0, ui.KeyF2):
			rename(k)
		case view.Shortcut(ui.Cmd, ui.KeyBackspace), view.Shortcut(0, ui.KeyDelete):
			remove(k)
		case view.Submitted():
			if tr.dirs[k] {
				if s.tree.Open.Has(k) {
					s.tree.Open.Remove(k)
				} else {
					s.tree.Open.Add(k)
				}
			} else {
				ev.Opened = k
			}
		}
	}
	if s.reveal != "" {
		for i := range s.tree.Rows() {
			if s.tree.Item(i) == s.reveal {
				s.row, s.reveal = i, ""
				view.Focus()
				break
			}
		}
		c.Invalidate()
	}
	return ev
}

// icon draws a folder or a file in color.
func icon(c *ui.Context, dir bool, color ui.Color) {
	ui.Box(c).Size(14, 14).Draw(func(p *ui.Painter, r ui.Rect) {
		x, y := r.X, r.Y
		if dir {
			var tab ui.Path
			tab.MoveTo(x+1, y+3).LineTo(x+5.5, y+3).LineTo(x+7, y+4.5).LineTo(x+13, y+4.5).
				LineTo(x+13, y+12).LineTo(x+1, y+12).Close()
			p.StrokePath(&tab, 1.2, color)
			return
		}
		var page ui.Path
		page.MoveTo(x+3, y+1).LineTo(x+8.5, y+1).LineTo(x+11.5, y+4).LineTo(x+11.5, y+13).LineTo(x+3, y+13).Close().
			MoveTo(x+8.5, y+1).LineTo(x+8.5, y+4).LineTo(x+11.5, y+4)
		p.StrokePath(&page, 1.2, color)
	})
}

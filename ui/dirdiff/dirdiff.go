// Package dirdiff compares two folders as a tree, as Beyond Compare and
// Meld do: each file and folder on a row with its size and time on the
// left and on the right, and between them "+" for one only on the right,
// "−" for one only on the left, "≠" for one that differs and "=" for one
// the same, its name in the color of that. A folder differs when anything
// in it does, and opens as it comes when it does. A header counts the
// files of each kind, with a button showing only those that differ. A
// click or the arrow keys choose a row, which Chosen reports.
//
//	left, _ := dirdiff.Read(os.DirFS("release-1.2"))
//	right, _ := dirdiff.Read(os.DirFS("release-1.3"))
//	dirdiff.View(c, &app.diff, dirdiff.Dir{Name: "release-1.2", Entries: left},
//		dirdiff.Dir{Name: "release-1.3", Entries: right}).Grow(1)
package dirdiff

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Entry is a file in a folder: its path in the folder, with slashes, its
// size and the time it was modified, and a digest of its contents, as a
// SHA-256, or none. Two files are the same when their digests are, or,
// without both, their sizes and times to the second.
type Entry struct {
	Path     string
	Size     int64
	Modified time.Time
	Sum      string
}

// Dir is a folder compared: its name, which heads its column, and its
// files.
type Dir struct {
	Name    string
	Entries []Entry
}

// Read lists the files in fsys, without their digests.
func Read(fsys fs.FS) ([]Entry, error) {
	var es []Entry
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		es = append(es, Entry{Path: p, Size: info.Size(), Modified: info.ModTime()})
		return nil
	})
	return es, err
}

type status uint8

const (
	same status = iota
	added
	removed
	changed
)

func compare(l, r *Entry) status {
	switch {
	case l == nil:
		return added
	case r == nil:
		return removed
	case l.Sum != "" && r.Sum != "":
		if l.Sum == r.Sum {
			return same
		}
	case l.Size == r.Size && l.Modified.Truncate(time.Second).Equal(r.Modified.Truncate(time.Second)):
		return same
	}
	return changed
}

// node is a file or a folder of either side: its files on each, nil where
// it lacks one, or for a folder what it holds on each, and how it compares.
type node struct {
	left, right *Entry
	size        [2]int64
	has         [2]bool
	status      status
	kids        []string
}

// State is what a comparison keeps: the folders open, the row chosen and
// whether it shows only what differs. Keep it in the app's state, one for
// each comparison.
type State struct {
	tree  ui.OutlineState[string]
	row   int
	only  bool
	seen  map[string]bool
	nodes map[string]*node
}

// Chosen returns the path of the file or folder chosen, after View, or
// false with none.
func (s *State) Chosen() (string, bool) {
	if s.row < 0 || s.row >= s.tree.Rows() {
		return "", false
	}
	return s.tree.Item(s.row), true
}

// View creates the comparison of left and right. Give it a height, or
// Grow.
func View(c *ui.Context, s *State, left, right Dir) ui.Element {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	if s.tree.List.Selected == nil {
		s.row = -1
		s.tree.List.Selected = &s.row
		s.seen = map[string]bool{}
	}
	s.nodes = map[string]*node{"": {kids: []string{}}}
	for side, d := range []Dir{left, right} {
		for i := range d.Entries {
			e := &d.Entries[i]
			n := s.at(e.Path)
			if side == 0 {
				n.left = e
			} else {
				n.right = e
			}
		}
	}
	var counts [4]int
	var fold func(p string) *node
	fold = func(p string) *node {
		n := s.nodes[p]
		if n.kids == nil {
			n.status = compare(n.left, n.right)
			counts[n.status]++
			for side, e := range []*Entry{n.left, n.right} {
				if e != nil {
					n.size[side], n.has[side] = e.Size, true
				}
			}
			return n
		}
		slices.SortFunc(n.kids, func(a, b string) int {
			if da, db := s.nodes[a].kids != nil, s.nodes[b].kids != nil; da != db {
				if da {
					return -1
				}
				return 1
			}
			return strings.Compare(a, b)
		})
		for i, k := range n.kids {
			kn := fold(k)
			for side := range 2 {
				n.size[side] += kn.size[side]
				n.has[side] = n.has[side] || kn.has[side]
			}
			switch {
			case i == 0:
				n.status = kn.status
			case n.status != kn.status:
				n.status = changed
			}
		}
		if n.status != same && p != "" && !s.seen[p] {
			s.seen[p] = true
			s.tree.Open.Add(p)
		}
		return n
	}
	fold("")
	kids := func(p string) []string {
		n := s.nodes[p]
		if !s.only || n.kids == nil {
			return n.kids
		}
		return slices.DeleteFunc(slices.Clone(n.kids), func(k string) bool { return s.nodes[k].status == same })
	}
	s.tree.List.Label = func(i int) string {
		p := s.tree.Item(i)
		return path.Base(p) + ", " + [...]string{"same", "only on the right", "only on the left", "differs"}[s.nodes[p].status]
	}
	cols := []ui.TableColumn{
		{Title: "Name", MinWidth: 160},
		{Title: left.Name, ID: "left", Width: 200, MinWidth: 120},
		{Title: "", ID: "status", Width: 36, Align: ui.Center, Fixed: true},
		{Title: right.Name, ID: "right", Width: 200, MinWidth: 120},
	}
	return ui.Column(c).Children(func() {
		ui.Row(c).Gap(t.Space(4)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(t.Space(3)).Grow(1).Wrap().Children(func() {
				for _, k := range []status{added, removed, changed, same} {
					if counts[k] > 0 {
						ui.Row(c.Key(k)).Gap(t.Space(1)).Children(func() {
							ui.Text(c, [...]string{"=", "+", "−", "≠"}[k]).Bold().TextColor(color(t, k))
							ui.Text(c, fmt.Sprintf("%d %s", counts[k], [...]string{"same", "added", "removed", "changed"}[k])).FontFeatures("tnum")
						})
					}
				}
			})
			ui.Toggle(c, &s.only, "Only differences")
		})
		ui.Divider(c)
		ui.OutlineTable(c, &s.tree, cols, kids(""), kids, func(p string, col int) {
			n := s.nodes[p]
			chosen := s.row >= 0 && s.row < s.tree.Rows() && s.tree.Item(s.row) == p
			switch col {
			case 0:
				name := ui.Text(c, path.Base(p)).SingleLine()
				if n.status != same && !chosen {
					name.TextColor(color(t, n.status))
				}
			case 1, 3:
				side := col / 2
				e := [2]*Entry{n.left, n.right}[side]
				if !n.has[side] {
					return
				}
				ui.Row(c).Gap(t.Space(3)).Grow(1).Children(func() {
					ui.Text(c, size(n.size[side])).FontFeatures("tnum").Width(t.Space(16)).TextAlign(ui.End).SingleLine()
					if e != nil {
						date := ui.Text(c, i18n.FormatDate(locale, e.Modified, i18n.DateOptions{Skeleton: "MMMdjm"})).
							FontFeatures("tnum").SingleLine().Shrink(1)
						if !chosen {
							date.TextColor(t.TextMuted)
						}
					}
				})
			case 2:
				mark := ui.Text(c, [...]string{"=", "+", "−", "≠"}[n.status]).Bold()
				if !chosen {
					mark.TextColor(color(t, n.status))
				}
			}
		}).Grow(1)
	})
}

// at returns the node of p, making it and its folders.
func (s *State) at(p string) *node {
	if n, ok := s.nodes[p]; ok {
		return n
	}
	n := &node{}
	s.nodes[p] = n
	dir := path.Dir(p)
	if dir == "." {
		dir = ""
	}
	parent := s.at(dir)
	if parent.kids == nil {
		parent.kids = []string{}
	}
	parent.kids = append(parent.kids, p)
	return n
}

func color(t *ui.Theme, k status) ui.Color {
	return [...]ui.Color{t.TextMuted, t.Success, t.Danger, t.Warning}[k]
}

// size writes n bytes as "820 B", "14.2 KB" or "3.1 MB".
func size(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d B", n)
	}
	f, units := float64(n), []string{"KB", "MB", "GB", "TB"}
	for i, u := range units {
		f /= 1000
		if f < 1000 || i == len(units)-1 {
			return fmt.Sprintf("%.1f %s", f, u)
		}
	}
	return ""
}

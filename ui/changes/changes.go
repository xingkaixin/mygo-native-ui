// Package changes lists the files an agent changed, as coding agents'
// review panels do: the totals, and a tree of the folders, each file with
// the lines added and removed, +12 −3, and a letter of how it changed, A,
// M or D. The app shows the diff of the file chosen, as with the editor's
// DiffView:
//
//	changes.View(c, &app.changes, app.files).Grow(1)
//	if f, ok := app.changes.Chosen(); ok && f.Path != app.diffPath {
//		app.diffPath, app.diff = f.Path, app.load(f) // an *editor.DiffState
//	}
package changes

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Change is how a file changed.
type Change uint8

const (
	Modified Change = iota
	Added
	Deleted
)

// File is a file changed: its path, with slashes, how it changed, and how
// many lines were added and removed.
type File struct {
	Path           string
	Change         Change
	Added, Removed int
}

// State is what a list keeps: the folders open and the row chosen. Keep
// it in the app's state, one for each list.
type State struct {
	tree ui.OutlineState[string]
	row  int
	// seen holds the folders shown before, so that those the user closed
	// stay closed while new ones open.
	seen  map[string]bool
	files map[string]File
}

// Chosen returns the file chosen, after View, or false with none or a
// folder chosen.
func (s *State) Chosen() (File, bool) {
	if s.row < 0 || s.row >= s.tree.Rows() {
		return File{}, false
	}
	f, ok := s.files[s.tree.Item(s.row)]
	return f, ok
}

// View creates the list of files: a header of how many changed and the
// lines added and removed in all, and the tree, folders first, open as
// they come. A click or the arrow keys choose a file, which Chosen
// reports. Give it a height, or Grow.
func View(c *ui.Context, s *State, files []File) ui.Element {
	t := c.Theme()
	if s.tree.List.Selected == nil {
		s.row = -1
		s.tree.List.Selected = &s.row
		s.seen = map[string]bool{}
	}
	s.files = make(map[string]File, len(files))
	kids := map[string][]string{}
	added, removed := 0, 0
	for _, f := range files {
		s.files[f.Path] = f
		added += f.Added
		removed += f.Removed
		for p := f.Path; p != ""; {
			dir := path.Dir(p)
			if dir == "." {
				dir = ""
			}
			if !slices.Contains(kids[dir], p) {
				kids[dir] = append(kids[dir], p)
			}
			if dir != "" && !s.seen[dir] {
				s.seen[dir] = true
				s.tree.Open.Add(dir)
			}
			p = dir
		}
	}
	for _, k := range kids {
		slices.SortFunc(k, func(a, b string) int {
			if da, db := kids[a] != nil, kids[b] != nil; da != db {
				if da {
					return -1
				}
				return 1
			}
			return strings.Compare(a, b)
		})
	}
	s.tree.List.Label = func(i int) string { return path.Base(s.tree.Item(i)) }
	return ui.Column(c).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
			noun := "files"
			if len(files) == 1 {
				noun = "file"
			}
			ui.Text(c, fmt.Sprintf("%d %s changed", len(files), noun)).Bold().Grow(1)
			counts(c, added, removed, false)
		})
		ui.Divider(c)
		ui.Outline(c, &s.tree, kids[""], func(p string) []string { return kids[p] }, func(p string) {
			chosen := s.row >= 0 && s.row < s.tree.Rows() && s.tree.Item(s.row) == p
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Grow(1).Shrink(1).Children(func() {
				name := ui.Text(c, path.Base(p)).SingleLine().Grow(1).Shrink(1)
				f, ok := s.files[p]
				if !ok {
					return
				}
				if f.Change == Deleted {
					name.Strikethrough()
					if !chosen {
						name.TextColor(t.TextMuted)
					}
				}
				counts(c, f.Added, f.Removed, chosen)
				letter(c, f.Change, chosen)
			})
		}).Grow(1).Padding(t.Space(1))
	})
}

// counts shows "+added −removed", leaving out those 0, in the colors of
// the text when plain, as on a row chosen.
func counts(c *ui.Context, added, removed int, plain bool) {
	t := c.Theme()
	green, red := t.Success, t.Danger
	if plain {
		green, red = ui.Color{}, ui.Color{}
	}
	ui.Row(c).Gap(t.Space(1)).Children(func() {
		if added > 0 {
			tint(ui.Text(c, fmt.Sprintf("+%d", added)).FontSize(t.FontSize-1).FontFeatures("tnum"), green)
		}
		if removed > 0 {
			tint(ui.Text(c, fmt.Sprintf("−%d", removed)).FontSize(t.FontSize-1).FontFeatures("tnum"), red)
		}
	})
}

// letter shows A, M or D for change, in its color unless plain.
func letter(c *ui.Context, change Change, plain bool) {
	t := c.Theme()
	s, color := "M", t.Warning
	switch change {
	case Added:
		s, color = "A", t.Success
	case Deleted:
		s, color = "D", t.Danger
	}
	if plain {
		color = ui.Color{}
	}
	tint(ui.Text(c, s).Bold().FontSize(t.FontSize-1).Width(t.Space(3)).TextAlign(ui.End), color)
}

// tint colors e with color, unless it is the zero Color, which leaves the
// color it takes from its row.
func tint(e ui.Element, color ui.Color) {
	if color != (ui.Color{}) {
		e.TextColor(color)
	}
}

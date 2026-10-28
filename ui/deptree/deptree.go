// Package deptree shows the dependencies of a project as a tree, as npm
// ls and cargo tree print them: each package with its version and size,
// opened by a click on its arrow, and a package already shown higher up
// marked deduped rather than opened again. A package present at more
// than one version has its version in the warning color. A search keeps
// the packages whose names match and the paths down to them, opened.
//
// Beside the tree is the chosen package: its size and that of all it
// brings in, the versions of it that are installed, the packages that
// depend on it, and why it is installed, every path to it from the
// project. The up and down arrows choose the previous and the next row,
// and the left and right arrows close and open the chosen one.
//
//	deptree.View(c, &app.deps, app.packages)
package deptree

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Package is an installed package: its name, version and size in bytes,
// and the indices of the packages it depends on. The first package is
// the project.
type Package struct {
	Name    string
	Version string
	Size    int64
	Deps    []int
}

// State is what a tree keeps: the rows opened or closed, the search and
// the chosen package. Keep it in the app's state, one for each tree.
type State struct {
	// flipped holds the rows, by path, whose open state is not the
	// default: only the project is open by default.
	flipped map[string]bool
	query   string
	// chosen is the path of the chosen row, empty for the project.
	chosen string
}

type row struct {
	path    []int
	key     string
	deduped bool
	open    bool
	leaf    bool
}

// View creates the tree of pkgs beside the chosen package. Give it a
// size, or grow it in its parent.
func View(c *ui.Context, s *State, pkgs []Package) ui.Element {
	t := c.Theme()
	if len(pkgs) == 0 {
		return ui.Box(c)
	}
	versions := map[string][]string{}
	for _, p := range pkgs {
		if !slices.Contains(versions[p.Name], p.Version) {
			versions[p.Name] = append(versions[p.Name], p.Version)
		}
	}
	rows := walk(s, pkgs)
	at := slices.IndexFunc(rows, func(r row) bool { return r.key == s.chosen })
	if at < 0 {
		at, s.chosen = 0, rows[0].key
	}
	return ui.Row(c).AlignItems(ui.Stretch).MinHeight(t.Space(80)).Border(1, t.Border).Radius(t.Radius + 2).Clip().Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Basis(0).Children(func() {
			ui.Box(c).Padding(t.Space(3)).Children(func() {
				ui.SearchField(c, &s.query).FillWidth().Label("Search dependencies")
			})
			ui.Divider(c)
			list := ui.Scroll(c).Basis(0).Grow(1).Focusable().Role(ui.RoleTree).Label("Dependencies")
			switch {
			case list.Shortcut(0, ui.KeyDown):
				at = min(at+1, len(rows)-1)
			case list.Shortcut(0, ui.KeyUp):
				at = max(at-1, 0)
			case list.Shortcut(0, ui.KeyRight) && !rows[at].open && !rows[at].leaf && !rows[at].deduped:
				toggle(s, rows[at])
			case list.Shortcut(0, ui.KeyLeft) && rows[at].open && len(rows[at].path) > 1:
				toggle(s, rows[at])
			}
			s.chosen = rows[at].key
			list.Children(func() {
				ui.Column(c).Padding(t.Space(1.5)).Children(func() {
					for i, r := range rows {
						line(c.Key(r.key), s, pkgs, r, versions, i == at, list.FocusWithin())
					}
					if len(rows) == 1 && s.query != "" {
						ui.Text(c, "No dependencies match").TextColor(t.TextMuted).Padding(t.Space(3))
					}
				})
			})
		})
		ui.Box(c).Width(1).FillHeight().Background(t.Border)
		detail(c, pkgs, rows[at].path[len(rows[at].path)-1], versions)
	})
}

func keyOf(path []int) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = fmt.Sprint(p)
	}
	return strings.Join(parts, "/")
}

func toggle(s *State, r row) {
	if s.flipped == nil {
		s.flipped = map[string]bool{}
	}
	if s.flipped[r.key] {
		delete(s.flipped, r.key)
	} else {
		s.flipped[r.key] = true
	}
}

// walk returns the rows shown: the tree in order, each package opened
// only where it first appears, through open rows, or with a search
// through the rows on the way to a match.
func walk(s *State, pkgs []Package) []row {
	query := strings.ToLower(strings.TrimSpace(s.query))
	seen := map[int]bool{0: true}
	var out []row
	// visit appends the rows of the subtree of path, and reports whether
	// it has a match.
	var visit func(path []int) bool
	visit = func(path []int) bool {
		id := path[len(path)-1]
		p := pkgs[id]
		r := row{path: path, key: keyOf(path), leaf: len(p.Deps) == 0}
		deduped := len(path) > 1 && seen[id]
		seen[id] = true
		r.deduped = deduped
		mine := len(out)
		out = append(out, r)
		match := query == "" || strings.Contains(strings.ToLower(p.Name), query)
		if deduped || r.leaf {
			if !match {
				out = out[:mine]
			}
			return match
		}
		open := (len(path) == 1) != s.flipped[r.key]
		if query != "" {
			open = true
		}
		before := len(out)
		found := false
		for _, d := range p.Deps {
			child := append(path[:len(path):len(path)], d)
			if visit(child) {
				found = true
			}
		}
		if !open {
			out = out[:before]
		}
		out[mine].open = len(out) > before
		if query != "" && !match && !found && len(path) > 1 {
			out = out[:mine]
			return false
		}
		return match || found
	}
	visit([]int{0})
	return out
}

// line creates the row r of the tree.
func line(c *ui.Context, s *State, pkgs []Package, r row, versions map[string][]string, chosen, focused bool) {
	t := c.Theme()
	p := pkgs[r.path[len(r.path)-1]]
	e := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(2), t.Space(1), t.Space(2)+t.Space(5)*float32(len(r.path)-1)).Radius(t.Space(1.5)).FocusRing(false).
		Role(ui.RoleTreeItem).Label(p.Name + " " + p.Version).Expanded(r.open)
	switch {
	case chosen && focused:
		e.Background(t.Accent)
	case chosen:
		e.Background(t.Text.Alpha(0.08))
	}
	if e.Clicked() {
		s.chosen = r.key
	}
	muted := t.TextMuted
	if chosen && focused {
		muted = t.AccentText.Alpha(0.75)
	}
	e.Children(func() {
		arrow := ui.ButtonBase(c).Size(t.Space(4), t.Space(4)).Center().Radius(t.Space(1)).Label("Open")
		if r.leaf || r.deduped {
			arrow.Disabled(true)
		} else if arrow.Clicked() {
			toggle(s, r)
			s.chosen = r.key
		}
		open := r.open
		arrow.Draw(func(pt *ui.Painter, b ui.Rect) {
			if r.leaf || r.deduped {
				return
			}
			cx, cy, d := b.X+b.W/2, b.Y+b.H/2, b.W*0.18
			pts := [][2]float32{{cx - d/2, cy - d}, {cx + d/2, cy}, {cx - d/2, cy + d}}
			if open {
				pts = [][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}
			}
			pt.StrokePolyline(pts, 1.5, muted)
		})
		name := ui.Text(c, p.Name).SingleLine()
		if len(r.path) == 1 {
			name.FontWeight(700)
		}
		switch {
		case r.deduped:
			name.TextColor(muted)
		case chosen && focused:
			name.TextColor(t.AccentText)
		}
		ver := ui.Text(c, p.Version).Font("monospace").FontSize(t.FontSize - 2).TextColor(muted).SingleLine()
		if len(versions[p.Name]) > 1 && !(chosen && focused) {
			ver.TextColor(t.Warning)
		}
		if r.deduped {
			ui.Text(c, "deduped").FontSize(t.FontSize-3).TextColor(muted).Padding(0, t.Space(1.5)).
				Border(1, muted.Alpha(0.4)).Radius(t.Space(1))
		}
		ui.Box(c).Grow(1)
		ui.Text(c, Size(p.Size)).FontSize(t.FontSize - 2).TextColor(muted).FontFeatures("tnum").SingleLine()
	})
	if chosen {
		e.ScrollIntoView()
	}
}

// detail creates the panel of package id.
func detail(c *ui.Context, pkgs []Package, id int, versions map[string][]string) {
	t := c.Theme()
	p := pkgs[id]
	reach := reachable(pkgs, id)
	var total int64
	for _, k := range reach {
		total += pkgs[k].Size
	}
	var dependents []int
	for k, q := range pkgs {
		if slices.Contains(q.Deps, id) {
			dependents = append(dependents, k)
		}
	}
	paths := whyPaths(pkgs, id, 50)
	ui.Scroll(c).Width(t.Space(84)).Background(t.Surface).Children(func() {
		ui.Column(c).Padding(t.Space(4)).Gap(t.Space(5)).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, p.Name).FontWeight(700).FontSize(t.FontSize + 3)
				ui.Text(c, p.Version).Font("monospace").TextColor(t.TextMuted)
			})
			section(c, "Size", func() {
				fact(c, "Own", Size(p.Size))
				if len(reach) > 1 {
					fact(c, "With "+count(len(reach)-1, "dependency", "dependencies"), Size(total))
				}
			})
			if vs := versions[p.Name]; len(vs) > 1 {
				section(c, "Installed versions", func() {
					for _, v := range vs {
						l := ui.Text(c.Key(v), v).Font("monospace").FontSize(t.FontSize - 1)
						if v == p.Version {
							l.FontWeight(700)
						} else {
							l.TextColor(t.Warning)
						}
					}
				})
			}
			if len(dependents) > 0 {
				section(c, "Depended on by", func() {
					for _, k := range dependents {
						q := pkgs[k]
						ui.Row(c.Key(k)).Gap(t.Space(1.5)).AlignItems(ui.Baseline).Children(func() {
							ui.Text(c, q.Name).FontSize(t.FontSize - 1)
							ui.Text(c, q.Version).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
						})
					}
				})
			}
			if id != 0 {
				section(c, "Why it is installed", func() {
					for i, path := range paths {
						names := make([]string, len(path))
						for k, n := range path {
							names[k] = pkgs[n].Name
						}
						ui.Text(c.Key(i), strings.Join(names, " › ")).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Wrap()
					}
				})
			}
		})
	})
}

func section(c *ui.Context, title string, body func()) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 2).FontWeight(600).TextColor(t.TextMuted)
		body()
	})
}

func fact(c *ui.Context, name, value string) {
	t := c.Theme()
	ui.Row(c.Key(name)).Gap(t.Space(2)).Children(func() {
		ui.Text(c, name).FontSize(t.FontSize - 1).Grow(1)
		ui.Text(c, value).FontSize(t.FontSize - 1).FontWeight(600).FontFeatures("tnum")
	})
}

// reachable returns id and every package it brings in, each once.
func reachable(pkgs []Package, id int) []int {
	seen := map[int]bool{id: true}
	out := []int{id}
	for i := 0; i < len(out); i++ {
		for _, d := range pkgs[out[i]].Deps {
			if !seen[d] {
				seen[d] = true
				out = append(out, d)
			}
		}
	}
	return out
}

// whyPaths returns up to limit paths from the project to id, shortest
// first, none passing through a package twice.
func whyPaths(pkgs []Package, id, limit int) [][]int {
	// Only packages that lead to id are walked, so that every branch ends
	// at it and the walk stops after limit paths.
	leads := map[int]bool{id: true}
	for grew := true; grew; {
		grew = false
		for k, p := range pkgs {
			if !leads[k] && slices.ContainsFunc(p.Deps, func(d int) bool { return leads[d] }) {
				leads[k], grew = true, true
			}
		}
	}
	var out [][]int
	var visit func(path []int)
	visit = func(path []int) {
		if len(out) >= limit {
			return
		}
		last := path[len(path)-1]
		if last == id {
			out = append(out, slices.Clone(path))
			return
		}
		for _, d := range pkgs[last].Deps {
			if leads[d] && !slices.Contains(path, d) {
				visit(append(path, d))
			}
		}
	}
	visit([]int{0})
	slices.SortStableFunc(out, func(a, b []int) int { return len(a) - len(b) })
	return out
}

// Size writes n bytes as npm does: 512 B, 14.2 kB, 3.1 MB.
func Size(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d B", n)
	case n < 1e6:
		return fmt.Sprintf("%.1f kB", float64(n)/1e3)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1e6)
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

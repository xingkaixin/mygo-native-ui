// Package procmon is a process monitor, as macOS's Activity Monitor: the
// processes in a table of their names, CPU with a graph of its last
// minute, memory, threads, PIDs and users, sorted by a click on a
// column's header, all in a list or as a tree of parents and children,
// and found by name or PID. Stop quits the process chosen after asking,
// or forces it to quit. A graph of the CPU of them all runs below.
//
//	app.procs.Sample(read()) // every second
//	if pid, force, ok := procmon.View(c, &app.procs).Quit(); ok {
//		app.kill(pid, force)
//	}
package procmon

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// keep is how many samples the graphs keep.
const keep = 60

// Process is a process as a sample saw it: its CPU in percent of one
// core, its memory in bytes.
type Process struct {
	PID, PPID int
	Name      string
	User      string
	CPU       float64
	Memory    uint64
	Threads   int
}

// State is what a monitor keeps: the last sample and the CPU of those
// before it, how its rows are sorted and found, the one chosen, and the
// question before stopping it. Keep it in the app's state.
type State struct {
	procs   []Process
	history map[int][]float64
	total   []float64

	sort  ui.SortOrder
	list  ui.ListState
	tree  ui.OutlineState[int]
	row   int
	trow  int
	mode  int
	query string
	// opened is whether the tree has opened its parents, as it does the
	// first time it shows.
	opened bool

	asking   bool
	stopping Process
}

// Sample takes a new reading of the processes.
func (s *State) Sample(procs []Process) {
	if s.history == nil {
		s.history = map[int][]float64{}
		s.row, s.trow = -1, -1
	}
	seen := make(map[int]bool, len(procs))
	sum := 0.0
	for _, p := range procs {
		seen[p.PID] = true
		s.history[p.PID] = last(append(s.history[p.PID], p.CPU))
		sum += p.CPU
	}
	for pid := range s.history {
		if !seen[pid] {
			delete(s.history, pid)
		}
	}
	s.total = last(append(s.total, sum))
	s.procs = procs
}

func last(v []float64) []float64 {
	if len(v) > keep {
		return v[len(v)-keep:]
	}
	return v
}

// Events are what the user did with a monitor in a frame.
type Events struct {
	pid          int
	force, asked bool
}

// Quit returns the process the user chose to quit, and whether to force
// it to; the app quits it.
func (e Events) Quit() (pid int, force, ok bool) { return e.pid, e.force, e.asked }

var columns = []ui.TableColumn{
	{Title: "Process Name", Sortable: true, MinWidth: 160},
	{Title: "% CPU", Width: 130, Align: ui.End, Sortable: true},
	{Title: "Memory", Width: 96, Align: ui.End, Sortable: true},
	{Title: "Threads", Width: 70, Align: ui.End, Sortable: true},
	{Title: "PID", Width: 70, Align: ui.End, Sortable: true},
	{Title: "User", Width: 100, Sortable: true},
}

// View creates the monitor of the processes sampled last. Give it a size,
// or grow it in its parent.
func View(c *ui.Context, s *State) Events {
	var ev Events
	t := c.Theme()
	if s.sort.Column == "" {
		s.sort = ui.SortOrder{Column: "% CPU", Descending: true}
	}
	byPID := make(map[int]Process, len(s.procs))
	for _, p := range s.procs {
		byPID[p.PID] = p
	}
	found := s.find(byPID)
	slices.SortFunc(found, s.compare)

	var chosen Process
	ok := false
	switch {
	case s.mode == 0 && s.row >= 0 && s.row < len(found):
		chosen, ok = found[s.row], true
	case s.mode == 1 && s.trow >= 0 && s.trow < s.tree.Rows():
		chosen, ok = byPID[s.tree.Item(s.trow)]
	}

	ui.Column(c).Grow(1).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			stop := ui.Button(c, "Stop").Disabled(!ok).Tooltip("Quit the process chosen")
			if stop.Clicked() && ok {
				s.asking, s.stopping = true, chosen
			}
			ui.Segmented(c, &s.mode, "All Processes", "Tree")
			ui.Spacer(c)
			ui.SearchField(c, &s.query).Width(t.Space(52)).Placeholder("Name or PID")
		})
		ui.Divider(c)
		s.list.Sort, s.tree.List.Sort = &s.sort, &s.sort
		s.list.KeepOffset, s.tree.List.KeepOffset = true, true
		if s.mode == 0 {
			s.list.Selected = &s.row
			s.list.Key = func(i int) any { return found[i].PID }
			ui.Table(c, &s.list, columns, len(found), func(row, col int) {
				s.cell(c, found[row], col, ok && found[row].PID == chosen.PID)
			}).Grow(1).Label("Processes")
		} else {
			roots, kids := tree(found)
			if !s.opened {
				s.opened = true
				for pid := range kids {
					s.tree.Open.Add(pid)
				}
			}
			s.tree.List.Selected = &s.trow
			ui.OutlineTable(c, &s.tree, columns, roots, func(pid int) []int {
				return kids[pid]
			}, func(pid int, col int) {
				s.cell(c, byPID[pid], col, ok && pid == chosen.PID)
			}).Grow(1).Label("Processes")
		}
		ui.Divider(c)
		s.footer(c)
	})

	switch ui.AlertDialog(c, &s.asking, "Are you sure you want to quit this process?",
		"Do you really want to quit “"+s.stopping.Name+"”?", "Force Quit", "Cancel", "Quit") {
	case 0:
		ev = Events{pid: s.stopping.PID, force: true, asked: true}
	case 2:
		ev = Events{pid: s.stopping.PID, asked: true}
	}
	return ev
}

// find returns the processes whose name or PID holds the query, in tree
// mode with their parents too.
func (s *State) find(byPID map[int]Process) []Process {
	q := strings.ToLower(strings.TrimSpace(s.query))
	if q == "" {
		return slices.Clone(s.procs)
	}
	shown := map[int]bool{}
	for _, p := range s.procs {
		if !strings.Contains(strings.ToLower(p.Name), q) && !strings.Contains(strconv.Itoa(p.PID), q) {
			continue
		}
		shown[p.PID] = true
		for pp, ok := byPID[p.PPID]; ok && s.mode == 1 && !shown[pp.PID]; pp, ok = byPID[pp.PPID] {
			shown[pp.PID] = true
		}
	}
	var found []Process
	for _, p := range s.procs {
		if shown[p.PID] {
			found = append(found, p)
		}
	}
	return found
}

func (s *State) compare(a, b Process) int {
	var d int
	switch s.sort.Column {
	case "Process Name":
		d = cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	case "% CPU":
		d = cmp.Compare(a.CPU, b.CPU)
	case "Memory":
		d = cmp.Compare(a.Memory, b.Memory)
	case "Threads":
		d = cmp.Compare(a.Threads, b.Threads)
	case "User":
		d = cmp.Compare(a.User, b.User)
	}
	if d == 0 {
		d = cmp.Compare(a.PID, b.PID)
	}
	if s.sort.Descending {
		d = -d
	}
	return d
}

// tree returns the PIDs of the processes found whose parents are not,
// and the children of each, in their order.
func tree(found []Process) ([]int, map[int][]int) {
	in := make(map[int]bool, len(found))
	for _, p := range found {
		in[p.PID] = true
	}
	var roots []int
	kids := map[int][]int{}
	for _, p := range found {
		if in[p.PPID] && p.PPID != p.PID {
			kids[p.PPID] = append(kids[p.PPID], p.PID)
		} else {
			roots = append(roots, p.PID)
		}
	}
	return roots, kids
}

// cell builds column col of p's row, on the accent color when chosen.
func (s *State) cell(c *ui.Context, p Process, col int, chosen bool) {
	t := c.Theme()
	line := t.Accent
	if chosen {
		line = t.AccentText
	}
	switch col {
	case 0:
		ui.Text(c, p.Name).SingleLine().Ellipsis("…")
	case 1:
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.End).FillWidth().Children(func() {
			spark(c, s.history[p.PID], 100, t.Space(14), t.Space(3.5), line)
			ui.Text(c, strconv.FormatFloat(p.CPU, 'f', 1, 64)).FontFeatures("tnum").Width(t.Space(10)).TextAlign(ui.End)
		})
	case 2:
		ui.Text(c, bytes(p.Memory)).FontFeatures("tnum")
	case 3:
		ui.Text(c, strconv.Itoa(p.Threads)).FontFeatures("tnum")
	case 4:
		ui.Text(c, strconv.Itoa(p.PID)).FontFeatures("tnum")
	case 5:
		ui.Text(c, p.User).SingleLine().Ellipsis("…")
	}
}

// spark draws the values, from 0 to top, as an area w by h in color.
func spark(c *ui.Context, values []float64, top float64, w, h float32, color ui.Color) ui.Element {
	return ui.Box(c).Size(w, h).Draw(func(p *ui.Painter, r ui.Rect) {
		if len(values) < 2 {
			return
		}
		peak := max(top, slices.Max(values))
		step := r.W / float32(keep-1)
		x0 := r.X + r.W - float32(len(values)-1)*step
		path := (&ui.Path{}).MoveTo(x0, r.Y+r.H)
		for i, v := range values {
			path.LineTo(x0+float32(i)*step, r.Y+r.H-r.H*float32(v/peak))
		}
		path.LineTo(r.X+r.W, r.Y+r.H).Close()
		p.FillPath(path, color.Alpha(0.35))
		line := make([][2]float32, len(values))
		for i, v := range values {
			line[i] = [2]float32{x0 + float32(i)*step, r.Y + r.H - r.H*float32(v/peak)}
		}
		p.StrokePolyline(line, 1, color)
	})
}

func (s *State) footer(c *ui.Context) {
	t := c.Theme()
	threads, memory := 0, uint64(0)
	for _, p := range s.procs {
		threads += p.Threads
		memory += p.Memory
	}
	cpu := 0.0
	if len(s.total) > 0 {
		cpu = s.total[len(s.total)-1]
	}
	ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(8)).AlignItems(ui.Center).Children(func() {
		stat := func(label, value string) {
			ui.Column(c.Key(label)).Gap(t.Space(0.5)).Children(func() {
				ui.Text(c, label).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				ui.Text(c, value).FontFeatures("tnum").Bold()
			})
		}
		stat("CPU", fmt.Sprintf("%.1f%%", cpu))
		stat("Processes", strconv.Itoa(len(s.procs)))
		stat("Threads", strconv.Itoa(threads))
		stat("Memory", bytes(memory))
		ui.Spacer(c)
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "CPU load").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			spark(c, s.total, 100, t.Space(60), t.Space(8), t.Accent).Background(t.Text.Alpha(0.04)).Radius(2)
		})
	})
}

// bytes writes n in the largest unit it fills: 812 KB, 1.24 GB.
func bytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatUint(n, 10) + " B"
	}
	v, i := float64(n), -1
	for v >= unit && i < 3 {
		v /= unit
		i++
	}
	format := "%.1f %cB"
	if v >= 100 {
		format = "%.0f %cB"
	}
	return fmt.Sprintf(format, v, "KMGT"[i])
}

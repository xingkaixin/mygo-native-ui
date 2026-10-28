// Package containers lists containers, as Docker Desktop does: a row for
// each with a dot of its status, its name and image, its published ports
// as links to localhost, its CPU as a sparkline and a percentage, its
// memory against its limit, and how long it has been up or since it
// exited. Containers of a Compose project fall under a row of the project
// that folds them away and counts those running. A search narrows the
// rows by name, image or project, a switch hides those not running, and
// the buttons at the end of a row start or stop it, restart it, or remove
// it, which the app does.
//
//	ev := containers.View(c, &app.list, app.containers)
//	if ev.ID != "" {
//		docker(ev.Action, ev.ID)
//	}
package containers

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Status is what a container is doing.
type Status uint8

const (
	Running Status = iota
	Paused
	Restarting
	Exited
)

var statusNames = [...]string{"Running", "Paused", "Restarting", "Exited"}

func (s Status) String() string { return statusNames[s] }

// Port is a port published from a container to the host.
type Port struct {
	Host, Container int
}

// Container is a container: its ID, name, image and Compose project, if
// any; its status, since when, and the code it exited with; its published
// ports; its recent CPU use in percent, oldest first; and its memory use
// and limit in bytes.
type Container struct {
	ID, Name, Image, Project string
	Status                   Status
	Since                    time.Time
	ExitCode                 int
	Ports                    []Port
	CPU                      []float64
	Memory, MemoryLimit      int64
}

// Action is what a row's buttons ask of a container.
type Action uint8

const (
	Start Action = iota
	Stop
	Restart
	Remove
)

// State is what a list keeps: its search, whether it shows only the
// containers running, and the projects folded. Keep it in the app's
// state, one for each list.
type State struct {
	query    string
	running  bool
	collapse map[string]bool
}

// Events is a list, and the action asked of a container in it in the
// frame: ID is empty when none was.
type Events struct {
	ui.Element
	Action Action
	ID     string
}

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	startIcon   = icon(`<path d="M7 5v14l11-7z" fill="currentColor"/>`)
	stopIcon    = icon(`<rect x="6" y="6" width="12" height="12" rx="1.5" fill="currentColor"/>`)
	restartIcon = icon(`<path d="M20 11a8 8 0 1 0-2.3 5.7"/><path d="M20 4v7h-7"/>`)
	removeIcon  = icon(`<path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3"/>`)
	chevron     = icon(`<path d="M9 6l6 6-6 6"/>`)
)

func (s *State) shows(ct Container) bool {
	if s.running && ct.Status != Running {
		return false
	}
	q := strings.ToLower(strings.TrimSpace(s.query))
	return q == "" || strings.Contains(strings.ToLower(ct.Name+" "+ct.Image+" "+ct.Project), q)
}

// View creates the list of containers.
func View(c *ui.Context, s *State, containers []Container) Events {
	t := c.Theme()
	var ev Events
	if s.collapse == nil {
		s.collapse = map[string]bool{}
	}
	now := c.Now()
	c.After(time.Minute)
	running, memory := 0, int64(0)
	for _, ct := range containers {
		if ct.Status == Running {
			running++
			memory += ct.Memory
		}
	}
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.SearchField(c, &s.query).Width(t.Space(60))
			ui.Checkbox(c, &s.running, "Only running")
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d running · %s", running, bytes(memory))).TextColor(t.TextMuted)
		})
		tracks := []ui.Track{ui.Fr(1.4), ui.Fr(1.2), ui.Fixed(t.Space(28)), ui.Fixed(t.Space(30)), ui.Fixed(t.Space(32)), ui.Fixed(t.Space(38)), ui.Fixed(t.Space(24))}
		ui.Grid(c).ColumnTracks(tracks...).GapX(t.Space(3)).AlignItems(ui.Center).Children(func() {
			for _, h := range []string{"Name", "Image", "Ports", "CPU", "Memory", "Status", ""} {
				ui.Text(c.Key("head"+h), h).FontSize(t.FontSize-2).Bold().TextColor(t.TextMuted).Padding(t.Space(1.5), 0)
			}
			var projects []string
			for _, ct := range containers {
				if ct.Project != "" && !slices.Contains(projects, ct.Project) {
					projects = append(projects, ct.Project)
				}
			}
			for _, p := range projects {
				var members []Container
				for _, ct := range containers {
					if ct.Project == p && s.shows(ct) {
						members = append(members, ct)
					}
				}
				if len(members) == 0 {
					continue
				}
				s.project(c, p, members)
				if s.collapse[p] {
					continue
				}
				for _, ct := range members {
					if a, ok := row(c, ct, true, now); ok {
						ev.Action, ev.ID = a, ct.ID
					}
				}
			}
			for _, ct := range containers {
				if ct.Project == "" && s.shows(ct) {
					if a, ok := row(c, ct, false, now); ok {
						ev.Action, ev.ID = a, ct.ID
					}
				}
			}
		})
	})
	if ev.ID != "" {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// project creates the row of project p over its containers shown.
func (s *State) project(c *ui.Context, p string, members []Container) {
	t := c.Theme()
	up := 0
	for _, ct := range members {
		if ct.Status == Running {
			up++
		}
	}
	head := ui.ButtonBase(c.Key("project/"+p)).ColumnSpan(7).Padding(t.Space(2), t.Space(1)).Radius(t.Radius).
		Role(ui.RoleDisclosure).Expanded(!s.collapse[p]).Label(p)
	if head.Hovered() {
		head.Background(t.SurfaceHover)
	}
	if head.Clicked() {
		s.collapse[p] = !s.collapse[p]
	}
	turn := float32(90)
	if s.collapse[p] {
		turn = 0
	}
	turn = head.AnimateTo("containers.chevron", turn, ui.Snappy)
	head.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, chevron).Size(t.Space(3.5), t.Space(3.5)).TextColor(t.TextMuted).Rotate(turn)
			ui.Text(c, p).Bold()
			ui.Text(c, fmt.Sprintf("%d of %d running", up, len(members))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
	})
}

var dots = [...]ui.Color{Running: ui.Hex("#30B158"), Paused: ui.Hex("#E9A23B"), Restarting: ui.Hex("#2F8CF0"), Exited: ui.Hex("#9A9AA0")}

// row creates the cells of ct, indented under its project, and returns
// the action asked of it.
func row(c *ui.Context, ct Container, indented bool, now time.Time) (Action, bool) {
	t := c.Theme()
	var act Action
	asked := false
	key := func(cell string) *ui.Context { return c.Key(ct.ID + "/" + cell) }
	muted := ct.Status == Exited
	ui.Row(key("name")).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(2), 0).Children(func() {
		if indented {
			ui.Box(c).Width(t.Space(3.5))
		}
		ui.Box(c).Size(8, 8).Radius(4).Shrink(0).Background(dots[ct.Status]).Tooltip(ct.Status.String())
		ui.Column(c).Shrink(1).Children(func() {
			name := ui.Text(c, ct.Name).SingleLine().Ellipsis("…")
			if muted {
				name.TextColor(t.TextMuted)
			}
			ui.Text(c, ct.ID[:min(12, len(ct.ID))]).Font("monospace").FontSize(t.FontSize - 3).TextColor(t.TextMuted)
		})
	})
	ui.Text(key("image"), ct.Image).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
	ui.Column(key("ports")).Children(func() {
		for _, p := range ct.Ports {
			label := fmt.Sprintf("%d:%d", p.Host, p.Container)
			if ct.Status == Running {
				ui.Link(c.Key(label), label, fmt.Sprintf("http://localhost:%d", p.Host)).FontFeatures("tnum")
			} else {
				ui.Text(c.Key(label), label).TextColor(t.TextMuted).FontFeatures("tnum")
			}
		}
	})
	ui.Row(key("cpu")).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		if ct.Status != Running || len(ct.CPU) == 0 {
			ui.Text(c, "—").TextColor(t.TextMuted)
			return
		}
		ui.Box(c).Size(t.Space(14), t.Space(5)).Draw(func(p *ui.Painter, r ui.Rect) {
			sparkline(p, r, ct.CPU, t.Accent)
		})
		ui.Text(c, fmt.Sprintf("%.1f%%", ct.CPU[len(ct.CPU)-1])).FontFeatures("tnum").FontSize(t.FontSize - 1)
	})
	ui.Column(key("memory")).Gap(t.Space(1)).Children(func() {
		if ct.Status != Running {
			ui.Text(c, "—").TextColor(t.TextMuted)
			return
		}
		ui.Text(c, bytes(ct.Memory)+" / "+bytes(ct.MemoryLimit)).FontSize(t.FontSize - 1).FontFeatures("tnum")
		share := float32(ct.Memory) / float32(max(1, ct.MemoryLimit))
		ui.Box(c).Height(3).Radius(1.5).Background(t.Text.Alpha(0.1)).Draw(func(p *ui.Painter, r ui.Rect) {
			col := t.Accent
			if share > 0.85 {
				col = t.Danger
			}
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * min(1, share), H: r.H}, col, 1.5)
		})
	})
	status := "Up " + since(now, ct.Since)
	switch ct.Status {
	case Exited:
		status = fmt.Sprintf("Exited (%d) %s ago", ct.ExitCode, since(now, ct.Since))
	case Paused:
		status = "Paused"
	case Restarting:
		status = "Restarting"
	}
	ui.Text(key("status"), status).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
	ui.Row(key("actions")).Gap(t.Space(0.5)).Justify(ui.End).Children(func() {
		button := func(ic *ui.SVG, label string, a Action) {
			b := ui.ButtonBase(c.Key(label)).Size(t.Space(7), t.Space(7)).Radius(t.Radius).Center().Label(label + " " + ct.Name).Tooltip(label)
			if b.Hovered() {
				b.Background(t.SurfaceHover)
			}
			b.Children(func() { ui.Icon(c, ic).Size(t.Space(3.5), t.Space(3.5)).TextColor(t.TextMuted) })
			if b.Clicked() {
				act, asked = a, true
			}
		}
		if ct.Status == Exited {
			button(startIcon, "Start", Start)
		} else {
			button(stopIcon, "Stop", Stop)
		}
		button(restartIcon, "Restart", Restart)
		button(removeIcon, "Remove", Remove)
	})
	return act, asked
}

// sparkline draws values, from 0 to the larger of 100 and their peak, as
// a line over a faint fill.
func sparkline(p *ui.Painter, r ui.Rect, values []float64, col ui.Color) {
	top := 100.0
	for _, v := range values {
		top = max(top, v)
	}
	var line, area ui.Path
	area.MoveTo(r.X, r.Y+r.H)
	for i, v := range values {
		x := r.X + r.W*float32(i)/float32(max(1, len(values)-1))
		y := r.Y + r.H - r.H*float32(v/top)
		if i == 0 {
			line.MoveTo(x, y)
		} else {
			line.LineTo(x, y)
		}
		area.LineTo(x, y)
	}
	area.LineTo(r.X+r.W, r.Y+r.H).Close()
	p.FillPath(&area, col.Alpha(0.15))
	p.StrokePath(&line, 1.5, col)
}

func since(now, t time.Time) string {
	switch d := now.Sub(t); {
	case d < time.Minute:
		return "a few seconds"
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}

func bytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%d KB", n>>10)
}

// Package tooltoggles turns an assistant's tools on and off, grouped by
// the server that gives them, as the tool servers of an AI agent are
// set up. Each server is a row with how many of its tools are on and a
// switch for all of them: on once all are, and a click turns all on, or
// all off when all are. A click on the row opens its tools, each with a
// switch, what it does, and a tag on those that change things. Searching
// shows the tools whose name or description has the words, under their
// servers, opened, and a count of the tools on closes the list.
//
//	tooltoggles.View(c, &app.tools, &app.servers)
package tooltoggles

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Tool is a tool: its name, what it does, whether it is on, and whether it
// changes things rather than only reads them.
type Tool struct {
	Name        string
	Description string
	On          bool
	Writes      bool
}

// Server is a server and the tools it gives.
type Server struct {
	Name  string
	Tools []Tool
}

// State is what a list keeps: the search and the servers opened. Keep it
// in the app's state, one for each list.
type State struct {
	query string
	open  map[string]bool
}

var chevron = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M9 6l6 6-6 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

// View creates the list of *servers' tools.
func View(c *ui.Context, s *State, servers *[]Server) ui.Element {
	t := c.Theme()
	if s.open == nil {
		s.open = map[string]bool{}
	}
	words := strings.Fields(strings.ToLower(s.query))
	matches := func(sv Server, tl Tool) bool {
		text := strings.ToLower(sv.Name + " " + tl.Name + " " + tl.Description)
		for _, w := range words {
			if !strings.Contains(text, w) {
				return false
			}
		}
		return true
	}
	root := ui.Column(c).Gap(t.Space(3))
	changed := false
	root.Children(func() {
		ui.SearchField(c, &s.query).FillWidth().Label("Search tools")
		ui.Column(c).FillWidth().Radius(t.Space(2.5)).Border(1, t.Border).Clip().Children(func() {
			shown := 0
			for si := range *servers {
				sv := &(*servers)[si]
				var tools []int
				for ti, tl := range sv.Tools {
					if matches(*sv, tl) {
						tools = append(tools, ti)
					}
				}
				if len(tools) == 0 {
					continue
				}
				on := 0
				for _, tl := range sv.Tools {
					if tl.On {
						on++
					}
				}
				open := s.open[sv.Name] || len(words) > 0
				ui.Column(c.Key(sv.Name)).FillWidth().Children(func() {
					if shown > 0 {
						ui.Box(c).FillWidth().Height(1).Background(t.Border)
					}
					shown++
					ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, t.Space(3), 0, 0).Children(func() {
						head := ui.ButtonBase(c).Grow(1).Gap(t.Space(2.5)).AlignItems(ui.Center).Padding(t.Space(2.5), t.Space(3)).
							Cursor(ui.CursorPointer).Label(sv.Name).Expanded(open)
						if head.Hovered() {
							head.Background(t.SurfaceHover)
						}
						if head.Clicked() && len(words) == 0 {
							s.open[sv.Name] = !s.open[sv.Name]
						}
						head.Children(func() {
							turn := float32(0)
							if open {
								turn = 90
							}
							ui.Icon(c, chevron).Size(t.Space(3.5), t.Space(3.5)).TextColor(t.TextMuted).Rotate(turn)
							mark(c, sv.Name)
							ui.Text(c, sv.Name).Bold().Grow(1)
							ui.Text(c, fmt.Sprintf("%d of %d", on, len(sv.Tools))).FontSize(t.FontSize - 1).
								FontFeatures("tnum").TextColor(t.TextMuted)
						})
						all := on == len(sv.Tools)
						sw := ui.Switch(c, &all).Label("All " + sv.Name + " tools")
						if on > 0 && !all {
							sw.Mixed()
						}
						if sw.Changed() {
							for ti := range sv.Tools {
								sv.Tools[ti].On = all
							}
							changed = true
						}
					})
					if !open {
						return
					}
					for _, ti := range tools {
						tl := &sv.Tools[ti]
						ui.Row(c.Key(ti)).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).
							Padding(t.Space(2), t.Space(3), t.Space(2), t.Space(12)).Background(t.Surface.Alpha(0.5)).Children(func() {
							ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
								ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
									ui.Text(c, tl.Name).Font("monospace").FontSize(t.FontSize - 1)
									if tl.Writes {
										ui.Text(c, "Writes").FontSize(t.FontSize-3).Bold().TextColor(ui.Hex("#C2410C")).
											Padding(1, t.Space(1.5)).Radius(t.Space(1)).Background(ui.Hex("#F97316").Alpha(0.14))
									}
								})
								ui.Text(c, tl.Description).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
							})
							if ui.Switch(c, &tl.On).Label(tl.Name).Changed() {
								changed = true
							}
						})
					}
				})
			}
			if shown == 0 {
				ui.Text(c, "No tools match").TextColor(t.TextMuted).Padding(t.Space(4))
			}
		})
		on, all := 0, 0
		for _, sv := range *servers {
			for _, tl := range sv.Tools {
				all++
				if tl.On {
					on++
				}
			}
		}
		ui.Text(c, fmt.Sprintf("%d of %d tools on", on, all)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// mark creates a server's mark: its initial on a color of its own.
func mark(c *ui.Context, name string) {
	t := c.Theme()
	h := fnv.New32a()
	h.Write([]byte(name))
	initial := ""
	for _, r := range name {
		initial = strings.ToUpper(string(r))
		break
	}
	ui.Box(c).Size(t.Space(6), t.Space(6)).Radius(t.Space(1.5)).Background(ui.Oklch(0.62, 0.13, float32(h.Sum32()%360))).
		Center().Children(func() {
		ui.Text(c, initial).FontSize(t.FontSize - 1).Bold().TextColor(ui.RGB(255, 255, 255))
	})
}

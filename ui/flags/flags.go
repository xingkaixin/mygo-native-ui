// Package flags switches feature flags across environments, as the
// dashboards of flag services do: a row a flag, with its key and what it
// does, and a column an environment, each cell a switch and the share of
// users the flag reaches there, a pie by its percent. A click on the share
// opens a slider to roll the flag out further or back. The last
// environment is taken as production: a switch there asks first, and
// changes only if the change is confirmed. The search field above keeps
// the flags whose key or description has what is typed.
//
//	flags.View(c, &app.flagList, []string{"Development", "Staging", "Production"}, &app.flags)
//
// Changed reports a flag switched or rolled out.
package flags

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Env is a flag in an environment: whether it is on, and the percent of
// users it reaches while on, from 0 to 100.
type Env struct {
	On      bool
	Percent int
}

// Flag is a feature flag: its key, what it does, and its state in each
// environment, in the order of the names given to View.
type Flag struct {
	Key, Description string
	Envs             []Env
}

// cell is an environment of a flag, by their indexes.
type cell struct{ flag, env int }

// State is what a list of flags keeps: the search, the switch waiting to
// be confirmed, and the share being rolled out. Keep it in the app's
// state, one for each list.
type State struct {
	query   string
	asking  bool
	pending cell
	rolling bool
	rollout cell
	slider  float64
}

// View creates the flags of *flags in the environments named by envs.
func View(c *ui.Context, s *State, envs []string, flags *[]Flag) ui.Element {
	t := c.Theme()
	fl := *flags
	changed := false
	colW := t.Space(30)
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.SearchField(c, &s.query).Label("Search flags")
		q := strings.ToLower(strings.TrimSpace(s.query))
		var shown []int
		for i, f := range fl {
			if q == "" || strings.Contains(strings.ToLower(f.Key), q) || strings.Contains(strings.ToLower(f.Description), q) {
				shown = append(shown, i)
			}
		}
		ui.Column(c).Radius(t.Space(2)).Border(1, t.Border).Clip().Gap(1).Dividers(1, t.Border).Children(func() {
			ui.Row(c).Padding(t.Space(2), t.Space(4)).Background(t.Surface).Children(func() {
				ui.Text(c, "Flag").FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted).Grow(1)
				for k, name := range envs {
					ui.Text(c.Key(k), name).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted).Width(colW).TextAlign(ui.Center)
				}
			})
			if len(shown) == 0 {
				msg := "No flags"
				if q != "" {
					msg = "No flags match “" + strings.TrimSpace(s.query) + "”"
				}
				ui.Text(c, msg).TextColor(t.TextMuted).Padding(t.Space(6)).AlignSelf(ui.Center)
			}
			for _, i := range shown {
				f := &fl[i]
				ui.Row(c.Key(f.Key)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
						ui.Text(c, f.Key).Font("monospace").Bold().SingleLine()
						if f.Description != "" {
							ui.Text(c, f.Description).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
						}
					})
					for k := range envs {
						if k >= len(f.Envs) {
							ui.Box(c.Key(k)).Width(colW)
							continue
						}
						if s.cell(c.Key(k), envs, f, cell{i, k}, colW) {
							changed = true
						}
					}
				})
			}
		})
		if s.asking {
			p := s.pending
			if p.flag < len(fl) && p.env < len(fl[p.flag].Envs) {
				e := &fl[p.flag].Envs[p.env]
				verb := "on"
				if e.On {
					verb = "off"
				}
				title := "Turn " + verb + " " + fl[p.flag].Key + " in " + envs[p.env] + "?"
				if ui.AlertDialog(c, &s.asking, title, "Users see the change right away.", "Cancel", "Turn "+strings.ToUpper(verb[:1])+verb[1:]) == 1 {
					e.On, changed = !e.On, true
				}
			} else {
				s.asking = false
			}
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// cell creates the switch and share of at, and reports a change.
func (s *State) cell(c *ui.Context, envs []string, f *Flag, at cell, w float32) (changed bool) {
	t := c.Theme()
	e := &f.Envs[at.env]
	ui.Row(c).Width(w).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		on := e.On
		if ui.Switch(c, &on).Label(f.Key + " in " + envs[at.env]).Changed() {
			if at.env == len(envs)-1 {
				s.asking, s.pending = true, at
			} else {
				e.On, changed = on, true
			}
		}
		share := ui.ButtonBase(c).Width(t.Space(14)).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(1.5)).Radius(t.Space(1.5)).
			AlignItems(ui.Center).Label("Rollout").Value(strconv.Itoa(e.Percent) + "%")
		if share.Hovered() {
			share.Background(t.SurfaceHover)
		}
		col := t.Accent
		if !e.On {
			col = t.TextMuted.Alpha(0.6)
		}
		share.Children(func() {
			frac := float32(max(0, min(100, e.Percent))) / 100
			ui.Box(c).Size(12, 12).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
				cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2
				var ring ui.Path
				ring.Circle(cx, cy, rad-0.75)
				p.StrokePath(&ring, 1.5, col)
				if frac == 0 {
					return
				}
				var pie ui.Path
				if frac >= 1 {
					pie.Circle(cx, cy, rad-0.75)
				} else {
					a0 := -math.Pi / 2
					a1 := a0 + 2*math.Pi*float64(frac)
					pie.MoveTo(cx, cy)
					for k := 0; k <= 24; k++ {
						a := a0 + (a1-a0)*float64(k)/24
						pie.LineTo(cx+(rad-0.75)*float32(math.Cos(a)), cy+(rad-0.75)*float32(math.Sin(a)))
					}
					pie.Close()
				}
				p.FillPath(&pie, col)
			})
			txt := ui.Text(c, strconv.Itoa(e.Percent)+"%").FontSize(t.FontSize - 1).FontFeatures("tnum")
			if !e.On {
				txt.TextColor(t.TextMuted)
			}
		})
		if share.Clicked() {
			s.rolling, s.rollout, s.slider = true, at, float64(e.Percent)
		}
		if s.rolling && s.rollout == at {
			ui.Popover(c, share, &s.rolling, func() {
				ui.Column(c).Width(t.Space(60)).Gap(t.Space(2)).Padding(t.Space(1)).Children(func() {
					ui.Text(c, "Rollout in "+envs[at.env]).Bold()
					if ui.Slider(c, &s.slider, 0, 100).Step(5).Label("Percent of users").Changed() {
						e.Percent, changed = int(math.Round(s.slider)), true
					}
					note := strconv.Itoa(e.Percent) + "% of users"
					if !e.On {
						note += " once it is on"
					}
					ui.Text(c, note).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				})
			})
		}
	})
	return changed
}

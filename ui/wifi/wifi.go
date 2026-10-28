// Package wifi builds the Wi-Fi menu of a menu bar panel, as macOS shows
// it: a switch, the network joined in an accent circle, the known
// networks in range, and the other networks under a disclosure, each with
// bars for its signal and a lock when it asks for a password. A click on
// a network joins it, through a password field that opens in place for a
// secured network the Mac does not know yet; the row spins while the app
// joins, until *joined names it, or the app reports what went wrong with
// Fail, which opens the field again with the message.
//
//	ev := wifi.View(c, &app.menu, &app.on, &app.joined, app.networks)
//	if ev.Join != nil {
//		go connect(ev.Join.SSID, ev.Password)
//	}
package wifi

import (
	"cmp"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Network is a network in range: its name, its signal from 0 to 1,
// whether it asks for a password, and whether it was joined before.
type Network struct {
	SSID   string
	Signal float64
	Secure bool
	Known  bool
}

// State is what a menu keeps: whether the other networks show, the
// network whose password is asked, what was typed, the network being
// joined, and what went wrong. Keep it in the app's state, one for each
// menu.
type State struct {
	others   bool
	asking   string
	password string
	joining  string
	problem  string
}

// Fail stops joining the network being joined, and asks for its
// password again above msg, as "Incorrect password".
func (s *State) Fail(msg string) {
	if s.joining != "" {
		s.asking, s.problem, s.joining = s.joining, msg, ""
	}
}

// Events is a menu, and what was asked of it in the frame: Join, nil
// when nothing, the network to join with Password, empty for a network
// that asks none or is known; and Settings, a click on Wi-Fi Settings.
type Events struct {
	ui.Element
	Join     *Network
	Password string
	Settings bool
}

// View creates the menu of the networks in range, switched by *on, with
// *joined the SSID of the network joined, empty for none.
func View(c *ui.Context, s *State, on *bool, joined *string, networks []Network) Events {
	t := c.Theme()
	var ev Events
	if s.joining != "" && s.joining == *joined {
		s.joining = ""
	}
	if !*on {
		s.joining, s.asking = "", ""
	}
	root := ui.Column(c).Width(t.Space(78)).Padding(t.Space(1.5)).Gap(t.Space(0.5))
	root.Children(func() {
		ui.Row(c).Padding(t.Space(1.5), t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Wi-Fi").Bold().Grow(1)
			if ui.Switch(c, on).Label("Wi-Fi").Changed() && !*on {
				*joined = ""
			}
		})
		if *on {
			var current, known, other []Network
			for _, n := range networks {
				switch {
				case n.SSID == *joined:
					current = append(current, n)
				case n.Known:
					known = append(known, n)
				default:
					other = append(other, n)
				}
			}
			strongest := func(a, b Network) int { return cmp.Compare(b.Signal, a.Signal) }
			slices.SortFunc(known, strongest)
			slices.SortFunc(other, strongest)
			list := func(head string, ns []Network) {
				if len(ns) == 0 {
					return
				}
				menuDivider(c)
				ui.Text(c, head).FontSize(t.FontSize-2).Bold().TextColor(t.TextMuted).Padding(t.Space(1), t.Space(2.5), t.Space(0.5))
				for _, n := range ns {
					s.network(c, n, n.SSID == *joined, &ev)
				}
			}
			list("Known Network", current)
			list("Known Networks", known)
			if len(other) > 0 {
				menuDivider(c)
				head := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2.5)).Radius(t.Space(1.5)).Role(ui.RoleDisclosure).Expanded(s.others).Label("Other Networks")
				if head.Hovered() {
					head.Background(t.SurfaceHover)
				}
				if head.Clicked() {
					s.others = !s.others
				}
				head.Children(func() {
					ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Other Networks").FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted).Grow(1)
						ui.Icon(c, chevron).Size(t.Space(3), t.Space(3)).TextColor(t.TextMuted).Rotate(head.AnimateTo("wifi.chevron", map[bool]float32{false: 0, true: 90}[s.others], ui.Snappy))
					})
				})
				if s.others {
					for _, n := range other {
						s.network(c, n, false, &ev)
					}
				}
			}
		}
		menuDivider(c)
		item := ui.ButtonBase(c).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Space(1.5)).Label("Wi-Fi Settings…")
		if item.Hovered() {
			item.Background(t.SurfaceHover)
		}
		ev.Settings = item.Clicked()
		item.Children(func() { ui.Text(c, "Wi-Fi Settings…").FillWidth() })
	})
	if ev.Join != nil || ev.Settings {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

func menuDivider(c *ui.Context) {
	t := c.Theme()
	ui.Divider(c).Margin(t.Space(1), t.Space(2.5))
}

// network creates the row of n, and its password field while it is
// asked.
func (s *State) network(c *ui.Context, n Network, joined bool, ev *Events) {
	t := c.Theme()
	asking := s.asking == n.SSID
	row := ui.ButtonBase(c.Key(n.SSID)).Padding(t.Space(1), t.Space(1.5)).Radius(t.Space(1.5)).Label(n.SSID)
	if row.Hovered() || asking {
		row.Background(t.SurfaceHover)
	}
	if row.Clicked() && !joined && s.joining == "" {
		switch {
		case n.Secure && !n.Known:
			s.asking, s.password, s.problem = n.SSID, "", ""
		default:
			join := n
			ev.Join, s.joining, s.asking = &join, n.SSID, ""
		}
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			d := t.Space(6.5)
			bg, fg := t.Text.Alpha(0.1), t.Text
			if joined {
				bg, fg = t.Accent, t.AccentText
			}
			ui.Box(c).Size(d, d).Shrink(0).Radius(d / 2).Background(bg).Draw(func(p *ui.Painter, r ui.Rect) {
				bars(p, r, n.Signal, fg)
			})
			ui.Text(c, n.SSID).Grow(1).Shrink(1).SingleLine().Ellipsis("…")
			switch {
			case s.joining == n.SSID:
				ui.Spinner(c).Size(t.Space(3.5), t.Space(3.5))
			case n.Secure:
				ui.Icon(c, lock).Size(t.Space(3.5), t.Space(3.5)).TextColor(t.TextMuted)
			}
		})
	})
	if !asking {
		return
	}
	ui.Column(c.Key(n.SSID+"/password")).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2.5), t.Space(2.5)).Children(func() {
		if s.problem != "" {
			ui.Text(c, s.problem).FontSize(t.FontSize - 1).TextColor(t.Danger)
		}
		field := ui.TextInput(c, &s.password).Password().Placeholder("Password").AutoFocus().Label("Password for " + n.SSID)
		ui.Row(c).Gap(t.Space(2)).Justify(ui.End).Children(func() {
			if ui.Button(c, "Cancel").Clicked() || field.Shortcut(0, ui.KeyEscape) {
				s.asking = ""
			}
			if ui.PrimaryButton(c, "Join").Disabled(s.password == "").Clicked() || field.Submitted() && s.password != "" {
				join := n
				ev.Join, ev.Password = &join, s.password
				s.joining, s.asking, s.password = n.SSID, "", ""
			}
		})
	})
}

// bars draws the Wi-Fi symbol in a circle r, its arcs lit as far as signal
// reaches: one up to a third, two up to two thirds, three past.
func bars(p *ui.Painter, r ui.Rect, signal float64, col ui.Color) {
	cx, cy := r.X+r.W/2, r.Y+r.H*0.68
	unit := r.W * 0.13
	lit := 1 + int(signal*3-0.0001)
	var dot ui.Path
	p.FillPath(dot.Circle(cx, cy, unit*0.55), col)
	for k := 1; k <= 3; k++ {
		rad := unit * (0.6 + 1.05*float32(k))
		var arc ui.Path
		for i := 0; i <= 12; i++ {
			a := math.Pi*1.25 + math.Pi*0.5*float64(i)/12
			x, y := cx+rad*float32(math.Cos(a)), cy+rad*float32(math.Sin(a))
			if i == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		c := col
		if k > lit {
			c = col.Alpha(0.3)
		}
		p.StrokePath(arc.Cap(ui.RoundCap), unit*0.62, c)
	}
}

var (
	lock    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M7 10V7a5 5 0 0 1 10 0v3" fill="none" stroke="currentColor" stroke-width="2"/><rect x="5" y="10" width="14" height="11" rx="2.5" fill="currentColor"/></svg>`))
	chevron = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M9 6l6 6-6 6" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
)

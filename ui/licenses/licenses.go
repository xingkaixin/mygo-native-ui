// Package licenses lists the open-source packages an app is built with
// and their licenses, as an About window's acknowledgements: a search, a
// chip for each license with how many packages use it, which keeps only
// those when clicked, and the packages with their versions and license
// tags, tinted by kind: permissive in green, weak copyleft in amber and
// strong copyleft in red. A click on a package, or the arrow keys in the
// list, shows its license's full text beside the list, with a link to
// the package's page.
//
//	licenses.View(c, &app.licenses, app.packages).Grow(1)
package licenses

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Package is a package the app uses: its name, version, the SPDX
// identifier of its license, as MIT or Apache-2.0, the URL of its page,
// and its license's text.
type Package struct {
	Name, Version string
	License       string
	URL           string
	Text          string
}

// Kind is how much a license asks of the app that uses the package.
type Kind uint8

const (
	// Permissive licenses ask for their notice to be kept.
	Permissive Kind = iota
	// WeakCopyleft licenses ask for changes to the package to be shared.
	WeakCopyleft
	// StrongCopyleft licenses ask for the whole app to be shared.
	StrongCopyleft
)

// KindOf returns the kind of the license with SPDX identifier id. The
// licenses it does not know are permissive.
func KindOf(id string) Kind {
	id = strings.ToUpper(id)
	switch {
	case strings.HasPrefix(id, "AGPL"), strings.HasPrefix(id, "GPL"), strings.HasPrefix(id, "SSPL"):
		return StrongCopyleft
	case strings.HasPrefix(id, "LGPL"), strings.HasPrefix(id, "MPL"), strings.HasPrefix(id, "EPL"),
		strings.HasPrefix(id, "CDDL"), strings.HasPrefix(id, "EUPL"):
		return WeakCopyleft
	}
	return Permissive
}

// State is what a list keeps: the search, the license kept and the
// package shown. Keep it in the app's state, one for each list.
type State struct {
	query   string
	license string
	// chosen is the name of the package shown.
	chosen string
}

// View creates the list of pkgs beside the license of the one chosen.
// Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, pkgs []Package) ui.Element {
	t := c.Theme()
	shown := filter(s, pkgs)
	at := slices.IndexFunc(shown, func(p Package) bool { return p.Name == s.chosen })
	if at < 0 && len(shown) > 0 {
		at, s.chosen = 0, shown[0].Name
	}
	return ui.Row(c).AlignItems(ui.Stretch).MinHeight(t.Space(80)).Border(1, t.Border).Radius(t.Radius + 2).Clip().Children(func() {
		ui.Column(c).Width(t.Space(82)).Background(t.Surface).Children(func() {
			ui.Column(c).Padding(t.Space(3)).Gap(t.Space(2.5)).Children(func() {
				ui.SearchField(c, &s.query).FillWidth().Label("Search packages")
				chips(c, s, pkgs)
			})
			ui.Divider(c)
			list := ui.Scroll(c).Basis(0).Grow(1).Focusable().Role(ui.RoleList).Label("Packages")
			if len(shown) > 0 {
				switch {
				case list.Shortcut(0, ui.KeyDown):
					at = min(at+1, len(shown)-1)
				case list.Shortcut(0, ui.KeyUp):
					at = max(at-1, 0)
				}
				s.chosen = shown[at].Name
			}
			list.Children(func() {
				ui.Column(c).Padding(t.Space(1.5)).Children(func() {
					for i, p := range shown {
						row(c.Key(p.Name), s, p, i == at, list.FocusWithin())
					}
					if len(shown) == 0 {
						ui.Text(c, "No packages match").TextColor(t.TextMuted).Padding(t.Space(3))
					}
				})
			})
			ui.Divider(c)
			ui.Text(c, count(len(shown), "package")).FontSize(t.FontSize-2).TextColor(t.TextMuted).
				Padding(t.Space(2), t.Space(3)).FontFeatures("tnum")
		})
		ui.Box(c).Width(1).FillHeight().Background(t.Border)
		ui.Column(c).Grow(1).Shrink(1).Background(t.Background).Children(func() {
			if at < 0 {
				return
			}
			detail(c, shown[at])
		})
	})
}

// filter returns the packages of pkgs that the search and the license
// kept, sorted by name.
func filter(s *State, pkgs []Package) []Package {
	words := strings.Fields(strings.ToLower(s.query))
	var out []Package
	for _, p := range pkgs {
		if s.license != "" && p.License != s.license {
			continue
		}
		text := strings.ToLower(p.Name + " " + p.License)
		if !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(text, w) }) {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Package) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out
}

// chips creates a chip for each license in pkgs, the most used first,
// with its count; a click keeps only its packages, or all again.
func chips(c *ui.Context, s *State, pkgs []Package) {
	t := c.Theme()
	counts := map[string]int{}
	var ids []string
	for _, p := range pkgs {
		if counts[p.License] == 0 {
			ids = append(ids, p.License)
		}
		counts[p.License]++
	}
	slices.SortStableFunc(ids, func(a, b string) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return strings.Compare(a, b)
	})
	ui.Row(c).Wrap().Gap(t.Space(1.5)).Children(func() {
		for _, id := range ids {
			on := s.license == id
			col := tint(t, KindOf(id))
			b := ui.ButtonBase(c.Key(id)).Row().Gap(t.Space(1.5)).AlignItems(ui.Center).
				Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(3)).Border(1, t.Border).Checked(on).
				Label(id + ", " + count(counts[id], "package"))
			switch {
			case on:
				b.Background(col.Alpha(0.16)).BorderColor(col.Alpha(0.6))
			case b.Hovered():
				b.Background(t.SurfaceHover)
			}
			if b.Clicked() {
				if on {
					s.license = ""
				} else {
					s.license = id
				}
			}
			b.Children(func() {
				ui.Box(c).Size(t.Space(1.5), t.Space(1.5)).Radius(t.Space(0.75)).Background(col)
				ui.Text(c, id).FontSize(t.FontSize - 2)
				ui.Text(c, strconv.Itoa(counts[id])).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum")
			})
		}
	})
}

// row creates p's row, lit when chosen, in the accent color while the
// list has the focus.
func row(c *ui.Context, s *State, p Package, chosen, focused bool) {
	t := c.Theme()
	b := ui.ButtonBase(c).Row().Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2)).
		Radius(t.Radius).FocusRing(false).Selected(chosen).Label(p.Name)
	name, muted := t.Text, t.TextMuted
	switch {
	case chosen && focused:
		b.Background(t.Accent)
		name, muted = t.AccentText, t.AccentText.Alpha(0.8)
	case chosen:
		b.Background(t.Text.Alpha(0.1))
	case b.Hovered():
		b.Background(t.Text.Alpha(0.05))
	}
	if chosen {
		b.ScrollIntoView()
	}
	if b.Clicked() {
		s.chosen = p.Name
	}
	b.Children(func() {
		ui.Text(c, p.Name).TextColor(name).SingleLine().Shrink(1)
		ui.Text(c, p.Version).FontSize(t.FontSize - 2).TextColor(muted).FontFeatures("tnum").SingleLine().Grow(1).Shrink(1)
		tag(c, p.License, chosen && focused)
	})
}

// tag creates the tag of license id, in the color of its kind, or in
// the accent's text color on a row lit in the accent.
func tag(c *ui.Context, id string, onAccent bool) {
	t := c.Theme()
	col := tint(t, KindOf(id))
	text, fill := col, col.Alpha(0.14)
	if onAccent {
		text, fill = t.AccentText, t.AccentText.Alpha(0.2)
	}
	ui.Text(c, id).FontSize(t.FontSize-3).FontWeight(600).TextColor(text).Background(fill).
		Padding(t.Space(0.25), t.Space(1.5)).Radius(t.Space(1)).SingleLine()
}

// detail creates p's name, version, license and link above its license's
// text.
func detail(c *ui.Context, p Package) {
	t := c.Theme()
	ui.Column(c).Padding(t.Space(4), t.Space(5)).Gap(t.Space(1.5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Baseline).Children(func() {
			ui.Text(c, p.Name).Bold().FontSize(t.FontSize + 3).Shrink(1).Role(ui.RoleHeading)
			ui.Text(c, p.Version).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			tag(c, p.License, false)
			ui.Text(c, kindName(KindOf(p.License))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			if p.URL != "" {
				ui.Spacer(c)
				ui.Link(c, strings.TrimPrefix(strings.TrimPrefix(p.URL, "https://"), "http://"), p.URL).FontSize(t.FontSize - 1)
			}
		})
	})
	ui.Divider(c)
	ui.Scroll(c).Basis(0).Grow(1).Children(func() {
		text := p.Text
		if text == "" {
			text = "No license text was given for " + p.Name + "."
		}
		ui.Text(c, text).Font("monospace").FontSize(t.FontSize-1).LineHeight(1.5).Selectable().
			Padding(t.Space(4), t.Space(5))
	})
}

func kindName(k Kind) string {
	switch k {
	case WeakCopyleft:
		return "Weak copyleft: changes to this package are shared"
	case StrongCopyleft:
		return "Strong copyleft: the whole app is shared under it"
	}
	return "Permissive: keep the notice below"
}

// tint returns the color of licenses of kind k.
func tint(t *ui.Theme, k Kind) ui.Color {
	l := float32(0.58)
	if t.Dark {
		l = 0.74
	}
	switch k {
	case WeakCopyleft:
		return ui.Oklch(l+0.08, 0.15, 70)
	case StrongCopyleft:
		return ui.Oklch(l, 0.18, 25)
	}
	return ui.Oklch(l, 0.13, 150)
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// Package keymap shows an app's keyboard shortcuts on a keyboard, as the
// keyboard maps of editors do: a US keyboard, its keys as wide as on a
// real one, each key showing the command it runs with the modifiers on,
// its cap in the color of the command's group, and a legend of the groups
// below. A click on a modifier key turns it on or off, and the modifiers
// held on the keyboard count too; with none on, the keys show what they
// run alone, as F5 does. A key that two commands share with the same
// modifiers is red and lists both. Hovering a key shows all its
// shortcuts, with any modifiers; a click on it chooses it, which Chosen
// returns.
//
//	keymap.View(c, &app.keymap, app.bindings)
//	if k, ok := app.keymap.Chosen(); ok {
//		... // list the bindings of k
//	}
//
// The modifiers show as package shortcut writes them, ⌃ ⌥ ⇧ ⌘ on macOS
// and Ctrl, Alt, Shift and Win, or Super on Linux, elsewhere, where the
// bottom row is a PC's.
package keymap

import (
	"cmp"
	"math/bits"
	"runtime"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/shortcut"
)

// Binding is a shortcut of the app: its keys as an accelerator, such as
// "CmdOrCtrl+Shift+K", as shortcut.Parse reads them, the command they
// run, and the group of the command, which colors its key. A binding
// whose keys do not parse shows nowhere.
type Binding struct {
	Keys    string
	Command string
	Group   string
}

// State is what a view keeps: the modifiers turned on and the key chosen.
// Keep it in the app's state, one for each view.
type State struct {
	mods   ui.Modifiers
	chosen ui.Key
}

// Chosen returns the key chosen, or false with none.
func (s *State) Chosen() (ui.Key, bool) { return s.chosen, s.chosen != ui.KeyUnknown }

// keycap is a key of the keyboard: one a shortcut can hold, a modifier, or
// another, as Caps Lock, by its name; where it is from the top left, and
// how wide, in widths of a letter key.
type keycap struct {
	key     ui.Key
	mod     ui.Modifiers
	name    string
	x, y, w float32
}

// width and height are the keyboard's size in widths of a letter key.
const width, height = 18.25, 6.25

// board is a US ANSI keyboard without a number pad, its bottom row as
// Apple's on macOS and as a PC's elsewhere.
var board = func() []keycap {
	mac := runtime.GOOS == "darwin"
	var out []keycap
	var x, y float32
	put := func(keys ...keycap) {
		for _, k := range keys {
			k.x, k.y = x, y
			out = append(out, k)
			x += k.w
		}
	}
	key := func(k ui.Key, w float32) keycap { return keycap{key: k, w: w} }
	mod := func(m ui.Modifiers, w float32) keycap { return keycap{mod: m, w: w} }
	chars := func(s string) {
		for _, r := range s {
			sc, _ := shortcut.Parse(string(r))
			put(key(sc.Key, 1))
		}
	}
	put(key(ui.KeyEscape, 1))
	for i := range 12 {
		x = 2 + float32(i) + float32(i/4)*0.5
		put(key(ui.KeyF1+ui.Key(i), 1))
	}
	x, y = 0, 1.25
	chars("`1234567890-=")
	put(key(ui.KeyBackspace, 2))
	x, y = 0, 2.25
	put(key(ui.KeyTab, 1.5))
	chars("QWERTYUIOP[]")
	put(key(ui.KeyBackslash, 1.5))
	x, y = 0, 3.25
	caps := "Caps Lock"
	if mac {
		caps = "⇪"
	}
	put(keycap{name: caps, w: 1.75})
	chars("ASDFGHJKL;'")
	put(key(ui.KeyEnter, 2.25))
	x, y = 0, 4.25
	put(mod(ui.Shift, 2.25))
	chars("ZXCVBNM,./")
	put(mod(ui.Shift, 2.75))
	x, y = 0, 5.25
	if mac {
		put(mod(ui.Ctrl, 1.5), mod(ui.Alt, 1.25), mod(ui.Super, 1.5), key(ui.KeySpace, 6.5),
			mod(ui.Super, 1.5), mod(ui.Alt, 1.25), mod(ui.Ctrl, 1.5))
	} else {
		put(mod(ui.Ctrl, 1.25), mod(ui.Super, 1.25), mod(ui.Alt, 1.25), key(ui.KeySpace, 6.25),
			mod(ui.Alt, 1.25), mod(ui.Super, 1.25), keycap{name: "Menu", w: 1.25}, mod(ui.Ctrl, 1.25))
	}
	x, y = 15.25, 1.25
	put(key(ui.KeyInsert, 1), key(ui.KeyHome, 1), key(ui.KeyPageUp, 1))
	x, y = 15.25, 2.25
	put(key(ui.KeyDelete, 1), key(ui.KeyEnd, 1), key(ui.KeyPageDown, 1))
	x, y = 16.25, 4.25
	put(key(ui.KeyUp, 1))
	x, y = 15.25, 5.25
	put(key(ui.KeyLeft, 1), key(ui.KeyDown, 1), key(ui.KeyRight, 1))
	return out
}()

// View creates the keyboard of bindings.
func View(c *ui.Context, s *State, bindings []Binding) ui.Element {
	t := c.Theme()
	keys := make([]shortcut.Shortcut, len(bindings))
	same := map[shortcut.Shortcut]int{}
	group := map[string]int{}
	var groups []string
	twice := false
	for i, b := range bindings {
		sc, err := shortcut.Parse(b.Keys)
		if err != nil {
			continue
		}
		keys[i] = sc
		same[sc]++
		twice = twice || same[sc] > 1
		if _, ok := group[b.Group]; !ok {
			group[b.Group] = len(groups)
			groups = append(groups, b.Group)
		}
	}
	mods := s.mods | c.Modifiers()
	u, gap, pad := t.Space(15), t.Space(1), t.Space(2)
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		plate := ui.Box(c).Size(width*u-gap+2*pad, height*u-gap+2*pad).Radius(t.Space(3)).
			Background(t.Surface).Border(1, t.Border)
		plate.Children(func() {
			for i, k := range board {
				var on, all []int
				for j, sc := range keys {
					if k.key != ui.KeyUnknown && sc.Key == k.key {
						all = append(all, j)
						if sc.Mods == mods {
							on = append(on, j)
						}
					}
				}
				var el ui.Element
				button := k.key != ui.KeyUnknown || k.mod != 0
				if button {
					el = ui.ButtonBase(c.Key(i))
				} else {
					el = ui.Box(c.Key(i))
				}
				el.Absolute().Left(pad+k.x*u).Top(pad+k.y*u).Size(k.w*u-gap, u-gap).Radius(t.Space(1.5)).Clip().
					Background(t.Background).Border(1, t.Border).Shadow(0, 1, 0, 0, t.Border)
				ink, label, lit := t.TextMuted, k.name, false
				switch {
				case k.mod != 0:
					label, ink = shortcut.Shortcut{Mods: k.mod}.MenuText(), t.Text
					active := mods&k.mod != 0
					el.Role(ui.RoleToggleButton).Checked(active).Label(label)
					if el.Clicked() {
						s.mods ^= k.mod
					}
					if active {
						el.Background(t.Accent).Border(1, t.Accent)
						ink, lit = t.AccentText, true
					}
				case k.key != ui.KeyUnknown:
					label = shortcut.Shortcut{Key: k.key}.MenuText()
					var commands []string
					for _, j := range on {
						commands = append(commands, bindings[j].Command)
					}
					el.Label(shortcut.Shortcut{Key: k.key}.String()).Description(strings.Join(commands, ", "))
					if el.Clicked() {
						s.chosen = k.key
					}
					switch {
					case len(on) > 1:
						el.Background(t.Danger.Alpha(0.2).Over(t.Background)).Border(1, t.Danger)
						ink, lit = t.Text, true
					case len(on) == 1:
						fill, edge := tint(group[bindings[on[0]].Group], t)
						el.Background(fill).Border(1, edge)
						ink, lit = t.Text, true
					}
				}
				if button && el.Hovered() {
					if !lit {
						el.Border(1, t.Text.Alpha(0.5))
					}
					if len(all) > 0 {
						card(c, el, bindings, keys, same, group, all)
					}
				}
				if k.key != ui.KeyUnknown && k.key == s.chosen {
					// A ring over the border, as a wider border would move
					// the text.
					el.DrawOver(func(p *ui.Painter, r ui.Rect) { p.Stroke(r, t.Accent, t.Space(1.5), 2) })
				}
				el.Children(func() {
					ui.Column(c).Fill().Padding(t.Space(1)).Gap(t.Space(0.5)).Children(func() {
						ui.Text(c, label).FontSize(t.FontSize - 2).SingleLine().TextColor(ink)
						lines := 2
						if len(on) > 1 {
							lines = 1
						}
						for _, j := range on {
							ui.Text(c.Key(j), bindings[j].Command).FontSize(t.FontSize - 4).LineHeight(1.15).
								MaxLines(lines).TextColor(ink)
						}
					})
				})
			}
		})
		ui.Row(c).Gap(t.Space(4)).Wrap().AlignItems(ui.Center).Children(func() {
			for i, g := range groups {
				fill, edge := tint(i, t)
				swatch(c.Key(i), fill, edge, g)
			}
			if twice {
				swatch(c.Key("twice"), t.Danger.Alpha(0.2).Over(t.Background), t.Danger, "Bound twice")
			}
		})
	})
}

// card shows by el the bindings all of its key, with any modifiers, in
// the order of their modifiers, those bound twice in red.
func card(c *ui.Context, el ui.Element, bindings []Binding, keys []shortcut.Shortcut, same map[shortcut.Shortcut]int, group map[string]int, all []int) {
	t := c.Theme()
	slices.SortStableFunc(all, func(a, b int) int {
		ma, mb := keys[a].Mods, keys[b].Mods
		return cmp.Or(cmp.Compare(bits.OnesCount8(uint8(ma)), bits.OnesCount8(uint8(mb))), cmp.Compare(ma, mb))
	})
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the keys
		// under it, which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).
			AttachTo(el, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1.5), 0).
			Padding(t.Space(2), t.Space(3)).Gap(t.Space(1.5)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			for _, j := range all {
				b := bindings[j]
				ui.Row(c.Key(j)).PassThrough().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					fill, edge := tint(group[b.Group], t)
					ink := t.Text
					if same[keys[j]] > 1 {
						ink = t.Danger
					}
					ui.Box(c).PassThrough().Size(t.Space(2.5), t.Space(2.5)).Radius(2).Background(fill).Border(1, edge)
					ui.Text(c, b.Command).PassThrough().Grow(1).SingleLine().TextColor(ink)
					ui.Text(c, keys[j].MenuText()).PassThrough().MarginStart(t.Space(4)).TextColor(t.TextMuted)
				})
			}
		})
	})
}

// swatch shows a legend's swatch of fill and edge, and text beside it.
func swatch(c *ui.Context, fill, edge ui.Color, text string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(t.Space(3.5), t.Space(3.5)).Radius(t.Space(1)).Background(fill).Border(1, edge)
		ui.Text(c, text).FontSize(t.FontSize - 1)
	})
}

// hues are the hues of the groups, in order, away from the red of keys
// bound twice.
var hues = []float32{255, 150, 85, 305, 195, 345, 120, 225}

// tint returns the face and the edge of a key of group k.
func tint(k int, t *ui.Theme) (fill, edge ui.Color) {
	h := ui.Oklch(0.65, 0.13, hues[k%len(hues)])
	return h.Alpha(0.22).Over(t.Background), h.Alpha(0.8).Over(t.Background)
}

// Package clipboard lists what was copied, as clipboard managers do: a
// search over the clips, pinned clips first and then the rest newest
// first, each with its kind drawn, a color as its swatch and an image as
// a thumbnail, the app it came from and how long ago, and ⌘1 to ⌘9 by the
// first nine. Beside the list, the chosen clip in full: a text with its
// counts, a link with its host, a color with its values, an image at its
// size. While the search has the focus, the arrows move through the
// list; Enter, ⌘1 to ⌘9 or a double click pastes, ⌘P pins and unpins, and
// ⌘⌫ deletes, as the clip's context menu and the buttons below it do.
//
//	clipboard.Add(&app.clips, clipboard.Clip{Text: text, Source: "Safari", Copied: time.Now()}, 200)
//	ev := clipboard.View(c, &app.history, &app.clips)
//	if ev.Pasted != nil {
//		paste(*ev.Pasted)
//	}
package clipboard

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Kind is what a clip holds.
type Kind uint8

const (
	Text Kind = iota
	Link
	Color
	Image
)

var kinds = [...]string{"Text", "Link", "Color", "Image"}

func (k Kind) String() string { return kinds[k] }

// Clip is something copied: its text, or its image, the app it was copied
// from, when, and whether it is pinned.
type Clip struct {
	Text   string
	Image  *ui.Bitmap
	Source string
	Copied time.Time
	Pinned bool
}

var (
	hexColor = regexp.MustCompile(`^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	rgbColor = regexp.MustCompile(`^rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(,\s*[\d.]+\s*)?\)$`)
)

// Kind returns what c holds: an image, a color written as CSS writes one,
// as "#0a84ff" or "rgb(10, 132, 255)", a web address alone, or text.
func (c Clip) Kind() Kind {
	s := strings.TrimSpace(c.Text)
	switch {
	case c.Image != nil:
		return Image
	case hexColor.MatchString(s) || rgbColor.MatchString(s):
		return Color
	}
	if u, err := url.Parse(s); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && !strings.ContainsAny(s, " \n") {
		return Link
	}
	return Text
}

func (c Clip) color() ui.Color {
	s := strings.TrimSpace(c.Text)
	if m := rgbColor.FindStringSubmatch(s); m != nil {
		var v [3]uint8
		for i := range v {
			n, _ := strconv.Atoi(m[i+1])
			v[i] = uint8(min(255, n))
		}
		return ui.RGB(v[0], v[1], v[2])
	}
	return ui.Hex(s)
}

func (c Clip) title() string {
	if c.Image != nil {
		w, h := c.Image.Size()
		return fmt.Sprintf("Image %d × %d", w, h)
	}
	for l := range strings.SplitSeq(c.Text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return "Blank text"
}

// Add puts clip first in *clips, or moves the clip already there with the
// same text or image first, pinned if it was; past limit clips, the
// oldest that are not pinned go.
func Add(clips *[]Clip, clip Clip, limit int) {
	list := slices.Clone(*clips)
	if i := slices.IndexFunc(list, func(o Clip) bool { return o.Text == clip.Text && o.Image == clip.Image }); i >= 0 {
		clip.Pinned = list[i].Pinned
		list = slices.Delete(list, i, i+1)
	}
	list = slices.Insert(list, 0, clip)
	for i := len(list) - 1; i >= 0 && len(list) > limit; i-- {
		if !list[i].Pinned {
			list = slices.Delete(list, i, i+1)
		}
	}
	*clips = list
}

// State is what a list keeps: its search, and the clip chosen. Keep it in
// the app's state, one for each list.
type State struct {
	query    string
	searched string
	selected int
}

// Events is a list, and the clip pasted from it in the frame, nil when
// none was. The app pastes it, and adds it again as the system clipboard
// takes it, which brings it first.
type Events struct {
	ui.Element
	Pasted *Clip
}

// order returns where the clips matching the search are in clips, pinned
// first.
func (s *State) order(clips []Clip) []int {
	q := strings.ToLower(strings.TrimSpace(s.query))
	var pinned, rest []int
	for i, clip := range clips {
		if q != "" && !strings.Contains(strings.ToLower(clip.Text), q) && !strings.Contains(strings.ToLower(clip.Source), q) {
			continue
		}
		if clip.Pinned {
			pinned = append(pinned, i)
		} else {
			rest = append(rest, i)
		}
	}
	return append(pinned, rest...)
}

type action uint8

const (
	none action = iota
	paste
	pin
	remove
)

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	textIcon = icon(`<path d="M5 6h14M5 10h14M5 14h10M5 18h7"/>`)
	linkIcon = icon(`<path d="M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1"/><path d="M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1"/>`)
	pinIcon  = icon(`<path d="M9 3h6l-1 6 3 3v2H7v-2l3-3z"/><path d="M12 14v7"/>`)
)

// View creates the list of *clips and the clip chosen in it.
func View(c *ui.Context, s *State, clips *[]Clip) Events {
	t := c.Theme()
	var ev Events
	order := s.order(*clips)
	if s.query != s.searched {
		s.selected, s.searched = 0, s.query
	}
	s.selected = max(0, min(s.selected, len(order)-1))
	root := ui.Row(c).Fill().Background(t.Background)
	moved := false
	if root.Shortcut(0, ui.KeyDown) {
		s.selected, moved = min(s.selected+1, len(order)-1), true
	}
	if root.Shortcut(0, ui.KeyUp) {
		s.selected, moved = max(s.selected-1, 0), true
	}
	do, at := none, -1
	if len(order) > 0 {
		switch {
		case root.Shortcut(0, ui.KeyEnter):
			do, at = paste, order[s.selected]
		case root.Shortcut(ui.Cmd, ui.KeyP):
			do, at = pin, order[s.selected]
		case root.Shortcut(ui.Cmd, ui.KeyBackspace):
			do, at = remove, order[s.selected]
		}
	}
	for k := range min(9, len(order)) {
		if root.Shortcut(ui.Cmd, ui.Key1+ui.Key(k)) {
			do, at = paste, order[k]
		}
	}
	now := c.Now()
	c.After(time.Minute - now.Sub(now.Truncate(time.Minute)))
	root.Children(func() {
		ui.Column(c).Width(t.Space(84)).FillHeight().Children(func() {
			ui.SearchField(c, &s.query).AutoFocus().Margin(t.Space(3))
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Padding(0, t.Space(2), t.Space(2), t.Space(2)).Children(func() {
					if len(order) == 0 {
						msg := "Nothing copied yet"
						if s.query != "" {
							msg = "No clips match"
						}
						ui.Text(c, msg).TextColor(t.TextMuted).Padding(t.Space(4)).AlignSelf(ui.Center)
					}
					for k, i := range order {
						clip := (*clips)[i]
						if k == 0 && clip.Pinned || k > 0 && !clip.Pinned && (*clips)[order[k-1]].Pinned {
							head := "Recent"
							if clip.Pinned {
								head = "Pinned"
							}
							ui.Text(c.Key(head), head).FontSize(t.FontSize-2).Bold().TextColor(t.TextMuted).Padding(t.Space(2), t.Space(2), t.Space(1))
						}
						row := s.row(c.Key(i), clip, k, k == s.selected, now)
						if moved && k == s.selected {
							row.ScrollIntoView()
						}
						if row.Clicked() {
							s.selected = k
						}
						if row.DoubleClicked() {
							do, at = paste, i
						}
						row.ContextMenu(func(m *ui.Menu) {
							if m.Item("Paste").Chosen() {
								do, at = paste, i
							}
							label := "Pin"
							if clip.Pinned {
								label = "Unpin"
							}
							if m.Item(label).Shortcut(ui.Cmd, ui.KeyP).Chosen() {
								do, at = pin, i
							}
							m.Separator()
							if m.Item("Delete").Shortcut(ui.Cmd, ui.KeyBackspace).Chosen() {
								do, at = remove, i
							}
						})
					}
				})
			})
		})
		ui.Divider(c).Vertical()
		if len(order) > 0 {
			i := order[s.selected]
			if a := preview(c, (*clips)[i], now); a != none {
				do, at = a, i
			}
		} else {
			ui.Box(c).Grow(1)
		}
	})
	switch do {
	case paste:
		clip := (*clips)[at]
		ev.Pasted = &clip
	case pin:
		list := slices.Clone(*clips)
		list[at].Pinned = !list[at].Pinned
		*clips = list
	case remove:
		*clips = slices.Delete(slices.Clone(*clips), at, at+1)
	}
	if do != none {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// row creates the row of clip, the k-th shown.
func (s *State) row(c *ui.Context, clip Clip, k int, selected bool, now time.Time) ui.Element {
	t := c.Theme()
	row := ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).
		Role(ui.RoleListItem).Label(clip.title()).Selected(selected)
	text, muted := t.Text, t.TextMuted
	switch {
	case selected:
		row.Background(t.Accent)
		text, muted = t.AccentText, t.AccentText.Alpha(0.75)
	case row.Hovered():
		row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		thumb(c, clip, t.Space(8))
		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			ui.Text(c, clip.title()).TextColor(text).SingleLine().Ellipsis("…")
			sub := ago(now, clip.Copied)
			if clip.Source != "" {
				sub = clip.Source + " · " + sub
			}
			ui.Text(c, sub).FontSize(t.FontSize - 2).TextColor(muted).SingleLine().Ellipsis("…")
		})
		if clip.Pinned {
			ui.Icon(c, pinIcon).Size(t.Space(3.5), t.Space(3.5)).TextColor(muted)
		}
		if k < 9 {
			ui.Text(c, "⌘"+strconv.Itoa(k+1)).FontSize(t.FontSize - 2).TextColor(muted).FontFeatures("tnum")
		}
	})
	return row
}

// thumb creates clip as a square of side d: its image, its color, or an
// icon of its kind.
func thumb(c *ui.Context, clip Clip, d float32) {
	t := c.Theme()
	box := ui.Box(c).Size(d, d).Shrink(0).Radius(t.Space(1.5)).Clip().Center().Border(1, t.Border)
	switch clip.Kind() {
	case Image:
		box.Children(func() { ui.Image(c, clip.Image).Size(d, d).Fit(ui.Cover) })
	case Color:
		box.Background(clip.color())
	default:
		ic := textIcon
		if clip.Kind() == Link {
			ic = linkIcon
		}
		box.Background(t.Surface).Children(func() {
			ui.Icon(c, ic).Size(d/2, d/2).TextColor(t.TextMuted)
		})
	}
}

// preview creates clip in full, and returns what its buttons ask.
func preview(c *ui.Context, clip Clip, now time.Time) action {
	t := c.Theme()
	do := none
	ui.Column(c).Grow(1).Basis(0).FillHeight().Children(func() {
		ui.Column(c).Grow(1).Basis(0).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
			kind := clip.Kind()
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, kind.String()).Bold()
				head := ago(now, clip.Copied)
				if clip.Source != "" {
					head = "From " + clip.Source + ", " + strings.ToLower(head[:1]) + head[1:]
				}
				ui.Text(c, head).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
			})
			switch kind {
			case Image:
				w, h := clip.Image.Size()
				ui.Image(c, clip.Image).Grow(1).Basis(0).FillWidth().Fit(ui.Contain)
				ui.Text(c, fmt.Sprintf("%d × %d pixels", w, h)).TextColor(t.TextMuted).FontFeatures("tnum")
			case Color:
				col := clip.color()
				ui.Box(c).FillWidth().Height(t.Space(36)).Radius(t.Radius*2).Background(col).Border(1, t.Border)
				h, sat, l := hsl(col)
				ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(t.Space(6)).GapY(t.Space(2)).Children(func() {
					for _, v := range [][2]string{
						{"HEX", fmt.Sprintf("#%02X%02X%02X", col.R, col.G, col.B)},
						{"RGB", fmt.Sprintf("rgb(%d, %d, %d)", col.R, col.G, col.B)},
						{"HSL", fmt.Sprintf("hsl(%.0f, %.0f%%, %.0f%%)", h, sat*100, l*100)},
					} {
						ui.Text(c, v[0]).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
						ui.Text(c, v[1]).Font("monospace").Selectable()
					}
				})
			case Link:
				u, _ := url.Parse(strings.TrimSpace(clip.Text))
				ui.Text(c, strings.TrimPrefix(u.Host, "www.")).FontSize(t.FontSize + 6).Bold()
				ui.Text(c, clip.Text).TextColor(t.TextMuted).Selectable().MaxLines(4)
				if ui.Button(c, "Open in Browser").AlignSelf(ui.Start).Clicked() {
					c.OpenURL(strings.TrimSpace(clip.Text))
				}
			default:
				ui.Scroll(c).Grow(1).Basis(0).Children(func() {
					ui.Text(c, clip.Text).Selectable().Font("monospace").FontSize(t.FontSize - 1)
				})
				lines := strings.Count(strings.TrimRight(clip.Text, "\n"), "\n") + 1
				ui.Text(c, fmt.Sprintf("%s · %s · %s", count(utf8.RuneCountInString(clip.Text), "character"),
					count(len(strings.Fields(clip.Text)), "word"), count(lines, "line"))).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
			}
		})
		ui.Divider(c)
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3), t.Space(5)).Children(func() {
			if ui.PrimaryButton(c, "Paste").Clicked() {
				do = paste
			}
			label := "Pin"
			if clip.Pinned {
				label = "Unpin"
			}
			if ui.Button(c, label).Clicked() {
				do = pin
			}
			ui.Spacer(c)
			if ui.Button(c, "Delete").Clicked() {
				do = remove
			}
		})
	})
	return do
}

func count(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

func ago(now, then time.Time) string {
	switch d := now.Sub(then); {
	case d < time.Minute:
		return "Just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hr ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "Yesterday"
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

// hsl returns c's hue in degrees, and its saturation and lightness from 0
// to 1.
func hsl(c ui.Color) (h, s, l float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	hi, lo := max(r, g, b), min(r, g, b)
	l = (hi + lo) / 2
	d := hi - lo
	if d == 0 {
		return 0, 0, l
	}
	s = d / (1 - math.Abs(2*l-1))
	switch hi {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, l
}

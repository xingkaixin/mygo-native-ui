// Package sources shows where a model's reply comes from, as AI search
// apps show it: below the reply, the sources' sites and "Used 5 sources",
// which a click opens to a numbered list of them; and by a sentence, a
// citation, a chip of the site of its first source and how many more,
// "go.dev +2", which shows a card of them while the pointer rests on it.
// A click on a source opens it in the browser.
//
//	ui.Row(c).Wrap().Children(func() {
//		markdown.View(c, paragraph, nil)
//		sources.Citation(c, reply.sources[:2])
//	})
//	sources.List(c, &reply.sourcesOpen, reply.sources)
package sources

import (
	"hash/fnv"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Source is a page the model read: its title, its address, and a line of
// what it says, which a citation's card shows under the title.
type Source struct {
	Title   string
	URL     string
	Snippet string
}

// Site returns the host of the source's address without "www.", as
// "go.dev", or the address when it has none.
func (s Source) Site() string {
	u, err := url.Parse(s.URL)
	if err != nil || u.Host == "" {
		return s.URL
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

// List creates the row "Used 5 sources", with the first sites' badges,
// and below it, while *open is true, which a click on the row toggles,
// the sources numbered from 1, as citations number them.
func List(c *ui.Context, open *bool, sources []Source) ui.Element {
	t := c.Theme()
	return ui.Column(c).Gap(t.Space(1)).Children(func() {
		label := "Used " + strconv.Itoa(len(sources)) + " sources"
		if len(sources) == 1 {
			label = "Used 1 source"
		}
		head := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(2)).Padding(t.Space(1), t.Space(2), t.Space(1), t.Space(1)).
			Radius(t.Space(4)).AlignSelf(ui.Start).Expanded(*open).Label(label)
		if head.Clicked() {
			*open = !*open
		}
		if head.Hovered() {
			head.Background(t.Text.Alpha(0.06))
		}
		head.Children(func() {
			ui.Row(c).Children(func() {
				for i, s := range sources[:min(3, len(sources))] {
					b := badge(c.Key(i), s.Site())
					if i > 0 {
						b.MarginStart(-t.Space(1.5))
					}
				}
			})
			ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			chevron(c, *open)
		})
		if !*open {
			return
		}
		ui.Column(c).Padding(0, 0, t.Space(1)).Children(func() {
			for i, s := range sources {
				row := link(c.Key(i), s).Gap(t.Space(2.5)).Padding(t.Space(1.5), t.Space(2))
				row.Children(func() {
					ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).
						FontFeatures("tnum").Width(t.Space(4)).TextAlign(ui.End)
					badge(c, s.Site())
					ui.Text(c, s.Title).SingleLine().Shrink(1)
					ui.Text(c, s.Site()).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Shrink(2)
				})
			}
		})
	})
}

// card is what a citation keeps: whether its card shows, held there by a
// click or the keyboard rather than the pointer, whether the pointer was
// over the card in the last frame, and since when the pointer has been
// over the chip or the card, or away from both.
type card struct {
	open, held, over bool
	since            time.Time
	hovering         bool
}

// delay is how long the pointer rests on a citation before its card
// shows, and is away from both before the card goes.
const delay = 300 * time.Millisecond

// Citation creates a chip citing sources: the site of the first and how
// many more, "go.dev +2". Its card of the sources, each with its site,
// title and snippet, shows once the pointer has rested on the chip, and
// stays while the pointer is over either; a click on the chip, Enter or
// Space hold it until Escape or a click outside.
func Citation(c *ui.Context, sources []Source) ui.Element {
	t := c.Theme()
	if len(sources) == 0 {
		return ui.Box(c)
	}
	text := sources[0].Site()
	label := "Source: " + text
	if n := len(sources) - 1; n > 0 {
		text += " +" + strconv.Itoa(n)
		label = "Sources: " + sources[0].Site() + " and " + strconv.Itoa(n) + " more"
	}
	chip := ui.ButtonBase(c).Height(t.Space(5)).Padding(0, t.Space(2)).Radius(t.Space(2.5)).Label(label).
		Background(t.Text.Alpha(0.06)).MarginStart(t.Space(1))
	st := ui.Local(chip, "sources.card", func() card { return card{} })
	if chip.Clicked() {
		st.open, st.held = true, true
	}
	if !st.held {
		hover(c, st, chip.Hovered() || (st.open && st.over))
	}
	if st.open || chip.Hovered() {
		chip.Background(t.Text.Alpha(0.12))
	}
	chip.Children(func() {
		ui.Text(c, text).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
	})
	st.over = false
	panel := ui.PopoverBase(c, chip, &st.open, func(panel ui.Element) {
		st.over = panel.Hovered()
		panel.Width(t.Space(80)).Padding(t.Space(1)).Radius(t.Space(2.5)).Background(t.Background).
			Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGB(0, 0, 0).Alpha(0.12))
		panel.Children(func() {
			for i, s := range sources {
				ui.Column(c.Key(i)).Children(func() {
					if i > 0 {
						ui.Divider(c).Margin(t.Space(1), t.Space(2))
					}
					entry(c, s)
				})
			}
		})
	})
	if panel.Valid() && panel.Dismissed() {
		st.held = false
	}
	return chip
}

// entry shows s in a citation's card: its site, its title and its
// snippet.
func entry(c *ui.Context, s Source) {
	t := c.Theme()
	link(c, s).Column().AlignItems(ui.Stretch).Gap(t.Space(1)).Padding(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			badge(c, s.Site())
			ui.Text(c, s.Site()).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine().Shrink(1)
		})
		ui.Text(c, s.Title).Bold().MaxLines(2)
		if s.Snippet != "" {
			ui.Text(c, s.Snippet).FontSize(t.FontSize - 1).TextColor(t.TextMuted).MaxLines(3)
		}
	})
}

// hover opens and closes a card as the pointer comes and goes, on is
// whether it is over the chip or the card, once it has been for delay.
func hover(c *ui.Context, st *card, on bool) {
	now := c.Now()
	if on != st.hovering {
		st.hovering, st.since = on, now
	}
	if st.open == on {
		return
	}
	if left := delay - now.Sub(st.since); left > 0 {
		c.After(left)
		return
	}
	st.open = on
}

// link creates a button opening s in the browser, lit as the pointer is
// over it.
func link(c *ui.Context, s Source) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Justify(ui.Start).Radius(t.Space(1.5)).Label(s.Title).Description(s.URL).Role(ui.RoleLink)
	if b.Clicked() {
		c.OpenURL(s.URL)
	}
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.06))
	}
	return b
}

// badge shows the first letter of site in a circle of a color of its
// own, as a site's icon.
func badge(c *ui.Context, site string) ui.Element {
	t := c.Theme()
	h := fnv.New32a()
	h.Write([]byte(site))
	color := ui.Oklch(0.62, 0.13, float32(h.Sum32()%360))
	r, _ := utf8.DecodeRuneInString(site)
	letter := strings.ToUpper(string(r))
	d := t.Space(4.5)
	return ui.Row(c).Size(d, d).Radius(d/2).Background(color).Border(1.5, t.Background).
		Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, letter).FontSize(d * 0.55).Bold().TextColor(ui.RGB(255, 255, 255))
	})
}

// chevron shows an arrow pointing down, or up when open.
func chevron(c *ui.Context, open bool) {
	t := c.Theme()
	ui.Box(c).Size(t.Space(3), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
		points := [][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}
		if open {
			points = [][2]float32{{cx - k, cy + k/2}, {cx, cy - k/2}, {cx + k, cy + k/2}}
		}
		p.StrokePolyline(points, 1.5, t.TextMuted)
	})
}

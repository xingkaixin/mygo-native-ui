// Package pagination moves through the pages of a long list, as tables
// and search results do below them. View shows the pages by number
// between Previous and Next: all of them when they are few, or the first,
// the last and those around the current one, with ellipses for the
// rest, always as many buttons so that they do not move under the
// pointer. Footer shows the rows a page has, which rows show of how many,
// and arrows, as tables of data do. The left and right arrows turn the
// page while either has the focus.
//
//	if pagination.View(c, &app.page, pages).Changed() {
//		app.load(app.page)
//	}
//	pagination.Footer(c, &app.page, &app.size, len(app.rows), 10, 25, 50)
package pagination

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// slots is how many numbers and ellipses View shows once there are more
// pages than fit.
const slots = 7

// View creates the pages, *page, from 0, the one showing, of pages.
// Changed reports a new page.
func View(c *ui.Context, page *int, pages int) ui.Element {
	t := c.Theme()
	*page = max(0, min(*page, pages-1))
	before := *page
	el := ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Role(ui.RoleGroup).Label("Pages")
	el.Children(func() {
		if arrow(c, "‹", "Previous page", *page == 0).Clicked() {
			*page--
		}
		for k, p := range numbers(*page, pages) {
			if p < 0 {
				ui.Text(c.Key(k), "…").TextColor(t.TextMuted).Width(t.Space(8)).TextAlign(ui.Center)
				continue
			}
			if number(c.Key(k), p, p == *page).Clicked() {
				*page = p
			}
		}
		if arrow(c, "›", "Next page", *page >= pages-1).Clicked() {
			*page++
		}
	})
	keys(el, page, pages)
	if *page != before {
		el.SetChanged()
		c.Announce("Page " + strconv.Itoa(*page+1) + " of " + strconv.Itoa(pages))
	}
	return el
}

// numbers returns the pages View shows a button for around page, from 0,
// of pages, -1 for an ellipsis.
func numbers(page, pages int) []int {
	var out []int
	add := func(from, to int) {
		for p := from; p <= to; p++ {
			out = append(out, p)
		}
	}
	switch {
	case pages <= slots:
		add(0, pages-1)
	case page < slots-3:
		add(0, slots-3)
		out = append(out, -1, pages-1)
	case page > pages-slots+2:
		out = append(out, 0, -1)
		add(pages-slots+2, pages-1)
	default:
		out = append(out, 0, -1)
		add(page-1, page+1)
		out = append(out, -1, pages-1)
	}
	return out
}

// number creates the button of page p, from 0, lit when it is current.
func number(c *ui.Context, p int, current bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).MinWidth(t.Space(8)).Height(t.Space(8)).Padding(0, t.Space(2)).Radius(t.Radius).Center().
		Label("Page " + strconv.Itoa(p+1))
	color := t.Text
	switch {
	case current:
		b.Background(t.Accent).Selected(true)
		color = t.AccentText
	case b.Pressed():
		b.Background(t.SurfacePressed)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	return b.Children(func() {
		ui.Text(c, strconv.Itoa(p+1)).TextColor(color).FontFeatures("tnum")
	})
}

// arrow creates the button showing text, named label, dimmed when off.
func arrow(c *ui.Context, text, label string, off bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(8), t.Space(8)).Radius(t.Radius).Center().Label(label).Disabled(off)
	color := t.Text
	switch {
	case off:
		color = t.TextMuted.Alpha(0.5)
	case b.Pressed():
		b.Background(t.SurfacePressed)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	return b.Children(func() { ui.Text(c, text).FontSize(t.FontSize + 4).TextColor(color) })
}

// keys turns *page of pages with the arrow keys while el has the focus.
func keys(el ui.Element, page *int, pages int) {
	el.Focusable()
	if el.Shortcut(0, ui.KeyLeft) && *page > 0 {
		*page--
	}
	if el.Shortcut(0, ui.KeyRight) && *page < pages-1 {
		*page++
	}
}

// Footer creates the footer of a table of total rows, *size a page, one
// of sizes, showing page *page, from 0. Changed reports a new page or
// size; a new size keeps the first row showing in view.
func Footer(c *ui.Context, page, size *int, total int, sizes ...int) ui.Element {
	t := c.Theme()
	if *size <= 0 {
		*size = 10
		if len(sizes) > 0 {
			*size = sizes[0]
		}
	}
	pages := max(1, (total+*size-1) / *size)
	*page = max(0, min(*page, pages-1))
	before, was := *page, *size
	el := ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Justify(ui.End).Role(ui.RoleGroup).Label("Pages")
	el.Children(func() {
		if len(sizes) > 0 {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Rows per page").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				names := make([]string, len(sizes))
				for i, n := range sizes {
					names[i] = strconv.Itoa(n)
				}
				chosen := strconv.Itoa(*size)
				if ui.Select(c, &chosen, names).Label("Rows per page").Changed() {
					first := *page * *size
					*size, _ = strconv.Atoi(chosen)
					*page = first / *size
				}
			})
		}
		first, last := min(*page**size+1, total), min((*page+1)**size, total)
		ui.Text(c, group(first)+"–"+group(last)+" of "+group(total)).FontSize(t.FontSize - 1).FontFeatures("tnum")
		ui.Row(c).Children(func() {
			if arrow(c, "‹", "Previous page", *page == 0).Clicked() {
				*page--
			}
			if arrow(c, "›", "Next page", *page >= pages-1).Clicked() {
				*page++
			}
		})
	})
	keys(el, page, pages)
	if *page != before || *size != was {
		el.SetChanged()
	}
	return el
}

// group writes n with commas: 12,400.
func group(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

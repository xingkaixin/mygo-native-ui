// Package notifications shows an app's notifications as an inbox, as the
// bell menus of web apps and the notification centers of desktops do: a
// header with how many are unread, a choice of All or Unread and "Mark all
// as read"; then the notifications newest first under Today, Yesterday
// and Earlier, each with the avatar of who or what sent it, its title, its
// text cut to two lines, how long ago it came, and a dot of the accent
// while unread. The pointer on one shows buttons that mark it read or
// unread and dismiss it; a click opens it, marking it read; and its
// actions, as Reply or Approve, show as buttons under it.
//
//	ev := notifications.View(c, &app.inbox, &app.items)
//	if i, ok := ev.Opened(); ok {
//		app.open(app.items[i])
//	}
//	if i, action, ok := ev.Action(); ok {
//		app.do(app.items[i], action)
//	}
package notifications

import (
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Item is a notification: who or what sent it, which its avatar shows, its
// title and text, when it came, whether it was read, and the actions it
// offers.
type Item struct {
	Source  string
	Title   string
	Body    string
	Time    time.Time
	Read    bool
	Actions []string
}

// State is what an inbox keeps: whether it shows the unread only. Keep it
// in the app's state, one for each inbox.
type State struct {
	unread int
	list   ui.ListState
	rows   []int
}

// Events is what the user did in an inbox in a frame.
type Events struct {
	opened, acted int
	action        string
}

// Opened returns the notification the user clicked, which is now read.
func (e Events) Opened() (int, bool) { return e.opened - 1, e.opened > 0 }

// Action returns the notification whose action the user clicked, and the
// action.
func (e Events) Action() (int, string, bool) { return e.acted - 1, e.action, e.acted > 0 }

// View creates the inbox of *items, which it marks read and unread and
// dismisses from in place. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, items *[]Item) Events {
	t := c.Theme()
	now := c.Now()
	c.After(time.Minute - time.Duration(now.Second())*time.Second)
	var ev Events
	unread := 0
	for _, it := range *items {
		if !it.Read {
			unread++
		}
	}
	order := make([]int, 0, len(*items))
	for i, it := range *items {
		if s.unread == 0 || !it.Read {
			order = append(order, i)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int { return (*items)[b].Time.Compare((*items)[a].Time) })
	// rows holds a heading as -1 minus its group, and an item as its
	// index.
	s.rows = s.rows[:0]
	last := -1
	for _, i := range order {
		if g := group((*items)[i].Time, now); g != last {
			s.rows = append(s.rows, -1-g)
			last = g
		}
		s.rows = append(s.rows, i)
	}
	s.list.Header = func(r int) bool { return s.rows[r] < 0 }
	s.list.Key = func(r int) any { return s.rows[r] }
	dismiss := -1
	ui.Column(c).Grow(1).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Notifications").Bold().FontSize(t.FontSize + 2)
			if unread > 0 {
				ui.Badge(c, strconv.Itoa(unread)).Label(strconv.Itoa(unread) + " unread")
			}
			ui.Spacer(c)
			if ui.Button(c, "Mark all as read").Disabled(unread == 0).Clicked() {
				for i := range *items {
					(*items)[i].Read = true
				}
			}
		})
		ui.Row(c).Padding(0, t.Space(4), t.Space(2)).Children(func() {
			ui.Segmented(c, &s.unread, "All", "Unread").Label("Show")
		})
		ui.Divider(c)
		list := ui.List(c, &s.list, len(s.rows), func(r int) {
			i := s.rows[r]
			if i < 0 {
				ui.Text(c, groups[-1-i]).FontSize(t.FontSize-1).Bold().TextColor(t.TextMuted).
					Background(t.Background).Padding(t.Space(3), t.Space(4), t.Space(1)).Role(ui.RoleHeading)
				return
			}
			switch e := item(c, &(*items)[i], now); {
			case e.dismiss:
				dismiss = i
			case e.action != "":
				ev.acted, ev.action = i+1, e.action
			case e.open:
				(*items)[i].Read = true
				ev.opened = i + 1
			}
		}).Grow(1).Label("Notifications")
		list.Children(func() {
			if len(s.rows) > 0 {
				return
			}
			text := "No notifications"
			if s.unread == 1 && len(*items) > 0 {
				text = "You're all caught up"
			}
			ui.Text(c, text).TextColor(t.TextMuted).AlignSelf(ui.Center).MarginY(t.Space(12))
		})
	})
	if dismiss >= 0 {
		*items = slices.Delete(*items, dismiss, dismiss+1)
	}
	return ev
}

// groups name the groups of notifications by when they came.
var groups = []string{"Today", "Yesterday", "Earlier"}

// group returns the group of a notification that came at at.
func group(at, now time.Time) int {
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	switch {
	case !at.Before(today):
		return 0
	case !at.Before(today.AddDate(0, 0, -1)):
		return 1
	}
	return 2
}

// ago writes how long ago at was, short: "now", "5m", "3h", "2d", or the
// date after a week.
func ago(at, now time.Time) string {
	d := now.Sub(at)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h"
	case d < 7*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24)) + "d"
	case at.Year() == now.Year():
		return at.Format("Jan 2")
	}
	return at.Format("Jan 2, 2006")
}

// clicks is what the user did to a notification: inner reports a click
// on one of its buttons, which does not open it.
type clicks struct {
	open, dismiss, inner bool
	action               string
}

// item creates the row of it.
func item(c *ui.Context, it *Item, now time.Time) clicks {
	t := c.Theme()
	var cl clicks
	r := ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(3)).AlignItems(ui.Start).Cursor(ui.CursorPointer).
		Label(it.Title).Description(it.Body)
	hover := r.Hovered()
	if hover {
		r.Background(t.SurfaceHover)
	}
	ink := t.Text
	if it.Read {
		ink = t.TextMuted
	}
	r.Children(func() {
		ui.Avatar(c, it.Source, nil).Size(t.Space(9), t.Space(9)).Shrink(0)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				title := ui.Text(c, it.Title).TextColor(ink).SingleLine().Shrink(1).Grow(1)
				if !it.Read {
					title.Bold()
				}
				ui.Text(c, ago(it.Time, now)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum").
					Tooltip(it.Time.Format("Monday, January 2, 2006 at 15:04"))
			})
			if it.Body != "" {
				ui.Text(c, it.Body).FontSize(t.FontSize - 1).TextColor(t.TextMuted).MaxLines(2)
			}
			if len(it.Actions) > 0 {
				ui.Row(c).Gap(t.Space(2)).Margin(t.Space(1.5), 0, 0, 0).Children(func() {
					for k, a := range it.Actions {
						button := ui.Button
						if k == 0 {
							button = ui.PrimaryButton
						}
						b := button(c.Key(k), a)
						if b.Clicked() {
							cl.action, cl.inner = a, true
						}
					}
				})
			}
		})
		ui.Column(c).Width(t.Space(6)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			if !hover {
				if !it.Read {
					ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(t.Accent).Margin(t.Space(1.5), 0, 0, 0).
						Label("Unread")
				}
				return
			}
			mark := "Mark as read"
			glyph := "○"
			if it.Read {
				mark, glyph = "Mark as unread", "●"
			}
			if small(c, glyph, mark).Clicked() {
				it.Read, cl.inner = !it.Read, true
			}
			if small(c, "✕", "Dismiss").Clicked() {
				cl.dismiss, cl.inner = true, true
			}
		})
	})
	if r.Clicked() && !cl.inner {
		cl.open = true
	}
	return cl
}

// small creates a small round button of a glyph, named label.
func small(c *ui.Context, glyph, label string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(6), t.Space(6)).Radius(t.Space(3)).Label(label).Tooltip(label)
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.1))
	}
	b.Children(func() {
		ui.Text(c, glyph).FontSize(t.FontSize - 2).TextColor(t.TextMuted).PassThrough().Center()
	})
	return b
}

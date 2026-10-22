// Swipeactions shows a mail inbox whose messages swipe as in Mail: to the
// right to mark one read or unread, or pin it; to the left to flag it or
// archive it, all the way to archive at once. An archived message
// collapses out of the list, and Undo in the toast below brings it back.
// A click chooses a message, and the same actions are in its context menu.
//
//	go run ./examples/swipeactions
package main

import (
	"log"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/swipeactions"
)

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	unreadIcon  = icon(`<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 7 9 6 9-6"/>`)
	readIcon    = icon(`<path d="M3 10v9a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-9"/><path d="m3 10 9-7 9 7"/><path d="m3 10 9 6 9-6"/>`)
	pinIcon     = icon(`<path d="M12 17v5"/><path d="M9 3h6l-1 6 3 4H7l3-4z"/>`)
	flagIcon    = icon(`<path d="M5 22V4"/><path d="M5 4h12l-2 4.5 2 4.5H5"/>`)
	archiveIcon = icon(`<rect x="3" y="4" width="18" height="5" rx="1"/><path d="M5 9v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V9"/><path d="M10 13h4"/>`)
)

// message is a message of the inbox.
type message struct {
	id      int
	from    string
	subject string
	preview string
	time    string
	unread  bool
	flagged bool
	pinned  bool
}

// app is the state the window shows: the inbox, the message chosen, by
// its id, and the swipe of the list.
type app struct {
	inbox  []message
	chosen int
	swipe  swipeactions.State
}

var collapse = ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	blue := ui.Oklch(0.6, 0.18, 255)
	yellow := ui.Oklch(0.78, 0.15, 85)
	orange := ui.Oklch(0.7, 0.17, 55)
	purple := ui.Oklch(0.55, 0.18, 300)
	archive := swipeactions.Action{Label: "Archive", Icon: archiveIcon, Color: purple}

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		unread := 0
		for _, m := range a.inbox {
			if m.unread {
				unread++
			}
		}
		ui.Column(c).Padding(t.Space(4), t.Space(5), t.Space(3)).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Inbox").FontSize(t.FontSize + 8).Bold()
			ui.Text(c, strconv.Itoa(unread)+" unread").TextColor(t.TextMuted)
		})
		ui.Divider(c)
		archived := -1
		ui.Scroll(c).Grow(1).Label("Messages").Children(func() {
			for i := range a.inbox {
				m := &a.inbox[i]
				read := swipeactions.Action{Label: "Read", Icon: readIcon, Color: blue}
				if !m.unread {
					read = swipeactions.Action{Label: "Unread", Icon: unreadIcon, Color: blue}
				}
				pin := swipeactions.Action{Label: "Pin", Icon: pinIcon, Color: yellow}
				if m.pinned {
					pin.Label = "Unpin"
				}
				flag := swipeactions.Action{Label: "Flag", Icon: flagIcon, Color: orange}
				if m.flagged {
					flag.Label = "Unflag"
				}
				ui.Column(c.Key(m.id)).Transition(collapse).Children(func() {
					ev := swipeactions.Row(c, &a.swipe,
						[]swipeactions.Action{read, pin},
						[]swipeactions.Action{archive, flag},
						func() { a.row(c, m, orange) })
					if ev.Clicked() {
						a.chosen, m.unread = m.id, false
					}
					if act, ok := ev.Chosen(); ok {
						switch act {
						case read:
							m.unread = !m.unread
						case pin:
							m.pinned = !m.pinned
						case flag:
							m.flagged = !m.flagged
						case archive:
							archived = i
						}
					}
				})
			}
			if len(a.inbox) == 0 {
				ui.Text(c, "No messages").TextColor(t.TextMuted).Padding(t.Space(8)).AlignSelf(ui.Center)
			}
		})
		if archived >= 0 {
			m := a.inbox[archived]
			a.inbox = slices.Delete(a.inbox, archived, archived+1)
			c.ToastAction("Message archived", "Undo", func() {
				a.inbox = slices.Insert(a.inbox, min(archived, len(a.inbox)), m)
			})
		}
	})
}

// row shows message m: a dot while unread, the sender, the time, the
// subject and the start of the text, with a pin and a flag as it has them.
func (a *app) row(c *ui.Context, m *message, orange ui.Color) {
	t := c.Theme()
	r := ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(4), t.Space(2.5), t.Space(2)).AlignItems(ui.Start)
	if a.chosen == m.id {
		r.Background(t.Accent.Alpha(0.14))
	}
	r.Draw(func(p *ui.Painter, b ui.Rect) {
		inset := t.Space(6)
		p.Fill(ui.Rect{X: b.X + inset, Y: b.Y + b.H - 1, W: b.W - inset, H: 1}, t.Border, 0)
	})
	r.Children(func() {
		ui.Box(c).Size(t.Space(2.5), t.Space(5)).Draw(func(p *ui.Painter, b ui.Rect) {
			if m.unread {
				d := t.Space(2.5)
				p.Fill(ui.Rect{X: b.X, Y: b.Y + (b.H-d)/2, W: d, H: d}, t.Accent, d/2)
			}
		})
		ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(0.5)).Children(func() {
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
				name := ui.Text(c, m.from).SingleLine().Grow(1).Shrink(1)
				if m.unread {
					name.Bold()
				} else {
					name.FontWeight(500)
				}
				if m.pinned {
					ui.Icon(c, pinIcon).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				}
				if m.flagged {
					ui.Icon(c, flagIcon).FontSize(t.FontSize - 1).TextColor(orange)
				}
				ui.Text(c, m.time).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			})
			ui.Text(c, m.subject).SingleLine()
			ui.Text(c, m.preview).FontSize(t.FontSize - 1).TextColor(t.TextMuted).MaxLines(2)
		})
	})
}

func main() {
	a := &app{chosen: -1, inbox: []message{
		{1, "Maya Chen", "Design review moved to Thursday", "The room was double-booked, so I pushed the review to Thursday at 2pm. Same link, same agenda.", "9:41", true, false, false},
		{2, "GitHub", "[mygo] Release v0.5.5", "A new release is out: list focus handles, faster text layout, and fixes for scroll bars on Linux.", "9:12", true, false, false},
		{3, "Leo Martins", "Lunch?", "There's a new ramen place on 5th that opened this week. Want to try it around 12:30?", "8:55", false, false, true},
		{4, "Priya Natarajan", "Q3 roadmap draft", "Attached is the first draft. I left comments where we still need owners, mostly on the sync work.", "Yesterday", false, true, false},
		{5, "Northwind Air", "Your trip to Lisbon", "Check-in opens 24 hours before departure. Your seat 14A is confirmed.", "Yesterday", true, false, false},
		{6, "Sam Okafor", "Re: Onboarding notes", "Thanks, these are great. One thing to add: the staging keys rotate on Mondays.", "Tuesday", false, false, false},
		{7, "Figma", "Ana commented on Inbox v4", "\"Can the swipe buttons keep their labels when they get narrow?\"", "Tuesday", false, false, false},
		{8, "Hannah Weiss", "Photos from the offsite", "Here's the shared album. Feel free to add yours, there's room for everyone's.", "Monday", false, false, false},
		{9, "Daniel Park", "Invoice #2041", "Please find the invoice for September attached. Payment terms are 30 days.", "Monday", false, true, false},
		{10, "Ravi Iyer", "Weekend climbing", "The forecast looks dry. Meet at the crag at 8, I'll bring the second rope.", "Sunday", false, false, false},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Inbox",
			Width:   440,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

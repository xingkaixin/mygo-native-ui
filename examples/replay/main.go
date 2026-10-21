// Replay plays back a made-up computer-use session in which an agent
// RSVPs to a Go meetup in a browser: it searches for the meetup, waits
// for the results, scrolls to the group's page, copies the invite code,
// and fills in the RSVP form field by field, with a paste from a context
// menu and a drag of a slider. The twelve screenshots are fake browser
// screens drawn when it starts. Play it, step through it with the arrow
// keys, or click a thumbnail to go to its step.
//
//	go run ./examples/replay
package main

import (
	"image"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/replay"
)

type app struct {
	steps  []replay.Step
	step   int
	replay replay.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(3)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "RSVP to the Berlin Gophers meetup").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "A computer-use session in Seek, a made-up browser").TextColor(t.TextMuted)
		})
		replay.View(c, &a.replay, &a.step, a.steps).Grow(1)
	})
}

// The screens are 1280×800 DIPs drawn at 1.5 pixels a DIP, of which the
// browser's bar takes the top bar DIPs.
const (
	screenW, screenH = 1280, 800
	scale            = 1.5
	bar              = 84
)

// pt returns the pixel of the screenshot at (x, y) on a page, in DIPs
// under the browser's bar.
func pt(x, y float32) image.Point {
	return image.Pt(int(x*scale+0.5), int((y+bar)*scale+0.5))
}

// mid returns the pixel of the screenshot at the middle of r on a page.
func mid(r ui.Rect) image.Point { return pt(r.X+r.W/2, r.Y+r.H/2) }

// Where the parts the agent works with are on their pages.
var (
	searchBox = ui.Rect{X: 340, Y: 290, W: 600, H: 52}
	// resultsTop is where the first result starts, and resultH how far
	// apart the results are.
	resultsTop, resultH = float32(160), float32(124)
	// scrolled is how far the agent scrolls the results.
	scrolled = float32(320)
	codeChip = ui.Rect{X: 820, Y: 432, W: 132, H: 30}
	// card is how high the card of the event's details is.
	card     = float32(242)
	rsvp     = ui.Rect{X: 1000, Y: 116, W: 180, H: 48}
	nameBox  = ui.Rect{X: 340, Y: 150, W: 600, H: 44}
	emailBox = ui.Rect{X: 340, Y: 236, W: 600, H: 44}
	codeBox  = ui.Rect{X: 340, Y: 322, W: 600, H: 44}
	// track is the Guests slider, from 0 guests at its left to 4 at its
	// right.
	track   = ui.Rect{X: 340, Y: 430, W: 600, H: 4}
	confirm = ui.Rect{X: 340, Y: 500, W: 600, H: 48}
	// menuAt is where the context menu opens, and its items are menuItem
	// high from 6 below its top, Paste the fourth.
	menuAt   = [2]float32{520, 344}
	menuItem = float32(28)
)

// browser draws a browser window showing url in a tab titled title, with
// page drawn under its bar, as a screenshot.
func browser(title, url string, page func(c *ui.Context)) image.Image {
	return ui.Render(func(c *ui.Context) {
		t := c.Theme()
		ui.Column(c).Fill().Background(t.Background).Children(func() {
			ui.Column(c).Height(bar).Shrink(0).Background(t.Surface).Children(func() {
				ui.Row(c).Height(42).Gap(8).PaddingX(16).AlignItems(ui.End).Children(func() {
					ui.Row(c).Gap(8).AlignSelf(ui.Center).Children(func() {
						for _, light := range []ui.Color{ui.RGB(255, 95, 87), ui.RGB(254, 188, 46), ui.RGB(40, 200, 64)} {
							ui.Box(c).Size(12, 12).Radius(6).Background(light)
						}
					})
					ui.Row(c).MarginStart(16).Width(260).Height(34).Radius(8, 8, 0, 0).Background(t.Background).
						PaddingX(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, title).FontSize(12).SingleLine().Shrink(1)
					})
				})
				ui.Row(c).Height(42).Gap(14).PaddingX(16).AlignItems(ui.Center).Background(t.Background).
					Draw(func(p *ui.Painter, r ui.Rect) {
						p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, t.Border, 0)
					}).Children(func() {
					ui.Text(c, "←   →   ↻").FontSize(15).TextColor(t.TextMuted)
					ui.Row(c).Grow(1).Height(30).Radius(15).Background(t.Surface).PaddingX(14).AlignItems(ui.Center).Children(func() {
						ui.Text(c, url).FontSize(13).TextColor(t.TextMuted).SingleLine()
					})
				})
			})
			ui.Box(c).Grow(1).Clip().Children(func() { page(c) })
		})
	}, screenW, screenH, scale)
}

// at places e on a page at x, y, w wide, or as wide as it is for 0.
func at(e ui.Element, x, y, w float32) ui.Element {
	e.Absolute().Left(x).Top(y)
	if w > 0 {
		e.Width(w)
	}
	return e
}

// field draws a text field in r showing text or, empty, placeholder, with
// a ring and a caret while focused, round if round.
func field(c *ui.Context, r ui.Rect, text, placeholder string, focused, round bool) {
	t := c.Theme()
	radius := float32(8)
	if round {
		radius = r.H / 2
	}
	f := ui.Row(c).Absolute().Left(r.X).Top(r.Y).Size(r.W, r.H).Radius(radius).Border(1, t.Border).
		Background(t.Background).PaddingX(16).AlignItems(ui.Center)
	if focused {
		f.Border(2, t.Accent).Shadow(0, 0, 0, 4, t.Accent.Alpha(0.18))
	}
	f.Children(func() {
		caret := func() {
			if focused {
				ui.Box(c).Size(1.5, 20).Background(t.Text)
			}
		}
		if text == "" {
			caret()
			ui.Text(c, placeholder).FontSize(15).TextColor(t.TextMuted)
			return
		}
		ui.Text(c, text).FontSize(15)
		caret()
	})
}

// newTab draws the new tab page, its search field focused, holding query.
func newTab(query string) image.Image {
	return browser("New Tab", "Search or enter address", func(c *ui.Context) {
		t := c.Theme()
		at(ui.Text(c, "Seek"), 0, 170, screenW).TextAlign(ui.Center).FontSize(64).Bold().TextColor(t.Accent)
		field(c, searchBox, query, "Search the web", true, true)
		names := []string{"Mail", "Calendar", "Docs", "News", "Maps"}
		x := float32(screenW-len(names)*96-(len(names)-1)*24) / 2
		for i, name := range names {
			at(ui.Column(c.Key(i)), x+float32(i)*120, 400, 96).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Size(52, 52).Radius(14).Background(t.Surface).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					ui.Text(c, name[:1]).FontSize(20).Bold().TextColor(t.TextMuted)
				})
				ui.Text(c, name).FontSize(12).TextColor(t.TextMuted)
			})
		}
	})
}

// result is a search result: an ad or not, its site, title and snippet.
type result struct {
	ad                   bool
	site, title, snippet string
}

var results = []result{
	{true, "meetup-tickets.example", "Tech Meetups in Berlin — Tickets from €9", "Find developer meetups near you. Book in seconds, pay at the door. Over 400 events this month."},
	{true, "devjobs.example", "Go Developer Jobs in Berlin", "Senior Go roles at startups and scale-ups. Remote-friendly. Apply with one click."},
	{false, "berlingophers.dev › archive › 2025", "October Meetup 2025 · Berlin Gophers", "Talks from last year's October meetup: profiling in production and fuzzing the standard library."},
	{false, "gophers.slack.example › berlin", "#berlin on Gophers Slack", "Chat with Go developers in Berlin. Meetup announcements are pinned in the channel."},
	{false, "events.example › berlin › go", "Go events in Berlin, 2026", "A list of conferences, workshops and user groups for the Go programming language in Berlin."},
	{false, "berlingophers.dev › events › 2026-10", "October Meetup · Berlin Gophers", "Thu, Oct 23, 19:00. Generics in practice, and iterators with range-over-func. Members RSVP with the invite code."},
	{false, "go.dev › wiki › user-groups", "Go User Groups around the world", "A wiki page of Go user groups by country and city, with links to their sites and chats."},
}

// searchResults draws the results for the query, loading or scrolled down
// by scroll.
func searchResults(loading bool, scroll float32) image.Image {
	const query = "berlin gophers october meetup"
	return browser(query+" - Seek", "seek.example/search?q=berlin+gophers+october+meetup", func(c *ui.Context) {
		t := c.Theme()
		y := func(v float32) float32 { return v - scroll }
		field(c, ui.Rect{X: 40, Y: y(20), W: 640, H: 44}, query, "", false, true)
		at(ui.Row(c), 56, y(82), 0).Gap(28).Children(func() {
			for i, tab := range []string{"All", "Images", "News", "Maps", "Events"} {
				e := ui.Text(c, tab).FontSize(14).TextColor(t.TextMuted)
				if i == 0 {
					e.TextColor(t.Accent).Bold()
				}
			}
		})
		at(ui.Box(c), 0, y(110), screenW).Height(1).Background(t.Border)
		if loading {
			at(ui.Box(c), 0, 0, screenW*0.45).Height(3).Background(t.Accent)
			for i := range 4 {
				top := y(resultsTop + float32(i)*resultH)
				for j, w := range []float32{180, 420, 620, 540} {
					at(ui.Box(c.Key([2]int{i, j})), 40, top+[]float32{0, 22, 54, 74}[j], w).Height([]float32{10, 18, 12, 12}[j]).
						Radius(5).Background(t.Surface)
				}
			}
			return
		}
		at(ui.Text(c, "About 1,240 results (0.31 seconds)"), 40, y(126), 0).FontSize(12).TextColor(t.TextMuted)
		for i, r := range results {
			top := y(resultsTop + float32(i)*resultH)
			at(ui.Row(c.Key([2]int{i, 0})), 40, top, 640).Gap(6).Children(func() {
				if r.ad {
					ui.Text(c, "Sponsored").FontSize(12).Bold()
				}
				ui.Text(c, r.site).FontSize(12).TextColor(t.TextMuted)
			})
			at(ui.Text(c.Key([2]int{i, 1}), r.title), 40, top+20, 0).FontSize(19).TextColor(ui.RGB(26, 13, 171))
			at(ui.Text(c.Key([2]int{i, 2}), r.snippet), 40, top+50, 640).FontSize(13).TextColor(t.TextMuted).LineHeight(1.4)
		}
		at(ui.Column(c), 760, y(160), 440).Gap(8).Padding(20).Radius(12).Border(1, t.Border).Children(func() {
			ui.Text(c, "Berlin Gophers").FontSize(22).Bold()
			ui.Text(c, "Go user group in Berlin").FontSize(13).TextColor(t.TextMuted)
			ui.Text(c, "A monthly meetup with talks on the Go programming language, hosted at Factory Görlitzer Park since 2014.").
				FontSize(13).LineHeight(1.4)
			ui.Text(c, "Members  2,140").FontSize(13)
			ui.Text(c, "Next event  Thu, Oct 23").FontSize(13)
		})
	})
}

// event draws the meetup's page, its invite code selected if selected.
func event(selected bool) image.Image {
	return browser("October Meetup · Berlin Gophers", "berlingophers.dev/events/2026-10", func(c *ui.Context) {
		t := c.Theme()
		goBlue := ui.RGB(0, 173, 216)
		white := ui.RGB(255, 255, 255)
		at(ui.Box(c), 0, 0, screenW).Height(200).Background(goBlue)
		at(ui.Text(c, "BERLIN GOPHERS"), 80, 44, 0).FontSize(13).Bold().LetterSpacing(1.5).TextColor(white.Alpha(0.85))
		at(ui.Text(c, "October Meetup: Generics in Practice"), 80, 70, 0).FontSize(34).Bold().TextColor(white)
		at(ui.Text(c, "Thu, Oct 23, 2026 · 19:00–22:00"), 80, 126, 0).FontSize(16).TextColor(white)
		at(ui.Row(c), rsvp.X, rsvp.Y, rsvp.W).Height(rsvp.H).Radius(10).Background(white).Justify(ui.Center).
			AlignItems(ui.Center).Children(func() {
			ui.Text(c, "RSVP").FontSize(16).Bold().TextColor(goBlue)
		})
		at(ui.Column(c), 80, 240, 640).Gap(14).Children(func() {
			ui.Text(c, "Two talks, pizza and time to talk Go. This month we look at generics in real code bases, and at the iterators range-over-func made possible.").
				FontSize(15).LineHeight(1.5)
			for _, line := range [][2]string{
				{"19:00", "Doors open"},
				{"19:30", "Generics in practice — Lea Hoffmann"},
				{"20:15", "Iterators with range-over-func — Jonas Weber"},
				{"21:00", "Pizza, drinks and talking"},
			} {
				ui.Row(c).Gap(16).Children(func() {
					ui.Text(c, line[0]).FontSize(14).Bold().FontFeatures("tnum").Width(48)
					ui.Text(c, line[1]).FontSize(14)
				})
			}
		})
		at(ui.Box(c), 800, 240, 400).Height(card).Radius(12).Border(1, t.Border)
		for i, line := range []struct {
			y     float32
			text  string
			label bool
		}{
			{260, "Where", true},
			{280, "Factory Görlitzer Park\nLobeckstraße 36, 10969 Berlin", false},
			{330, "Spots", true},
			{350, "23 of 80 left", false},
			{384, "Invite code", true},
			{404, "Members enter it when they RSVP.", false},
		} {
			e := at(ui.Text(c.Key(i), line.text), 820, line.y, 0).FontSize(14).LineHeight(1.4)
			if line.label {
				e.FontSize(12).Bold().TextColor(t.TextMuted)
			}
		}
		code := at(ui.Row(c), codeChip.X, codeChip.Y, codeChip.W).Height(codeChip.H).Radius(6).Background(t.Surface).
			Justify(ui.Center).AlignItems(ui.Center)
		code.Children(func() {
			text := ui.Text(c, "GOPHER1024").Font("monospace").FontSize(15).Bold()
			if selected {
				text.Background(t.Selection)
			}
		})
	})
}

// form is the state of the RSVP form: the name and whether it has the
// focus, the invite code, the guests, and whether the context menu shows
// over the invite code.
type form struct {
	name    string
	focused bool
	code    string
	guests  int
	menu    bool
}

// rsvpForm draws the RSVP form as f says.
func rsvpForm(f form) image.Image {
	return browser("RSVP · Berlin Gophers", "berlingophers.dev/events/2026-10/rsvp", func(c *ui.Context) {
		t := c.Theme()
		at(ui.Text(c, "RSVP"), 340, 36, 0).FontSize(28).Bold()
		at(ui.Text(c, "October Meetup: Generics in Practice · Thu, Oct 23"), 340, 78, 0).FontSize(14).TextColor(t.TextMuted)
		label := func(text string, r ui.Rect) {
			at(ui.Text(c.Key(text), text), r.X, r.Y-24, 0).FontSize(13).Bold()
		}
		label("Name", nameBox)
		field(c, nameBox, f.name, "Your name", f.focused, false)
		label("Email", emailBox)
		field(c, emailBox, "ada@example.com", "", false, false)
		label("Invite code", codeBox)
		field(c, codeBox, f.code, "From the event page", false, false)

		label("Guests", ui.Rect{X: track.X, Y: track.Y - 14})
		guests := "No guests"
		if f.guests > 0 {
			guests = "1 guest"
		}
		at(ui.Text(c, guests), track.X+track.W-120, track.Y-38, 120).FontSize(13).TextAlign(ui.End).TextColor(t.TextMuted)
		at(ui.Box(c), track.X, track.Y, track.W).Height(track.H).Radius(2).Background(t.Border)
		knob := track.X + float32(f.guests)*track.W/4
		at(ui.Box(c), track.X, track.Y, knob-track.X).Height(track.H).Radius(2).Background(t.Accent)
		for i := range 5 {
			at(ui.Text(c.Key(i), string(rune('0'+i))), track.X+float32(i)*track.W/4-20, track.Y+16, 40).
				FontSize(12).TextAlign(ui.Center).TextColor(t.TextMuted)
		}
		at(ui.Box(c), knob-11, track.Y+track.H/2-11, 22).Height(22).Radius(11).Background(t.Background).
			Border(1, t.Border).Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.2))

		at(ui.Row(c), confirm.X, confirm.Y, confirm.W).Height(confirm.H).Radius(10).Background(t.Accent).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Confirm RSVP").FontSize(16).Bold().TextColor(t.AccentText)
		})

		if f.menu {
			at(ui.Column(c), menuAt[0], menuAt[1], 200).Padding(6).Radius(8).Background(t.Background).
				Border(1, t.Border).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).Children(func() {
				for i, item := range []string{"Undo", "Cut", "Copy", "Paste", "Select All"} {
					e := ui.Text(c.Key(i), item).FontSize(14).Height(menuItem).PaddingX(10).AlignItems(ui.Center)
					if item != "Paste" && item != "Select All" {
						e.TextColor(t.TextMuted.Alpha(0.6))
					}
				}
			})
		}
	})
}

// session draws the screens and returns the steps the agent took over
// them.
func session() []replay.Step {
	start := time.Date(2026, 10, 21, 14, 3, 7, 0, time.UTC)
	selected := event(true)
	knobY := track.Y + track.H/2
	steps := []replay.Step{
		{
			Screenshot: newTab(""),
			Action:     replay.Action{Kind: replay.Type, At: mid(searchBox), Text: "berlin gophers october meetup"},
			Note:       "A new tab is open with the search field focused. I'll search for the meetup rather than guess the group's address.",
		},
		{
			Screenshot: newTab("berlin gophers october meetup"),
			Action:     replay.Action{Kind: replay.Key, At: mid(searchBox), Text: "Return"},
			Note:       "The query is in. Pressing Return to search.",
		},
		{
			Screenshot: searchResults(true, 0),
			Action:     replay.Action{Kind: replay.Wait},
			Note:       "The results are still loading. I'll wait for them before reading anything.",
		},
		{
			Screenshot: searchResults(false, 0),
			Action:     replay.Action{Kind: replay.Scroll, At: pt(420, 420), Delta: image.Pt(0, int(scrolled*scale))},
			Note:       "The first two results are ads, and the third is last year's October meetup. The group's page for this month should be further down.",
		},
		{
			Screenshot: searchResults(false, scrolled),
			Action:     replay.Action{Kind: replay.Click, At: pt(180, resultsTop+5*resultH+31-scrolled)},
			Note:       "“October Meetup · Berlin Gophers” on berlingophers.dev is the group's own site, and the right month. Opening it.",
		},
		{
			Screenshot: event(false),
			Action:     replay.Action{Kind: replay.DoubleClick, At: mid(codeChip)},
			Note:       "Members need the invite code on the RSVP form. Selecting it so that I can copy it exactly, rather than retype it.",
		},
		{
			Screenshot: selected,
			Action:     replay.Action{Kind: replay.Key, At: mid(codeChip), Text: "⌘C"},
			Note:       "The code is selected. Copying it.",
		},
		{
			Screenshot: selected,
			Action:     replay.Action{Kind: replay.Click, At: mid(rsvp)},
			Note:       "Now opening the RSVP form.",
		},
		{
			Screenshot: rsvpForm(form{focused: true}),
			Action:     replay.Action{Kind: replay.Type, At: mid(nameBox), Text: "Ada Lovelace"},
			Note:       "The form opened with the Name field focused, and the email is already filled in from the account. Typing the user's name.",
		},
		{
			Screenshot: rsvpForm(form{name: "Ada Lovelace"}),
			Action:     replay.Action{Kind: replay.RightClick, At: pt(menuAt[0], menuAt[1])},
			Note:       "Next the invite code. I'll paste it from the context menu of its field.",
		},
		{
			Screenshot: rsvpForm(form{name: "Ada Lovelace", menu: true}),
			Action:     replay.Action{Kind: replay.Click, At: pt(menuAt[0]+60, menuAt[1]+6+3.5*menuItem)},
			Note:       "Choosing Paste.",
		},
		{
			Screenshot: rsvpForm(form{name: "Ada Lovelace", code: "GOPHER1024"}),
			Action: replay.Action{Kind: replay.Drag, Path: []image.Point{
				pt(track.X, knobY), pt(track.X+24, knobY+1), pt(track.X+58, knobY+2), pt(track.X+96, knobY+2),
				pt(track.X+128, knobY+1), pt(track.X+146, knobY), pt(track.X+track.W/4, knobY),
			}},
			Note: "The code pasted as it was. The user is bringing a friend, so the Guests slider goes from 0 to 1.",
		},
		{
			Screenshot: rsvpForm(form{name: "Ada Lovelace", code: "GOPHER1024", guests: 1}),
			Action:     replay.Action{Kind: replay.Click, At: mid(confirm)},
			Note:       "The name, the invite code and one guest are set, and the email is the account's. Confirming the RSVP.",
		},
	}
	seconds := []float64{0, 3.8, 5.1, 8.4, 12.9, 16.2, 21.5, 23.0, 26.7, 30.9, 35.2, 37.0, 41.8}
	for i := range steps {
		steps[i].Time = start.Add(time.Duration(seconds[i] * float64(time.Second)))
	}
	return steps
}

func main() {
	a := &app{steps: session()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Replay",
			Width:   1280,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

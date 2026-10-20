// Tour shows an onboarding tour over a small notes app: Take the tour
// walks through the sidebar, the archive at its end, which scrolls into
// view, the formatting toolbar, the editor and the share button, each
// lit in turn in the dimmed window, with a popover beside it.
//
//	go run ./examples/tour
package main

import (
	"log"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tour"
)

// note is a note of the app, in the archive or not.
type note struct {
	title, body string
	archived    bool
}

// notes is the state the window shows: the notes, the one open, by its
// index, the formatting of the toolbar, the elements the tour points at,
// and the tour.
type notes struct {
	notes        []note
	open         string
	bold, italic bool
	checklist    bool

	sidebar, archive, format, editor, share ui.Handle

	touring bool
	step    int
	steps   []tour.Step
}

func (a *notes) view(c *ui.Context) {
	t := c.Theme()
	open, _ := strconv.Atoi(a.open)
	n := &a.notes[open]
	items := func(archived bool) {
		for i, n := range a.notes {
			if n.archived == archived {
				ui.SidebarItem(c, strconv.Itoa(i), nil, n.title)
			}
		}
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		ui.Sidebar(c, &a.open, func() {
			ui.SidebarSection(c, "Notes", nil, func() { items(false) })
			ui.SidebarSection(c, "Archive", nil, func() { items(true) }).Bind(&a.archive)
		}).Width(220).Bind(&a.sidebar)
		ui.Divider(c)
		ui.Column(c).Grow(1).Children(func() {
			ui.Toolbar(c, func() {
				ui.ToggleGroup(c, func() {
					ui.Toggle(c, &a.bold, "Bold")
					ui.Toggle(c, &a.italic, "Italic")
					ui.Toggle(c, &a.checklist, "Checklist")
				}).Bind(&a.format)
				ui.Spacer(c)
				if ui.Button(c, "Take the tour").Clicked() {
					a.touring, a.step = true, 0
				}
				ui.PrimaryButton(c, "Share").Bind(&a.share)
			}).Label("Format").Padding(t.Space(2), t.Space(3))
			ui.Divider(c)
			ui.Column(c).Grow(1).Gap(t.Space(2)).Padding(t.Space(6), t.Space(8)).Bind(&a.editor).Children(func() {
				ui.TextInputBase(c, &n.title).FontSize(t.FontSize + 9).Bold().Label("Title")
				ui.TextAreaBase(c, &n.body).Grow(1).LineHeight(1.5).Label("Note")
			})
		})
	})
	tour.Show(c, &a.touring, &a.step, a.steps)
}

func newNotes() *notes {
	a := &notes{
		open: "0",
		notes: []note{
			{"Trip to Lisbon", "Flights land at 9:40 on Friday.\n\n- Book the tram tour\n- Pastéis de Belém before noon\n- Ask Inês about the fado bar in Alfama", false},
			{"Reading list", "The Dispossessed\nA Pattern Language\nThe Making of the Atomic Bomb", false},
			{"Standup notes", "Shipped the export fix. Next: the sync conflict screen.", false},
			{"Gift ideas", "Mum: the blue teapot from the market.\nSam: climbing chalk bag.", false},
			{"Sourdough", "500 g flour, 350 g water, 100 g starter, 10 g salt. Fold every 30 minutes, four times.", false},
			{"Q3 planning", "Themes: onboarding, offline mode, faster search.", false},
			{"Garden", "Tomatoes by the south wall. Basil in pots.", false},
			{"Car service", "Due at 60,000 km. Call the garage on Monday.", false},
			{"Podcast ideas", "Interview the team behind the city's bike lanes.", false},
			{"Apartment", "Measure the alcove for the bookshelf.", false},
			{"Japanese verbs", "taberu, nomu, iku, kuru, suru", false},
			{"Workout", "Mon: legs. Wed: back. Fri: chest and arms.", false},
			{"Recipes to try", "Shakshuka, dal makhani, miso salmon.", false},
			{"Meeting with Ana", "Wants a draft of the brief by Thursday.", false},
			{"Passwords policy", "Rotate the shared vault every quarter.", false},
			{"Birthday party", "Saturday, 4 pm. Bring the speaker.", false},
			{"Bike repair", "Rear brake pads, and true the front wheel.", false},
			{"Book club", "Next month: Piranesi. Host: Priya.", false},
			{"Movies", "Past Lives, Aftersun, Perfect Days.", false},
			{"Plumber", "Kitchen tap drips. Quote was 120.", false},
			{"Old flat inventory", "Sofa, two lamps, the oak desk.", true},
			{"2024 goals", "Run a half marathon. Read 30 books.", true},
			{"Conference talk", "Slides are in the shared drive.", true},
		},
	}
	a.steps = []tour.Step{
		{Target: &a.sidebar, Title: "Your notes", Text: "Every note you write is listed here, the latest first. Click one to open it."},
		{Target: &a.archive, Title: "The archive", Text: "Notes you are done with wait here, out of the way, until you need them again."},
		{Target: &a.format, Title: "Format as you write", Text: "Make the selected text bold or italic, or turn lines into a checklist."},
		{Target: &a.editor, Title: "Just write", Text: "Notes save as you type, so there is no Save button to remember."},
		{Target: &a.share, Title: "Share when it is ready", Text: "Send a link to the note. People you share it with can read it, or edit it if you let them."},
	}
	return a
}

func main() {
	a := newNotes()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Notes",
			Width:   960,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

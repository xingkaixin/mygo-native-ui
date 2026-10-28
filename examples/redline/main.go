// Redline reviews an assistant's edits to the draft of a launch post: each
// edit struck through and underlined in the text, with why it was made,
// and below it the post as decided so far.
//
//	go run ./examples/redline
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/redline"
)

const draft = `Today we are very excited to announce Lantern 2.0, which is a complete rewrite of our sync engine. In our testing it syncs a library of ten thousand notes in about four seconds, compared to the forty seconds it used to take.

The new engine also works offline. Changes that you make on a plane are kept on your device and they are merged when you get back online, even if somebody else edited the same note.

Lantern 2.0 is available today for everyone on macOS, Windows and Linux.`

const suggested = `Today we're announcing Lantern 2.0, a complete rewrite of our sync engine. It syncs a library of ten thousand notes in four seconds, down from forty.

The new engine also works offline. Changes made on a plane stay on your device and merge when you're back online, even if someone else edited the same note.

Lantern 2.0 is available today on macOS, Windows and Linux.`

// notes are why each edit was made, in the order Diff finds them.
var notes = []string{
	"Lead with the news, not the feeling.",
	"The appositive does the work of the clause.",
	"Readers know we tested it.",
	"Four seconds is precise enough to stand alone.",
	"Shorter, and the comparison reads at once.",
	"Forty is clear from the sentence before.",
	"The change is the subject, not the person.",
	"A plainer verb.",
	"Active, and parallel with “stay”.",
	"Contraction, as in the opening.",
	"“Someone” reads warmer than “somebody”.",
	"“For everyone” is implied.",
}

type app struct {
	review redline.State
	pieces []redline.Piece
	text   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(8)).Gap(t.Space(5)).MaxWidth(t.Space(170)).Children(func() {
			ui.Text(c, "Introducing Lantern 2.0").FontSize(t.FontSize + 8).Bold()
			if redline.View(c, &a.review, &a.pieces).Changed {
				a.text = redline.Text(a.pieces)
			}
			ui.Column(c).Gap(t.Space(2)).Padding(t.Space(4)).Radius(t.Space(2)).Background(t.Surface).Children(func() {
				ui.Text(c, "As decided").Bold().FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				ui.Text(c, a.text).Wrap().TextColor(t.TextMuted)
			})
		})
	})
}

func main() {
	a := &app{pieces: redline.Diff(draft, suggested), text: draft}
	n := 0
	for i := range a.pieces {
		if a.pieces[i].Edit() && n < len(notes) {
			a.pieces[i].Author, a.pieces[i].Note = "Editor", notes[n]
			n++
		}
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Redline",
			Width:   860,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// Sharesheet shares a photo: send it to a person through the app on their
// picture, open it in an app, or pick an action, and the toast says what
// happened. Share opens the sheet again after it is closed.
//
//	go run ./examples/sharesheet
package main

import (
	"image"
	"image/color"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sharesheet"
)

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	airdrop   = sharesheet.Target{Name: "AirDrop", Color: ui.Hex("#1E8CF5"), Icon: icon(`<circle cx="12" cy="13" r="1.5"/><path d="M8.5 16.5a5 5 0 1 1 7 0"/><path d="M5.6 19.4a9 9 0 1 1 12.8 0"/>`)}
	messages  = sharesheet.Target{Name: "Messages", Color: ui.Hex("#34C759"), Icon: icon(`<path d="M4 11c0-4 3.6-7 8-7s8 3 8 7-3.6 7-8 7c-1 0-2-.2-3-.5L5 19l1-3.2C4.7 14.6 4 12.9 4 11z" fill="currentColor"/>`)}
	mail      = sharesheet.Target{Name: "Mail", Color: ui.Hex("#2F7CF6"), Icon: icon(`<rect x="3" y="6" width="18" height="12" rx="2"/><path d="M3.5 7l8.5 6 8.5-6"/>`)}
	notes     = sharesheet.Target{Name: "Notes", Color: ui.Hex("#F5B700"), Icon: icon(`<rect x="5" y="4" width="14" height="16" rx="2"/><path d="M9 9h6M9 13h6M9 17h3"/>`)}
	reminders = sharesheet.Target{Name: "Reminders", Color: ui.Hex("#FF6B5A"), Icon: icon(`<circle cx="6" cy="7" r="1"/><circle cx="6" cy="12" r="1"/><circle cx="6" cy="17" r="1"/><path d="M10 7h9M10 12h9M10 17h9"/>`)}
	freeform  = sharesheet.Target{Name: "Freeform", Color: ui.Hex("#5E5CE6"), Icon: icon(`<path d="M4 16c3-6 5-9 7-6s-2 7 1 7 4-6 8-9"/>`)}
)

type app struct {
	open  bool
	photo *ui.Bitmap
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
		ui.Image(c, a.photo).Grow(1).Fit(ui.Contain).Radius(t.Space(2))
		if ui.Button(c, "Share…").Clicked() {
			a.open = true
		}
		ui.Modal(c, &a.open, func() {
			ev := sharesheet.View(c, sharesheet.Item{Title: "Lake at dawn.heic", Detail: "Photo · 3.2 MB", Image: a.photo},
				[]sharesheet.Person{
					{Name: "Maya Chen", Via: messages},
					{Name: "Leo Fischer", Via: airdrop},
					{Name: "Family", Via: messages},
					{Name: "Sam Okafor", Via: mail},
				},
				[]sharesheet.Target{airdrop, messages, mail, notes, reminders, freeform},
				[]sharesheet.Target{
					{Name: "Copy", Icon: icon(`<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/>`)},
					{Name: "Add to Favorites", Icon: icon(`<path d="M12 4l2.5 5 5.5.8-4 3.9.9 5.5-4.9-2.6-4.9 2.6.9-5.5-4-3.9 5.5-.8z"/>`)},
					{Name: "Save to Files", Icon: icon(`<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>`)},
					{Name: "Print", Icon: icon(`<path d="M7 9V4h10v5"/><rect x="4" y="9" width="16" height="7" rx="2"/><path d="M7 14h10v6H7z"/>`)},
				})
			switch {
			case ev.Person != nil:
				c.Toast("Sent to " + ev.Person.Name + " with " + ev.Person.Via.Name)
				a.open = false
			case ev.App != nil:
				c.Toast("Opened in " + ev.App.Name)
				a.open = false
			case ev.Action != nil:
				c.Toast(ev.Action.Name)
				a.open = false
			case ev.Closed:
				a.open = false
			}
		})
	})
}

func lake() *ui.Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, 480, 320))
	for y := range 320 {
		for x := range 480 {
			v := float64(y) / 320
			c := color.RGBA{uint8(250 - 140*v), uint8(190 - 60*v), uint8(160 + 60*v), 255}
			if y > 190 {
				c = color.RGBA{uint8(70 + 30*(1-v)), uint8(110 + 30*(1-v)), uint8(150 + 20*(1-v)), 255}
			}
			img.Set(x, y, c)
		}
	}
	return ui.NewBitmap(img)
}

func main() {
	a := &app{open: true, photo: lake()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Share",
			Width:   760,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

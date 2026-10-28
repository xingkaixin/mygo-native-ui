// Screenplay sets a short made-up screenplay, written in Fountain, about
// a lighthouse keeper and the last ship she guides in.
//
//	go run ./examples/screenplay
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/screenplay"
)

const fountain = `Title: The Last Light
Credit: Written by
Author: A. N. Example
Draft date: 28 October 2026

FADE IN:

EXT. HARROW POINT LIGHTHOUSE - NIGHT

Rain sweeps the rocks. The beam turns, slow and steady, over a black sea.

A single window glows at the base of the tower.

INT. KEEPER'S COTTAGE - CONTINUOUS

A kettle rattles on the stove. MARGIT HOLM (60s), oilskin still dripping, studies a chart pinned under a lamp.

The radio crackles.

RADIO (V.O.)
Harrow Point, this is the Selkie. We have lost our plotter. Do you read?

Margit grabs the handset.

MARGIT
Selkie, Harrow Point. I read you.
(beat)
Keep the light on your port bow and do not turn until I tell you.

RADIO (V.O.)
Understood. Port bow.

[[Make the radio sound tinnier in the mix.]]

CUT TO:

EXT. NORTH SEA - CONTINUOUS

A trawler pitches through the swell, the lighthouse a pinprick ahead.

INT. LIGHTHOUSE LAMP ROOM - LATER

Margit climbs the last step, out of breath. The great lens turns around her.

She checks her watch, then the sea.

MARGIT
(into handset)
Now, Selkie. Hard to starboard.

A long silence. Only the rain.

RADIO (V.O.)
We see the harbour lights. Thank you, Harrow Point.

Margit lowers the handset and, for the first time tonight, sits down.

MARGIT
(to herself)
Last one, then.

> THE END <
`

type app struct {
	script screenplay.Script
	reader screenplay.State
}

func (a *app) view(c *ui.Context) {
	screenplay.View(c, &a.reader, a.script)
}

func main() {
	a := &app{script: screenplay.Parse(fountain)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Screenplay",
			Width:   1100,
			Height:  780,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

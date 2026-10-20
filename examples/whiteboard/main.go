// Whiteboard is a team's sprint retro on an infinite board: sticky notes
// in three columns, Went well, To improve and Actions, with arrows from
// what to improve to what the team will do about it, a goal in an
// ellipse and a note circled with the pen. Given a file, it opens the
// board from it when there is one, and saves the board there as it
// changes.
//
//	go run ./examples/whiteboard [board.json]
package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"math"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/whiteboard"
)

type app struct {
	shapes []whiteboard.Shape
	board  whiteboard.State
	file   string
}

func (a *app) view(c *ui.Context) {
	if whiteboard.View(c, &a.board, &a.shapes).Fill().Changed() && a.file != "" {
		b, _ := json.MarshalIndent(a.shapes, "", "\t")
		if err := os.WriteFile(a.file, b, 0o644); err != nil {
			log.Print(err)
		}
	}
}

// retro returns the board of a retro: a title, a header and notes for
// each column, arrows from each thing to improve to its action, the next
// retro in an ellipse, and the most voted note circled.
func retro() []whiteboard.Shape {
	const (
		note   = 150
		top    = 152
		row    = note + 18
		column = 364
	)
	shapes := []whiteboard.Shape{{ID: 1, Kind: whiteboard.Text, X: 48, Y: 28, Text: "Sprint 24 retro"}}
	id := 1
	add := func(sh whiteboard.Shape) int {
		id++
		sh.ID = id
		shapes = append(shapes, sh)
		return id
	}
	sticky := func(x, y float32, color whiteboard.Color, text string) int {
		return add(whiteboard.Shape{Kind: whiteboard.Sticky, X: x, Y: y, W: note, H: note, Color: color, Text: text})
	}
	for i, name := range []string{"Went well", "To improve", "Actions"} {
		add(whiteboard.Shape{Kind: whiteboard.Rectangle, X: 48 + float32(i)*column, Y: 84, W: 2*note + 18, H: 46, Text: name})
	}
	for i, text := range []string{
		"Search shipped Tuesday with no rollback",
		"Pairing on the parser rewrite",
		"Flaky tests down from 14 to 3",
		"New onboarding docs got a new hire going in a day",
	} {
		sticky(48+float32(i%2)*(note+18), top+float32(i/2)*row, whiteboard.Green, text)
	}
	improve := []string{"Reviews wait two days for a first look", "Standups run past 30 minutes", "Staging was down twice"}
	actions := []string{"First review within 24 hours — Ana", "Timebox standup to 15 minutes — Lee", "Alert on staging health — Raj"}
	for i := range improve {
		y := top + float32(i)*row
		from := sticky(48+column+(2*note+18-note)/2, y, whiteboard.Pink, improve[i])
		to := sticky(48+2*column+(2*note+18-note)/2, y, whiteboard.Yellow, actions[i])
		add(whiteboard.Shape{Kind: whiteboard.Arrow, From: from, To: to})
	}
	add(whiteboard.Shape{Kind: whiteboard.Ellipse, X: 66, Y: top + 2*row + 12, W: 280, H: 120, Text: "Next retro: June 12"})

	// A loop drawn by hand around the note with the most votes.
	var loop []ui.Point
	for k := range 60 {
		t := float64(k) / 55 * 2 * math.Pi
		a := t - 2.4
		loop = append(loop, ui.Point{
			X: float32(120 * math.Cos(a) * (1 + 0.03*math.Sin(3*t))),
			Y: float32(94*math.Sin(a) + 3*t),
		})
	}
	cx, cy := float32(48+column+(2*note+18)/2), float32(top+note/2)
	x0, y0 := float32(math.Inf(1)), float32(math.Inf(1))
	x1, y1 := float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, p := range loop {
		x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
	}
	for k := range loop {
		loop[k].X -= x0
		loop[k].Y -= y0
	}
	add(whiteboard.Shape{Kind: whiteboard.Pen, X: cx + x0, Y: cy + y0 - 6, W: x1 - x0, H: y1 - y0, Points: loop})
	return shapes
}

func main() {
	a := &app{shapes: retro()}
	if len(os.Args) > 1 {
		a.file = os.Args[1]
		b, err := os.ReadFile(a.file)
		switch {
		case err == nil:
			if err := json.Unmarshal(b, &a.shapes); err != nil {
				log.Fatal(err)
			}
		case !errors.Is(err, fs.ErrNotExist):
			log.Fatal(err)
		}
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Whiteboard",
			Width:   1180,
			Height:  800,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

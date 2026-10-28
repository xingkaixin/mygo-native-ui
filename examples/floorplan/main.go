// Floorplan furnishes a made-up two-bedroom flat. Drag the furniture,
// double-click a piece or press R to turn it, and hover a room for the
// lengths of its walls. The sofa starts across the living room's wall,
// outlined in red until it is moved inside.
//
//	go run ./examples/floorplan
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/floorplan"
)

var rooms = []floorplan.Room{
	{Name: "Living room", Corners: [][2]float64{{0, 0}, {5.4, 0}, {5.4, 4.2}, {0, 4.2}}},
	{Name: "Kitchen", Corners: [][2]float64{{5.4, 0}, {8.4, 0}, {8.4, 3}, {5.4, 3}}},
	{Name: "Hall", Corners: [][2]float64{{5.4, 3}, {8.4, 3}, {8.4, 4.2}, {5.4, 4.2}}},
	{Name: "Bedroom", Corners: [][2]float64{{0, 4.2}, {4, 4.2}, {4, 7.6}, {0, 7.6}}},
	{Name: "Bath", Corners: [][2]float64{{4, 4.2}, {5.8, 4.2}, {5.8, 7.6}, {4, 7.6}}},
	{Name: "Study", Corners: [][2]float64{{5.8, 4.2}, {8.4, 4.2}, {8.4, 7.6}, {5.8, 7.6}}},
}

var doors = []floorplan.Door{
	{X: 5.4, Y: 3.2, Width: 0.9, Angle: 90, Swing: 1},
	{X: 7.4, Y: 3, Width: 0.8, Angle: 0, Swing: -1},
	{X: 1, Y: 4.2, Width: 0.8, Angle: 0, Swing: 1},
	{X: 4.4, Y: 4.2, Width: 0.7, Angle: 0, Swing: 1},
	{X: 7.4, Y: 4.2, Width: 0.8, Angle: 0, Swing: 1},
	{X: 8.4, Y: 3.3, Width: 0.9, Angle: 90, Swing: 1},
}

var (
	furniture = []floorplan.Item{
		{Name: "Sofa", X: 1.5, Y: 0.2, W: 2.2, D: 0.9},
		{Name: "Coffee table", X: 1.5, Y: 2.1, W: 1.2, D: 0.6},
		{Name: "Armchair", X: 4.5, Y: 2.1, W: 0.85, D: 0.85},
		{Name: "TV unit", X: 1.5, Y: 3.95, W: 1.8, D: 0.4},
		{Name: "Counter", X: 6.9, Y: 0.3, W: 3, D: 0.6},
		{Name: "Dining table", X: 6.5, Y: 2.35, W: 1.4, D: 0.8},
		{Name: "Bed", X: 3.15, Y: 5.3, W: 1.6, D: 2},
		{Name: "Wardrobe", X: 0.3, Y: 6.4, W: 1.8, D: 0.6, Turn: 1},
		{Name: "Bathtub", X: 4.9, Y: 6.85, W: 1.5, D: 0.75, Turn: 1},
		{Name: "Desk", X: 7.1, Y: 7.25, W: 1.4, D: 0.7},
	}
	plan floorplan.State
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Baseline).Children(func() {
			ui.Text(c, "Flat 4B").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Drag to move · double-click or R to turn").TextColor(t.TextMuted)
		})
		floorplan.View(c, &plan, rooms, doors, &furniture).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Floor plan",
			Width:   900,
			Height:  820,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// Design is an inspector of a card's fill and motion, as design tools
// have: a gradient editor for its background and an easing editor for how
// it moves, the card on the left showing both. Click the card to send it
// across.
//
//	go run ./examples/design
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/easing"
	"github.com/xingkaixin/mygo-native-ui/ui/gradient"
)

type app struct {
	fill   ui.LinearGradient
	editor gradient.State
	curve  easing.Curve
	right  bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		a.stage(c)
		ui.Divider(c).Vertical()
		ui.Scroll(c).Width(t.Space(140)).Children(func() {
			ui.Column(c).Gap(t.Space(4)).Padding(t.Space(5)).Children(func() {
				ui.Text(c, "Fill").Bold()
				gradient.Editor(c, &a.editor, &a.fill)
				ui.Divider(c)
				ui.Text(c, "Motion").Bold()
				easing.Editor(c, &a.curve)
			})
		})
	})
}

// stage shows the card, which a click sends to the other side with the
// curve.
func (a *app) stage(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Background(t.Surface).Clip().Children(func() {
		stage := ui.Column(c).Fill().Justify(ui.Center).Gap(t.Space(4)).Padding(t.Space(8))
		w := stage.LayoutBounds().W - 2*t.Space(8)
		stage.Children(func() {
			card := ui.ButtonBase(c).Label("Card").Size(t.Space(56), t.Space(36)).Radius(t.Space(4)).
				LinearGradient(a.fill).Shadow(0, 8, 24, 0, ui.RGB(0, 0, 0).Alpha(0.18))
			target := float32(0)
			if a.right {
				target = max(0, w-t.Space(56))
			}
			x := card.AnimateTo("x", target, ui.Animation{Duration: 700 * time.Millisecond, Ease: a.curve.Easing()})
			card.Translate(x, 0)
			if card.Clicked() {
				a.right = !a.right
			}
			ui.Text(c, "Click the card to move it").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
	})
}

func main() {
	a := &app{
		fill: ui.LinearGradient{Angle: 135, Oklab: true, Stops: []ui.GradientStop{
			{Color: ui.Hex("#ff7a59"), At: 0},
			{Color: ui.Hex("#ff3d8b"), At: 0.45},
			{Color: ui.Hex("#6d5cff"), At: 1},
		}},
		curve: easing.Curve{X1: 0.34, Y1: 1.56, X2: 0.64, Y2: 1},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Design",
			Width:   1120,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

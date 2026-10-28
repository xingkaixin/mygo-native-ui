// Colorblind shows a status dashboard as most people see it next to how
// it looks with a color vision deficiency: a chart of passing and failing
// deploys, service health in green, amber and red, a categorical palette
// and the spectrum. Choose the vision and its severity above.
//
//	go run ./examples/colorblind
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/colorblind"
)

var (
	pass = ui.RGB(52, 168, 83)
	fail = ui.RGB(219, 68, 55)
	warn = ui.RGB(244, 180, 0)
)

var days = []struct {
	day        string
	pass, fail int
}{
	{"Mon", 18, 2}, {"Tue", 22, 5}, {"Wed", 15, 9}, {"Thu", 24, 1}, {"Fri", 12, 7}, {"Sat", 4, 0}, {"Sun", 3, 2},
}

var services = []struct {
	name  string
	state string
	color ui.Color
}{
	{"API", "Healthy", pass}, {"Workers", "Degraded", warn}, {"Search", "Down", fail}, {"Billing", "Healthy", pass},
}

// tableau are Tableau's ten categorical colors.
var tableau = []ui.Color{
	ui.RGB(78, 121, 167), ui.RGB(242, 142, 43), ui.RGB(225, 87, 89), ui.RGB(118, 183, 178), ui.RGB(89, 161, 79),
	ui.RGB(237, 201, 72), ui.RGB(176, 122, 161), ui.RGB(255, 157, 167), ui.RGB(156, 117, 95), ui.RGB(186, 176, 172),
}

func dashboard(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(4)).Padding(t.Space(5)).Background(t.Surface).Border(1, t.Border).Radius(t.Space(3)).Children(func() {
		ui.Text(c, "Deploys this week").FontWeight(600)
		ui.Row(c).Height(t.Space(36)).Gap(t.Space(2)).AlignItems(ui.End).Children(func() {
			for i, d := range days {
				ui.Column(c.Key(i)).Grow(1).Gap(t.Space(1)).AlignItems(ui.Stretch).Children(func() {
					ui.Box(c).Height(float32(d.fail) * t.Space(1.1)).Background(fail).Radius(2)
					ui.Box(c).Height(float32(d.pass) * t.Space(1.1)).Background(pass).Radius(2)
					ui.Text(c, d.day).FontSize(t.FontSize - 2).TextColor(t.TextMuted).TextAlign(ui.Center)
				})
			}
		})
		ui.Row(c).Gap(t.Space(4)).Children(func() {
			for _, l := range []struct {
				name  string
				color ui.Color
			}{{"Passed", pass}, {"Failed", fail}} {
				ui.Row(c.Key(l.name)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(10, 10).Background(l.color).Radius(5)
					ui.Text(c, l.name).FontSize(t.FontSize - 1)
				})
			}
		})
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			for _, s := range services {
				ui.Row(c.Key(s.name)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(8, 8).Background(s.color).Radius(4)
					ui.Text(c, s.name).Grow(1)
					ui.Text(c, s.state).FontSize(t.FontSize - 1).TextColor(s.color).FontWeight(600)
				})
			}
		})
		ui.Row(c).Radius(4).Clip().Children(func() {
			for i, col := range tableau {
				ui.Box(c.Key(i)).Grow(1).Height(t.Space(5)).Background(col)
			}
		})
		ui.Box(c).Height(t.Space(5)).Radius(4).LinearGradient(ui.LinearGradient{Stops: []ui.GradientStop{
			{At: 0, Color: ui.RGB(255, 0, 0)}, {At: 0.17, Color: ui.RGB(255, 255, 0)}, {At: 0.33, Color: ui.RGB(0, 255, 0)},
			{At: 0.5, Color: ui.RGB(0, 255, 255)}, {At: 0.67, Color: ui.RGB(0, 0, 255)}, {At: 0.83, Color: ui.RGB(255, 0, 255)},
			{At: 1, Color: ui.RGB(255, 0, 0)},
		}, Angle: 90})
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			ui.PrimaryButton(c, "Retry failed")
			ui.Button(c, "Dismiss")
		})
	})
}

type app struct {
	vision   int
	severity float64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	names := make([]string, len(colorblind.Visions)-1)
	for i, v := range colorblind.Visions[1:] {
		names[i] = v.String()
	}
	v := colorblind.Visions[a.vision+1]
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.vision, names...).Label("Vision")
			ui.Text(c, "Severity")
			ui.Slider(c, &a.severity, 0, 1).Width(t.Space(40))
			ui.Text(c, fmt.Sprintf("%.0f%%", a.severity*100)).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Row(c).Gap(t.Space(5)).Children(func() {
			for i, side := range []struct {
				title string
				v     colorblind.Vision
			}{{"Normal", colorblind.Normal}, {v.String(), v}} {
				ui.Column(c.Key(i)).Grow(1).Basis(0).Gap(t.Space(2)).Children(func() {
					ui.Text(c, side.title).FontWeight(600).TextColor(t.TextMuted)
					colorblind.Simulate(c, side.v, float32(a.severity), func() { dashboard(c) })
				})
			}
		})
	})
}

func main() {
	a := &app{vision: 1, severity: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Colorblind",
			Width:   960,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

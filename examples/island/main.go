// Island runs a kitchen timer and a song as live activities: start them
// below, click the capsule at the top to open the one in front, and click
// the circle beside it to bring the other forward.
//
//	go run ./examples/island
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/island"
)

var (
	orange = ui.Hex("#ff9f0a")
	pink   = ui.Hex("#ff375f")
	white  = ui.RGB(255, 255, 255)
)

type app struct {
	island  island.State
	timerOn bool
	ends    time.Time
	paused  time.Duration
	playing bool
	song    time.Duration
	started time.Time
}

func (a *app) left(now time.Time) time.Duration {
	if a.paused > 0 {
		return a.paused
	}
	return max(0, a.ends.Sub(now))
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	var activities []island.Activity
	if a.timerOn {
		left := a.left(now)
		if left == 0 {
			a.timerOn = false
		} else if a.paused == 0 {
			c.After(left % time.Second)
		}
		clock := fmt.Sprintf("%d:%02d", int(left.Minutes()), int(left.Seconds())%60)
		activities = append(activities, island.Activity{
			Leading:  func() { timerIcon(c, 18) },
			Trailing: func() { ui.Text(c, clock).TextColor(orange).Bold().FontFeatures("tnum") },
			Minimal:  func() { timerIcon(c, 18) },
			Expanded: func() { a.timerPanel(c, clock) },
		})
	}
	if a.playing {
		activities = append(activities, island.Activity{
			Leading:  func() { cover(c, 22) },
			Trailing: func() { bars(c) },
			Minimal:  func() { cover(c, 22) },
			Expanded: func() { a.songPanel(c, now) },
		})
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(24), t.Space(10)).Gap(t.Space(3)).Children(func() {
		ui.Text(c, "Kitchen").FontSize(t.FontSize + 8).Bold()
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			timer := "Start 5-minute timer"
			if a.timerOn {
				timer = "Stop timer"
			}
			if ui.Button(c, timer).Clicked() {
				a.timerOn, a.ends, a.paused = !a.timerOn, now.Add(5*time.Minute), 0
			}
			music := "Play music"
			if a.playing {
				music = "Stop music"
			}
			if ui.Button(c, music).Clicked() {
				a.playing, a.started = !a.playing, now
			}
		})
		ui.Text(c, "Click the capsule to open it, and outside it to close it.").TextColor(t.TextMuted)
	})
	island.View(c, &a.island, activities)
}

func (a *app) timerPanel(c *ui.Context, clock string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
		timerIcon(c, 40)
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, "Pasta").TextColor(white.Alpha(0.6))
			ui.Text(c, clock).FontSize(t.FontSize * 3).TextColor(orange).FontFeatures("tnum")
		})
		pause := "Pause"
		if a.paused > 0 {
			pause = "Resume"
		}
		if round(c, pause, orange.Alpha(0.25), orange).Clicked() {
			if a.paused > 0 {
				a.ends, a.paused = c.Now().Add(a.paused), 0
			} else {
				a.paused = a.left(c.Now())
			}
		}
		if round(c, "Stop", white.Alpha(0.15), white).Clicked() {
			a.timerOn = false
		}
	})
}

func (a *app) songPanel(c *ui.Context, now time.Time) {
	t := c.Theme()
	const length = 3*time.Minute + 42*time.Second
	at := now.Sub(a.started) % length
	c.After(time.Second - at%time.Second)
	ui.Column(c).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			cover(c, 52)
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, "Low Tide").Bold()
				ui.Text(c, "The Harbour Lights").TextColor(white.Alpha(0.6))
			})
			bars(c)
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).FontSize(t.FontSize - 2).TextColor(white.Alpha(0.6)).Children(func() {
			ui.Text(c, fmt.Sprintf("%d:%02d", int(at.Minutes()), int(at.Seconds())%60)).FontFeatures("tnum")
			done := float32(at) / float32(length)
			ui.Box(c).Height(5).Grow(1).Radius(3).Background(white.Alpha(0.2)).Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * done, H: r.H}, white, 3)
			})
			ui.Text(c, "-"+fmt.Sprintf("%d:%02d", int((length-at).Minutes()), int((length-at).Seconds())%60)).FontFeatures("tnum")
		})
	})
}

// round creates a round button of label, in fill with ink.
func round(c *ui.Context, label string, fill, ink ui.Color) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c.Key(label)).Padding(t.Space(2), t.Space(3.5)).Radius(t.Space(5)).Background(fill).Label(label)
	b.Children(func() { ui.Text(c, label).TextColor(ink).Bold() })
	return b
}

// timerIcon draws a stopwatch size DIPs across.
func timerIcon(c *ui.Context, size float32) {
	ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2+r.H*0.05, r.W*0.4
		var ring ui.Path
		p.StrokePath(ring.Circle(cx, cy, rad), r.W*0.1, orange)
		p.Line(cx, cy, cx+rad*0.45, cy-rad*0.45, r.W*0.09, orange)
		p.Fill(ui.Rect{X: cx - r.W*0.1, Y: r.Y, W: r.W * 0.2, H: r.H * 0.1}, orange, 1)
	})
}

// cover draws an album cover size DIPs across.
func cover(c *ui.Context, size float32) {
	ui.Box(c).Size(size, size).Radius(size*0.2).Gradient(ui.Hex("#5e5ce6"), pink, 135)
}

// bars draws an equalizer moving to the music.
func bars(c *ui.Context) {
	ui.Box(c).Size(20, 16).Draw(func(p *ui.Painter, r ui.Rect) {
		p.AnimationFrameRate(30)
		s := float64(p.Now().UnixMilli()) / 1000
		for i := range 4 {
			f := 0.35 + 0.65*math.Abs(math.Sin(s*(3+float64(i)*1.3)+float64(i)))
			h := r.H * float32(f)
			p.Fill(ui.Rect{X: r.X + float32(i)*5.5, Y: r.Y + r.H - h, W: 3, H: h}, pink, 1.5)
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Island",
			Width:   720,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

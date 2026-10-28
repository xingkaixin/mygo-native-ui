// HUD plays an album in a window that answers the keys of a Mac's volume
// and brightness: ↑ and ↓ turn the volume, ← and → the brightness, and M
// mutes. Each change brings up its HUD over the window, which fades out
// a moment later; drag along its notches to set the level.
//
//	go run ./examples/hud
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hud"
)

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

const speakerBody = `<path d="M4 9.5h3.2L12 5.5v13l-4.8-4H4z" fill="currentColor"/>`

var (
	speakers = []*ui.SVG{
		icon(speakerBody + `<path d="M16 9.5l4 5M20 9.5l-4 5"/>`),
		icon(speakerBody + `<path d="M15.5 10a3 3 0 0 1 0 4"/>`),
		icon(speakerBody + `<path d="M15.5 10a3 3 0 0 1 0 4M18 8a6 6 0 0 1 0 8"/>`),
		icon(speakerBody + `<path d="M15.5 10a3 3 0 0 1 0 4M18 8a6 6 0 0 1 0 8M20.5 6a9 9 0 0 1 0 12"/>`),
	}
	sun = icon(`<circle cx="12" cy="12" r="4" fill="currentColor"/><path d="M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M5.3 18.7l1.4-1.4M17.3 6.7l1.4-1.4"/>`)
)

func speaker(v float64, muted bool) *ui.SVG {
	switch {
	case muted || v == 0:
		return speakers[0]
	case v < 0.34:
		return speakers[1]
	case v < 0.67:
		return speakers[2]
	}
	return speakers[3]
}

const step = 1.0 / 16

type app struct {
	volume, brightness float64
	muted              bool
	sound, light       hud.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	switch {
	case c.Shortcut(0, ui.KeyUp):
		a.volume, a.muted = min(1, a.volume+step), false
		a.sound.Show()
		a.light.Hide()
	case c.Shortcut(0, ui.KeyDown):
		a.volume, a.muted = max(0, a.volume-step), false
		a.sound.Show()
		a.light.Hide()
	case c.Shortcut(0, ui.KeyM):
		a.muted = !a.muted
		a.sound.Show()
		a.light.Hide()
	case c.Shortcut(0, ui.KeyRight):
		a.brightness = min(1, a.brightness+step)
		a.light.Show()
		a.sound.Hide()
	case c.Shortcut(0, ui.KeyLeft):
		a.brightness = max(0, a.brightness-step)
		a.light.Show()
		a.sound.Hide()
	}
	ui.Box(c).Fill().Children(func() {
		ui.Column(c).Fill().Padding(t.Space(10)).Gap(t.Space(4)).Center().
			LinearGradient(ui.LinearGradient{Angle: 135, Stops: []ui.GradientStop{
				{At: 0, Color: ui.RGB(251, 113, 133)}, {At: 0.5, Color: ui.RGB(192, 132, 252)}, {At: 1, Color: ui.RGB(56, 189, 248)},
			}}).Children(func() {
			ui.Box(c).Size(220, 220).Radius(16).Background(ui.RGB(255, 255, 255).Alpha(0.25)).Shadow(0, 12, 32, 0, ui.RGB(0, 0, 0).Alpha(0.25))
			ui.Text(c, "Neon Harbor").FontSize(t.FontSize + 10).Bold().TextColor(ui.RGB(255, 255, 255))
			ui.Text(c, "Tidal Lights · 2026").TextColor(ui.RGB(255, 255, 255).Alpha(0.8))
			ui.Text(c, "↑ ↓ volume   ← → brightness   M mute").FontSize(t.FontSize - 1).TextColor(ui.RGB(255, 255, 255).Alpha(0.7))
		})
		ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).PassThrough().Background(ui.RGB(0, 0, 0).Alpha(float32(1-a.brightness) * 0.7))

		level := a.volume
		if a.muted {
			level = 0
		}
		before := level
		if hud.View(c, &a.sound, speaker(level, a.muted), &level); level != before {
			a.volume, a.muted = level, false
			a.light.Hide()
		}
		before = a.brightness
		if hud.View(c, &a.light, sun, &a.brightness); a.brightness != before {
			a.sound.Hide()
		}
	})
}

func main() {
	a := &app{volume: 0.5, brightness: 0.8}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Now Playing",
			Width:   640,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

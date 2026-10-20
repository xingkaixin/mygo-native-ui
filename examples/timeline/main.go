// Timeline is the cut of a short film: drag clips to move and trim them,
// click the ruler to move the playhead, S to split, and play it to watch
// the monitor follow.
//
//	go run ./examples/timeline
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/timeline"
)

var tracks = []string{"Titles", "Video 2", "Video 1", "Music"}

func s(sec float64) time.Duration { return time.Duration(sec * float64(time.Second)) }

func cut() []timeline.Clip {
	return []timeline.Clip{
		{Name: "Opening title", Track: 0, Start: s(1), Length: s(4)},
		{Name: "Lower third", Track: 0, Start: s(14), Length: s(5)},
		{Name: "B-roll street", Track: 1, Start: s(8), Length: s(6), In: s(2), Source: s(20)},
		{Name: "Harbor wide", Track: 2, Start: 0, Length: s(9), Source: s(15)},
		{Name: "Interview", Track: 2, Start: s(9), Length: s(14), In: s(30), Source: s(90)},
		{Name: "Sunset", Track: 2, Start: s(23), Length: s(7), Source: s(12)},
		{Name: "Theme", Track: 3, Start: 0, Length: s(30), Source: s(184)},
	}
}

type app struct {
	edit     timeline.State
	clips    []timeline.Clip
	playhead time.Duration
	playing  bool
	last     time.Time
}

// play moves the playhead along by the time since the last frame, to the
// end of the cut and no further.
func (a *app) play(now time.Time) {
	if !a.playing {
		a.last = time.Time{}
		return
	}
	if !a.last.IsZero() {
		a.playhead += now.Sub(a.last)
	}
	a.last = now
	var end time.Duration
	for _, cl := range a.clips {
		end = max(end, cl.End())
	}
	if a.playhead >= end {
		a.playhead, a.playing = end, false
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.play(c.Now())
	if a.playing {
		c.After(time.Second / 30)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Harbor — rough cut").Bold().FontSize(t.FontSize + 6).Grow(1)
			ui.Text(c, "S splits the chosen clip at the playhead").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.Button(c, label).Clicked() {
				a.playing = !a.playing
			}
		})
		a.monitor(c)
		timeline.View(c, &a.edit, tracks, &a.clips, &a.playhead)
		chosen := "No clip chosen"
		if i, ok := a.edit.Chosen(); ok && i < len(a.clips) {
			cl := a.clips[i]
			chosen = cl.Name + " · " + timeline.Clock(cl.Start) + " – " + timeline.Clock(cl.End()) + " · source from " + timeline.Clock(cl.In)
		}
		ui.Text(c, chosen).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

// monitor shows what plays at the playhead: the top video, the title over
// it, the music under it.
func (a *app) monitor(c *ui.Context) {
	t := c.Theme()
	now := map[int]timeline.Clip{}
	for _, cl := range a.clips {
		if cl.Start <= a.playhead && a.playhead < cl.End() {
			now[cl.Track] = cl
		}
	}
	video, ok := now[1]
	if !ok {
		video, ok = now[2]
	}
	screen := ui.RGB(16, 18, 22)
	if ok {
		screen = ui.Oklch(0.45, 0.08, float32(len(video.Name)*47%360))
	}
	white := ui.RGB(255, 255, 255)
	ui.Column(c).AlignSelf(ui.Center).Size(t.Space(120), t.Space(60)).Radius(t.Space(2)).
		Background(screen).Justify(ui.End).Padding(t.Space(4)).Gap(t.Space(1)).Children(func() {
		if title, ok := now[0]; ok {
			ui.Text(c, title.Name).Bold().FontSize(t.FontSize + 8).TextColor(white)
		}
		if ok {
			ui.Text(c, video.Name+" @ "+timeline.Clock(video.In+a.playhead-video.Start)).FontSize(t.FontSize - 2).TextColor(white.Alpha(0.75)).Font("monospace")
		} else {
			ui.Text(c, "No video").FontSize(t.FontSize - 2).TextColor(white.Alpha(0.5))
		}
		if music, ok := now[3]; ok {
			ui.Text(c, "♪ "+music.Name).FontSize(t.FontSize - 2).TextColor(white.Alpha(0.75))
		}
	})
}

func main() {
	a := &app{clips: cut(), playhead: s(3)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Timeline",
			Width:   1000,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

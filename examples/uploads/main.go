// Uploads sends files to a made-up server, three at a time over a shared
// link, as cloud drives do. The recording fails partway the first time;
// retry it, or cancel others.
//
//	go run ./examples/uploads
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/uploads"
)

const (
	link     = 8_000_000 // bytes a second
	together = 3
)

var start = []uploads.File{
	{Name: "Quarterly report.pdf", Size: 4_200_000},
	{Name: "Team offsite.heic", Size: 3_100_000},
	{Name: "screen-recording.mov", Size: 64_000_000},
	{Name: "design-assets.zip", Size: 28_500_000},
	{Name: "IMG_2041.jpg", Size: 2_700_000},
	{Name: "Budget 2027.xlsx", Size: 860_000},
	{Name: "keynote-draft.key", Size: 18_300_000},
}

type app struct {
	files  []uploads.File
	queue  uploads.State
	last   time.Time
	failed bool
}

// send moves the files along by the time since the last frame.
func (a *app) send(now time.Time) bool {
	if a.last.IsZero() {
		a.last = now
	}
	dt := now.Sub(a.last).Seconds()
	a.last = now
	busy := 0
	for i := range a.files {
		f := &a.files[i]
		if f.Status == uploads.Waiting && busy < together {
			f.Status = uploads.Sending
		}
		if f.Status == uploads.Sending {
			busy++
		}
	}
	for i := range a.files {
		f := &a.files[i]
		if f.Status != uploads.Sending {
			continue
		}
		f.Sent = min(f.Size, f.Sent+int64(dt*link/float64(busy)))
		if f.Name == "screen-recording.mov" && !a.failed && f.Sent > f.Size*55/100 {
			f.Status, f.Error, a.failed = uploads.Failed, "Connection lost", true
		} else if f.Sent == f.Size {
			f.Status = uploads.Done
		}
	}
	return busy > 0
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.send(c.Now()) {
		c.After(time.Second / 30)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ev := uploads.View(c, &a.queue, a.files)
		if i, ok := ev.Canceled(); ok {
			a.files[i].Status = uploads.Canceled
		}
		if i, ok := ev.Retried(); ok {
			a.files[i] = uploads.File{Name: a.files[i].Name, Size: a.files[i].Size}
		}
		if ui.Button(c, "Upload again").AlignSelf(ui.Start).Clicked() {
			a.files, a.failed = append([]uploads.File(nil), start...), false
		}
	})
}

func main() {
	a := &app{files: append([]uploads.File(nil), start...)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Uploads",
			Width:   720,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

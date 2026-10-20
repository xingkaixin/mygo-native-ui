// Package uploads shows files being sent, as upload queues of cloud drives
// do: above, how many are done and how much is left of them all, at what
// speed and for how long; below, a row for each file, its progress, speed
// and time left, or why it failed, with a button to cancel it while it
// waits or is being sent and one to try it again after. The app sends the
// files, updating Sent and Status, and cancels or retries the one that
// Canceled or Retried returns in the same frame.
//
//	ev := uploads.View(c, &app.queue, app.files)
//	if i, ok := ev.Canceled(); ok {
//		app.cancel(i)
//	}
//	if i, ok := ev.Retried(); ok {
//		app.retry(i)
//	}
package uploads

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Status is where a file is.
type Status uint8

const (
	Waiting Status = iota
	Sending
	Done
	Failed
	Canceled
)

// File is a file to send: its name, its size and how much of it is sent,
// in bytes, where it is, and why it failed.
type File struct {
	Name   string
	Size   int64
	Sent   int64
	Status Status
	Error  string
}

// Events is what the user did to a queue in a frame.
type Events struct {
	cancel, retry int
}

// Canceled returns the index of the file to cancel.
func (e Events) Canceled() (int, bool) { return e.cancel - 1, e.cancel > 0 }

// Retried returns the index of the file to send again.
func (e Events) Retried() (int, bool) { return e.retry - 1, e.retry > 0 }

// State is what a queue keeps: how much was sent lately, for the speeds.
// Keep it in the app's state, one for each queue.
type State struct {
	samples map[int][]sample
	total   []sample
}

type sample struct {
	at   time.Time
	sent int64
}

// window is how far back speeds look, to steady them.
const window = 3 * time.Second

// speed adds sent at now to samples and returns them, with the speed
// over them in bytes a second.
func speed(samples []sample, now time.Time, sent int64) ([]sample, float64) {
	if n := len(samples); n > 0 && sent < samples[n-1].sent {
		samples = nil
	}
	samples = append(samples, sample{now, sent})
	k := 0
	for k < len(samples)-1 && now.Sub(samples[k].at) > window {
		k++
	}
	samples = samples[k:]
	first := samples[0]
	dt := now.Sub(first.at).Seconds()
	if dt < 0.5 {
		return samples, 0
	}
	return samples, float64(sent-first.sent) / dt
}

// View creates the queue of files.
func View(c *ui.Context, s *State, files []File) Events {
	t := c.Theme()
	now := c.Now()
	if s.samples == nil {
		s.samples = map[int][]sample{}
	}
	var ev Events
	var size, sent int64
	done, sending := 0, false
	for _, f := range files {
		if f.Status == Canceled {
			continue
		}
		size += f.Size
		sent += f.Sent
		if f.Status == Done {
			done++
		}
		sending = sending || f.Status == Sending
	}
	var rate float64
	if sending {
		s.total, rate = speed(s.total, now, sent)
	} else {
		s.total = nil
	}
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("%d of %d files uploaded", done, count(files))).Bold().Grow(1)
				ui.Text(c, line(sent, size, rate, sending)).FontFeatures("tnum").TextColor(t.TextMuted)
			})
			bar(c, sent, size, t.Accent).Height(t.Space(1.5))
		})
		ui.Column(c).Children(func() {
			for i, f := range files {
				var r float64
				if f.Status == Sending {
					s.samples[i], r = speed(s.samples[i], now, f.Sent)
				} else {
					delete(s.samples, i)
				}
				ui.Row(c.Key(i)).Gap(t.Space(3)).Padding(t.Space(2.5), 0).AlignItems(ui.Center).Children(func() {
					badge(c, f.Name)
					ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(1)).Children(func() {
						ui.Row(c).Gap(t.Space(2)).Children(func() {
							ui.Text(c, f.Name).SingleLine().Shrink(1).Grow(1)
							status, color := about(f, r), t.TextMuted
							if f.Status == Failed {
								color = t.Danger
							}
							ui.Text(c, status).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(color).SingleLine()
						})
						color := t.Accent
						switch f.Status {
						case Done:
							color = t.Success
						case Failed:
							color = t.Danger
						case Canceled, Waiting:
							color = t.TextMuted
						}
						bar(c, f.Sent, f.Size, color).Height(t.Space(1))
					})
					ui.Box(c).Width(t.Space(10)).Shrink(0).AlignItems(ui.Center).Children(func() {
						switch f.Status {
						case Waiting, Sending:
							if ui.Button(c, "×").Label("Cancel " + f.Name).Tooltip("Cancel").Clicked() {
								ev.cancel = i + 1
							}
						case Failed, Canceled:
							if ui.Button(c, "↻").Label("Retry " + f.Name).Tooltip("Retry").Clicked() {
								ev.retry = i + 1
							}
						default:
							ui.Text(c, "✓").TextColor(t.Success)
						}
					})
				})
			}
		})
	})
	if sending {
		// Speeds and times left change as time goes, without the app
		// changing anything.
		c.After(time.Second / 2)
	}
	return ev
}

func count(files []File) int {
	n := 0
	for _, f := range files {
		if f.Status != Canceled {
			n++
		}
	}
	return n
}

// line writes how much of size is sent, at rate, and how long is left.
func line(sent, size int64, rate float64, sending bool) string {
	out := bytes(sent) + " of " + bytes(size)
	if sending && rate > 0 {
		out += " · " + bytes(int64(rate)) + "/s · " + left(float64(size-sent)/rate)
	}
	return out
}

func about(f File, rate float64) string {
	switch f.Status {
	case Waiting:
		return "Waiting · " + bytes(f.Size)
	case Sending:
		return line(f.Sent, f.Size, rate, true)
	case Done:
		return bytes(f.Size)
	case Failed:
		if f.Error != "" {
			return f.Error
		}
		return "Failed"
	}
	return "Canceled"
}

// left writes seconds as a time left, rounded up: "8 s left", "3 min
// left".
func left(seconds float64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%d s left", int(seconds)+1)
	case seconds < 3600:
		return fmt.Sprintf("%d min left", int(seconds/60)+1)
	}
	return fmt.Sprintf("%d h left", int(seconds/3600)+1)
}

// bytes writes n bytes as "820 B", "4.2 MB" or "1.3 GB".
func bytes(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	f, k := float64(n)/unit, 0
	for f >= unit && k < 3 {
		f, k = f/unit, k+1
	}
	return fmt.Sprintf("%.1f %cB", f, "kMGT"[k])
}

func bar(c *ui.Context, sent, size int64, color ui.Color) ui.Element {
	t := c.Theme()
	frac := float32(0)
	if size > 0 {
		frac = min(1, float32(sent)/float32(size))
	}
	return ui.Box(c).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, t.Text.Alpha(0.08), r.H/2)
		if frac > 0 {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: max(r.H, r.W*frac), H: r.H}, color, r.H/2)
		}
	})
}

// badge shows a file's extension on a page, colored by its kind.
func badge(c *ui.Context, name string) {
	t := c.Theme()
	ext := strings.ToUpper(strings.TrimPrefix(path.Ext(name), "."))
	if len(ext) > 4 {
		ext = ext[:4]
	}
	hue := float32(250)
	switch strings.ToLower(ext) {
	case "png", "jpg", "jpeg", "gif", "heic", "webp":
		hue = 150
	case "mp4", "mov", "mkv":
		hue = 330
	case "zip", "tar", "gz":
		hue = 70
	case "pdf":
		hue = 25
	}
	color := ui.Oklch(0.62, 0.15, hue)
	ui.Box(c).Size(t.Space(9), t.Space(10)).Shrink(0).Radius(t.Space(1.5)).Background(color.Alpha(0.15)).
		Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, ext).FontSize(t.FontSize - 4).Bold().TextColor(color)
	})
}

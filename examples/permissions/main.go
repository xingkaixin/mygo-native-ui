// Permissions is the first run of a made-up meeting recorder, which needs
// the microphone and screen recording and would like the calendars and
// notifications. The system's answers are made up after a moment: screen
// recording is refused until Open System Settings, which turns it on, and
// notifications stay refused. Continue comes on once both required ones
// are allowed.
//
//	go run ./examples/permissions
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/permissions"
)

// answer is what the made-up system says to a permission, at a time.
type answer struct {
	at     time.Time
	status permissions.Status
}

type app struct {
	perms   []permissions.Permission
	pending map[int]answer
	done    bool
}

// says is what the made-up system answers the first time each is asked.
var says = map[permissions.Kind]permissions.Status{
	permissions.ScreenRecording: permissions.Denied,
	permissions.Notifications:   permissions.Denied,
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	for i, ans := range a.pending {
		if wait := ans.at.Sub(c.Now()); wait > 0 {
			c.After(wait)
			continue
		}
		a.perms[i].Status = ans.status
		delete(a.pending, i)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		if a.done {
			ui.Column(c).Fill().Center().Gap(t.Space(2)).Children(func() {
				ui.Text(c, "You're all set").Bold().FontSize(t.FontSize + 6)
				ui.Text(c, "Minutes will start recording when your next meeting begins.").TextColor(t.TextMuted)
			})
			return
		}
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, "Minutes needs a few permissions").Bold().FontSize(t.FontSize + 6)
			ui.Text(c, "You can change these later in System Settings → Privacy & Security.").TextColor(t.TextMuted)
		})
		ev := permissions.View(c, a.perms)
		if i, ok := ev.Requested(); ok {
			status, ok := says[a.perms[i].Kind]
			if !ok {
				status = permissions.Granted
			}
			a.pending[i] = answer{c.Now().Add(600 * time.Millisecond), status}
		}
		if i, ok := ev.OpenedSettings(); ok && a.perms[i].Kind == permissions.ScreenRecording {
			a.pending[i] = answer{c.Now().Add(1500 * time.Millisecond), permissions.Granted}
		}
		if ev.Continued() {
			a.done = true
		}
	})
}

func main() {
	a := &app{pending: map[int]answer{}, perms: []permissions.Permission{
		{Kind: permissions.Microphone, Required: true, Reason: "To record what you and others say in meetings."},
		{Kind: permissions.ScreenRecording, Required: true, Reason: "To capture the audio of Zoom, Meet and Teams calls."},
		{Kind: permissions.Calendars, Reason: "To name recordings after the meetings they belong to."},
		{Kind: permissions.Notifications, Reason: "To remind you to record as a meeting starts."},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Welcome to Minutes",
			Width:   720,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

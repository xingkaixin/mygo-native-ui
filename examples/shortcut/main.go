// Shortcut records two shortcuts in a settings window: one that brings the
// window to the front from any app, which it registers with
// mygo.GlobalShortcut, and one that works while the window has the
// keyboard, which counts its presses.
//
//	go run ./examples/shortcut
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/shortcut"
)

// settings is the state the window shows: the two shortcuts, the
// accelerator registered for the global one, "" while none is, why it is
// not, and how many times each shortcut was pressed.
type settings struct {
	win           *mygo.Window
	global, local shortcut.Shortcut
	registered    string
	err           string
	fronted       int
	pressed       int
}

// register registers the global shortcut in place of the one registered
// before, if any.
func (s *settings) register() {
	mygo.GlobalShortcut.Unregister(s.registered)
	s.registered, s.err = "", ""
	acc := s.global.String()
	if acc == "" {
		return
	}
	if err := mygo.GlobalShortcut.Register(acc, s.front); err != nil {
		s.err = err.Error()
		return
	}
	s.registered = acc
}

// front brings the window to the front, as the global shortcut does.
func (s *settings) front() {
	s.fronted++
	s.win.Show()
	s.win.Focus()
	s.win.Invalidate()
}

func (s *settings) view(c *ui.Context) {
	t := c.Theme()
	if s.local.Pressed(c) {
		s.pressed++
	}
	ui.Column(c).Fill().Padding(24).Gap(20).Background(t.Background).Children(func() {
		ui.Text(c, "Shortcuts").FontSize(16).Bold()
		ui.Form(c, func() {
			ui.Field(c, "Bring to front", func() {
				if shortcut.Field(c, &s.global).Changed() {
					s.register()
				}
			}).Description(fmt.Sprintf("From any app. Used %d times.", s.fronted)).Error(s.err)
			ui.Field(c, "Count a press", func() {
				shortcut.Field(c, &s.local)
			}).Description(fmt.Sprintf("While this window has the keyboard. Pressed %d times.", s.pressed))
		})
		ui.Text(c, "Click a field, or press Enter or Space in it, then press the keys. Escape cancels; Backspace clears.").
			FontSize(12).TextColor(t.TextMuted)
	})
}

func main() {
	global, _ := shortcut.Parse("Alt+Shift+M")
	local, _ := shortcut.Parse("CmdOrCtrl+K")
	s := &settings{global: global, local: local}
	mygo.App.WhenReady(func() {
		s.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Shortcuts",
			Width:   520,
			Height:  300,
			Content: ui.View(s.view),
		})
		s.register()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

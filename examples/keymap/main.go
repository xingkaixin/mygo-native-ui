// Keymap maps the shortcuts of a code editor on the keyboard, two of them
// bound twice by mistake, and lists the shortcuts of the key chosen.
//
//	go run ./examples/keymap
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/keymap"
	"github.com/xingkaixin/mygo-native-ui/ui/shortcut"
)

// groups are the shortcuts of the editor by group, two of them bound twice:
// Cmd+D, as Ctrl+D elsewhere, and F12.
var groups = []struct {
	name     string
	bindings [][2]string
}{
	{"Editing", [][2]string{
		{"CmdOrCtrl+Z", "Undo"},
		{"CmdOrCtrl+Shift+Z", "Redo"},
		{"CmdOrCtrl+X", "Cut"},
		{"CmdOrCtrl+C", "Copy"},
		{"CmdOrCtrl+V", "Paste"},
		{"CmdOrCtrl+A", "Select All"},
		{"CmdOrCtrl+S", "Save"},
		{"CmdOrCtrl+/", "Toggle Line Comment"},
		{"Alt+Shift+A", "Toggle Block Comment"},
		{"CmdOrCtrl+Shift+K", "Delete Line"},
		{"Alt+Up", "Move Line Up"},
		{"Alt+Down", "Move Line Down"},
		{"CmdOrCtrl+]", "Indent Line"},
		{"CmdOrCtrl+[", "Outdent Line"},
		{"CmdOrCtrl+Enter", "Insert Line Below"},
		{"CmdOrCtrl+D", "Duplicate Line"},
		{"Ctrl+Space", "Trigger Suggest"},
		{"Alt+Shift+F", "Format Document"},
		{"F2", "Rename Symbol"},
		{"CmdOrCtrl+.", "Quick Fix"},
	}},
	{"Navigation", [][2]string{
		{"CmdOrCtrl+P", "Go to File"},
		{"Ctrl+G", "Go to Line"},
		{"CmdOrCtrl+Shift+O", "Go to Symbol"},
		{"F12", "Go to Definition"},
		{"Alt+F12", "Peek Definition"},
		{"Shift+F12", "Go to References"},
		{"Alt+Left", "Go Back"},
		{"Alt+Right", "Go Forward"},
		{"F8", "Next Problem"},
		{"Shift+F8", "Previous Problem"},
		{"CmdOrCtrl+Home", "Go to Start of File"},
		{"CmdOrCtrl+End", "Go to End of File"},
		{"Ctrl+Tab", "Next Editor"},
		{"Ctrl+Shift+Tab", "Previous Editor"},
	}},
	{"Search", [][2]string{
		{"CmdOrCtrl+F", "Find"},
		{"CmdOrCtrl+Alt+F", "Replace"},
		{"F3", "Find Next"},
		{"Shift+F3", "Find Previous"},
		{"CmdOrCtrl+Shift+F", "Find in Files"},
		{"CmdOrCtrl+Shift+H", "Replace in Files"},
		{"CmdOrCtrl+D", "Add Next Occurrence"},
		{"CmdOrCtrl+Shift+L", "Select All Occurrences"},
	}},
	{"View", [][2]string{
		{"CmdOrCtrl+Shift+P", "Command Palette"},
		{"CmdOrCtrl+B", "Toggle Sidebar"},
		{"CmdOrCtrl+J", "Toggle Panel"},
		{"Ctrl+`", "Toggle Terminal"},
		{"CmdOrCtrl+Shift+E", "Show Explorer"},
		{"CmdOrCtrl+\\", "Split Editor"},
		{"CmdOrCtrl+=", "Zoom In"},
		{"CmdOrCtrl+-", "Zoom Out"},
		{"Alt+Z", "Toggle Word Wrap"},
		{"F12", "Toggle Developer Tools"},
	}},
	{"Debug", [][2]string{
		{"F5", "Start Debugging"},
		{"Shift+F5", "Stop Debugging"},
		{"CmdOrCtrl+Shift+F5", "Restart Debugging"},
		{"F6", "Pause"},
		{"F9", "Toggle Breakpoint"},
		{"F10", "Step Over"},
		{"F11", "Step Into"},
		{"Shift+F11", "Step Out"},
		{"CmdOrCtrl+Shift+Y", "Debug Console"},
	}},
}

// bindings are the shortcuts of groups, in order.
var bindings = func() []keymap.Binding {
	var out []keymap.Binding
	for _, g := range groups {
		for _, b := range g.bindings {
			out = append(out, keymap.Binding{Keys: b[0], Command: b[1], Group: g.name})
		}
	}
	return out
}()

type app struct {
	keymap keymap.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(24).Gap(16).Background(t.Background).Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, "Keyboard Shortcuts").FontSize(16).Bold()
			ui.Text(c, "Turn on a modifier key, or hold one, to see what it does with each key. Hover a key for all its shortcuts; click it to list them below.").
				FontSize(12).TextColor(t.TextMuted)
		})
		keymap.View(c, &a.keymap, bindings)
		k, ok := a.keymap.Chosen()
		if !ok {
			ui.Text(c, "No key chosen").TextColor(t.TextMuted)
			return
		}
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, "Shortcuts with "+shortcut.Shortcut{Key: k}.MenuText()).Bold()
			n := 0
			for i, b := range bindings {
				sc, err := shortcut.Parse(b.Keys)
				if err != nil || sc.Key != k {
					continue
				}
				n++
				ui.Row(c.Key(i)).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Width(110).Children(func() { shortcut.Keys(c, sc) })
					ui.Text(c, b.Command).Width(220)
					ui.Text(c, b.Group).TextColor(t.TextMuted)
				})
			}
			if n == 0 {
				ui.Text(c, "None: the key is free with any modifiers.").TextColor(t.TextMuted)
			}
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Keymap",
			Width:   1160,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

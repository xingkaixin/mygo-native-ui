// Package shortcut records keyboard shortcuts and shows them as keys, as
// the shortcut fields of macOS's settings, Raycast and Alfred do:
//
//	if shortcut.Field(c, &app.hotkey).Label("Show launcher").Changed() {
//		mygo.GlobalShortcut.Unregister(app.registered)
//		app.registered = app.hotkey.String()
//		mygo.GlobalShortcut.Register(app.registered, app.toggle)
//	}
//
// A Shortcut is a key and the modifiers held with it. String writes it as
// an accelerator, as menus and mygo.GlobalShortcut take, and Parse reads
// one back, as from the app's settings; Pressed reports it pressed in the
// window, as Context.Shortcut does. Keys shows it as the keys to press,
// ⇧⌘K on macOS and Ctrl Shift K elsewhere.
package shortcut

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Shortcut is a key pressed with modifiers. The zero Shortcut is none.
type Shortcut struct {
	Mods ui.Modifiers
	Key  ui.Key
}

// Pressed reports whether s was pressed anywhere in the window, as
// Context.Shortcut reports it; the zero Shortcut never is.
func (s Shortcut) Pressed(c *ui.Context) bool {
	return s.Key != ui.KeyUnknown && c.Shortcut(s.Mods, s.Key)
}

// String returns s as an accelerator, as menus and mygo.GlobalShortcut
// take: "Cmd+Shift+K", with Super as Cmd on macOS. It is "" for the zero
// Shortcut.
func (s Shortcut) String() string {
	name := keys[s.Key].name
	if name == "" {
		return ""
	}
	var parts []string
	for _, m := range accelOrder {
		if s.Mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, name), "+")
}

// Parse reads an accelerator such as "CmdOrCtrl+Shift+K", as String writes
// them and menus take them: modifiers by any of their names (Cmd, Command,
// Super, Meta, Win; Ctrl, Control; CmdOrCtrl, CommandOrControl, which is
// Cmd on macOS and Ctrl elsewhere; Alt, Option; Shift), case aside, and a
// key that a Shortcut can hold. "" is the zero Shortcut.
func Parse(s string) (Shortcut, error) {
	if s == "" {
		return Shortcut{}, nil
	}
	parts := strings.Split(s, "+")
	var out Shortcut
	for _, p := range parts[:len(parts)-1] {
		m, ok := modNames[strings.ToLower(strings.TrimSpace(p))]
		if !ok {
			return Shortcut{}, fmt.Errorf("shortcut: unknown modifier %q in %q", p, s)
		}
		out.Mods |= m
	}
	k, ok := keyNames[strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))]
	if !ok {
		return Shortcut{}, fmt.Errorf("shortcut: no key %q in %q", parts[len(parts)-1], s)
	}
	out.Key = k
	return out, nil
}

// Keys creates a row showing s as the keys to press, each on a key cap:
// the modifiers as macOS's symbols in its order, ⌃⌥⇧⌘, then the key, and
// elsewhere their names, Win Ctrl Alt Shift. Assistive technology reads the
// row as one text naming them, "Command Shift K". It shows nothing for the
// zero Shortcut.
func Keys(c *ui.Context, s Shortcut) ui.Element {
	t := c.Theme()
	caps := labels(s)
	row := ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Label(spoken(s))
	row.Children(func() {
		for _, k := range caps {
			ui.Text(c, k).NoWrap().FontSize(t.FontSize-1).TextAlign(ui.Center).
				MinWidth(t.Space(5)).Padding(1, t.Space(1.5)).Radius(t.Space(1)).
				Background(t.Background).Border(1, t.Border).TextColor(t.TextMuted)
		}
	})
	return row
}

// MenuText returns s as a menu writes it beside an item: ⇧⌘K on macOS,
// Ctrl+Shift+K elsewhere, and "" for the zero Shortcut.
func (s Shortcut) MenuText() string {
	sep := "+"
	if mac {
		sep = ""
	}
	return strings.Join(labels(s), sep)
}

// recorder is what a field keeps from frame to frame: whether it records,
// and the shortcut its input recorded since the last frame, to write
// through the field's pointer as it builds.
type recorder struct {
	recording bool
	pending   Shortcut
	recorded  bool
}

// Field creates a field recording a shortcut into *s. A click on it, or
// Enter or Space while it has the focus, starts recording, and the field
// then shows the modifiers held as they go down; a key pressed with Cmd,
// Ctrl, Alt or Super, or a function key alone, is recorded, and Changed
// reports it. While it records, Escape stops without changing *s, and
// Backspace or Delete clears it, as does the field's clear button; Tab
// moves on, which stops it, as a click elsewhere does. Keys typing text
// with Shift alone or no modifier are left out, as they would type.
//
// Keys that the window takes before the focused element never reach the
// field, which cannot record them: those of its menus, and those the view
// handles with Context.Shortcut. Name the field with Label.
func Field(c *ui.Context, s *Shortcut) ui.Element {
	t := c.Theme()
	field := ui.Row(c).Focusable().Role(ui.RoleButton).Gap(t.Space(1.5)).AlignItems(ui.Center).
		MinWidth(t.Space(40)).Height(t.Space(8)).PaddingStart(t.Space(2)).PaddingEnd(t.Space(1)).
		Radius(t.Radius).Background(t.Surface).Border(1, t.Border)
	st := ui.Local(field, "shortcut.Field", func() recorder { return recorder{} })

	if st.recorded {
		st.recorded = false
		if *s != st.pending {
			*s = st.pending
			field.SetChanged()
		}
	}
	if field.Clicked() {
		st.recording = !st.recording
	}
	if st.recording && !field.Focused() {
		st.recording = false
	}
	recording := st.recording

	field.HandleInput(func(ev ui.InputEvent) bool {
		if !st.recording {
			return false
		}
		switch ev.Kind {
		case ui.InputText, ui.InputCompose, ui.InputKeyUp:
			return true
		case ui.InputKeyDown:
		default:
			return false
		}
		key := ev.Key
		if key == ui.KeyUnknown {
			// On a layout typing no Latin letters, the key at the place
			// of one on a US keyboard, as Context.Shortcut takes it.
			key = ev.PhysicalKey
		}
		switch {
		case ev.Mods == 0 && key == ui.KeyTab:
			return false
		case ev.Repeat:
			return true
		case ev.Mods == 0 && key == ui.KeyEscape:
			st.recording = false
		case ev.Mods == 0 && (key == ui.KeyBackspace || key == ui.KeyDelete):
			st.recording, st.pending, st.recorded = false, Shortcut{}, true
		case recordable(ev.Mods, key):
			st.recording, st.pending, st.recorded = false, Shortcut{Mods: ev.Mods, Key: key}, true
		}
		return true
	})

	field.Value(spoken(*s))
	if recording {
		field.Border(1, t.Accent).Description("Recording: press a shortcut, Escape to cancel")
	}
	field.Children(func() {
		switch {
		case recording && c.Modifiers() != 0:
			Keys(c, Shortcut{Mods: c.Modifiers(), Key: ui.KeyUnknown}).Grow(1)
		case recording:
			ui.Text(c, "Type shortcut").Grow(1).TextColor(t.TextMuted)
		case s.Key == ui.KeyUnknown:
			ui.Text(c, "Record Shortcut").Grow(1).TextColor(t.TextMuted)
		default:
			Keys(c, *s).Grow(1)
		}
		if recording || s.Key == ui.KeyUnknown {
			return
		}
		x := ui.Box(c).Size(t.Space(5), t.Space(5)).Radius(999).Role(ui.RoleButton).Label("Clear shortcut").KeepFocus()
		if x.Clicked() {
			*s = Shortcut{}
			field.SetChanged()
		}
		if x.Hovered() {
			x.Background(t.SurfaceHover)
		}
		x.Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/6
			var path ui.Path
			path.MoveTo(cx-k, cy-k).LineTo(cx+k, cy+k).MoveTo(cx+k, cy-k).LineTo(cx-k, cy+k)
			p.StrokePath(&path, 1.5, t.TextMuted)
		})
	})
	return field
}

// recordable reports whether key pressed with mods makes a shortcut: one
// with a modifier other than Shift, or a function key.
func recordable(mods ui.Modifiers, key ui.Key) bool {
	k, ok := keys[key]
	if !ok {
		return false
	}
	return mods&^ui.Shift != 0 || k.function
}

// key is how a key is written: its name in accelerators, its cap on macOS
// and elsewhere, and whether it is a function key, which makes a shortcut
// alone.
type key struct {
	name, mac, other string
	function         bool
}

// keys are the keys a Shortcut can hold.
var keys = func() map[ui.Key]key {
	m := map[ui.Key]key{
		ui.KeyEnter:        {name: "Enter", mac: "↩", other: "Enter"},
		ui.KeyEscape:       {name: "Escape", mac: "⎋", other: "Esc"},
		ui.KeyBackspace:    {name: "Backspace", mac: "⌫", other: "Backspace"},
		ui.KeyDelete:       {name: "Delete", mac: "⌦", other: "Delete"},
		ui.KeyTab:          {name: "Tab", mac: "⇥", other: "Tab"},
		ui.KeySpace:        {name: "Space", mac: "Space", other: "Space"},
		ui.KeyInsert:       {name: "Insert", mac: "Ins", other: "Ins"},
		ui.KeyHome:         {name: "Home", mac: "↖", other: "Home"},
		ui.KeyEnd:          {name: "End", mac: "↘", other: "End"},
		ui.KeyPageUp:       {name: "PageUp", mac: "⇞", other: "PgUp"},
		ui.KeyPageDown:     {name: "PageDown", mac: "⇟", other: "PgDn"},
		ui.KeyLeft:         {name: "Left", mac: "←", other: "←"},
		ui.KeyRight:        {name: "Right", mac: "→", other: "→"},
		ui.KeyUp:           {name: "Up", mac: "↑", other: "↑"},
		ui.KeyDown:         {name: "Down", mac: "↓", other: "↓"},
		ui.KeyMinus:        {name: "-"},
		ui.KeyEqual:        {name: "="},
		ui.KeyComma:        {name: ","},
		ui.KeyPeriod:       {name: "."},
		ui.KeySlash:        {name: "/"},
		ui.KeySemicolon:    {name: ";"},
		ui.KeyQuote:        {name: "'"},
		ui.KeyBracketLeft:  {name: "["},
		ui.KeyBracketRight: {name: "]"},
		ui.KeyBackslash:    {name: `\`},
		ui.KeyBackquote:    {name: "`"},
	}
	for i := range 26 {
		m[ui.KeyA+ui.Key(i)] = key{name: string(rune('A' + i))}
	}
	for i := range 10 {
		m[ui.Key0+ui.Key(i)] = key{name: string(rune('0' + i))}
	}
	for i := range 12 {
		m[ui.KeyF1+ui.Key(i)] = key{name: fmt.Sprintf("F%d", i+1), function: true}
	}
	for k, v := range m {
		if v.mac == "" {
			v.mac, v.other = v.name, v.name
			m[k] = v
		}
	}
	return m
}()

// keyNames are the keys by their names in accelerators, in lower case,
// with the other names menus take for some.
var keyNames = func() map[string]ui.Key {
	m := map[string]ui.Key{"return": ui.KeyEnter, "esc": ui.KeyEscape, "del": ui.KeyDelete, "ins": ui.KeyInsert}
	for k, v := range keys {
		m[strings.ToLower(v.name)] = k
	}
	return m
}()

// modNames are the modifiers by their names in accelerators, in lower case.
var modNames = map[string]ui.Modifiers{
	"cmd": ui.Super, "command": ui.Super, "super": ui.Super, "meta": ui.Super, "win": ui.Super,
	"ctrl": ui.Ctrl, "control": ui.Ctrl,
	"cmdorctrl": ui.Cmd, "commandorcontrol": ui.Cmd,
	"alt": ui.Alt, "option": ui.Alt,
	"shift": ui.Shift,
}

// modifier is how a modifier is written: in accelerators, on its cap, and
// read aloud.
type modifier struct {
	mod              ui.Modifiers
	name, cap, spoke string
}

var (
	mac = runtime.GOOS == "darwin"
	// superName is Super's name in accelerators and on its cap.
	superName = func() string {
		switch runtime.GOOS {
		case "darwin":
			return "Cmd"
		case "windows":
			return "Win"
		}
		return "Super"
	}()
	// accelOrder is the order of the modifiers in accelerators, as mygo
	// writes them.
	accelOrder = []modifier{{mod: ui.Super, name: superName}, {mod: ui.Ctrl, name: "Ctrl"}, {mod: ui.Alt, name: "Alt"}, {mod: ui.Shift, name: "Shift"}}
	// capOrder is the order of the modifiers on caps: Apple's on macOS,
	// and with Win first elsewhere, as Windows writes Win+Shift+S.
	capOrder = func() []modifier {
		if mac {
			return []modifier{
				{mod: ui.Ctrl, cap: "⌃", spoke: "Control"}, {mod: ui.Alt, cap: "⌥", spoke: "Option"},
				{mod: ui.Shift, cap: "⇧", spoke: "Shift"}, {mod: ui.Super, cap: "⌘", spoke: "Command"},
			}
		}
		return []modifier{
			{mod: ui.Super, cap: superName, spoke: superName}, {mod: ui.Ctrl, cap: "Ctrl", spoke: "Control"},
			{mod: ui.Alt, cap: "Alt", spoke: "Alt"}, {mod: ui.Shift, cap: "Shift", spoke: "Shift"},
		}
	}()
)

// labels returns the caps of s: its modifiers, then its key, if it has one.
func labels(s Shortcut) []string {
	var out []string
	for _, m := range capOrder {
		if s.Mods&m.mod != 0 {
			out = append(out, m.cap)
		}
	}
	if k, ok := keys[s.Key]; ok {
		if mac {
			out = append(out, k.mac)
		} else {
			out = append(out, k.other)
		}
	}
	return out
}

// spoken returns s as assistive technology reads it: "Command Shift K".
func spoken(s Shortcut) string {
	var out []string
	for _, m := range capOrder {
		if s.Mods&m.mod != 0 {
			out = append(out, m.spoke)
		}
	}
	if k, ok := keys[s.Key]; ok {
		out = append(out, k.name)
	}
	return strings.Join(out, " ")
}

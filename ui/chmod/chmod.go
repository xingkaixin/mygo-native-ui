// Package chmod edits the permissions of a Unix file as a permissions
// dialog does: a tile for each of read, write and execute for the owner,
// the group and others, each row's octal digit beside it, and setuid,
// setgid and sticky under them. A click on a tile turns it on or off,
// and a click on a row's or a column's name turns all of it on, or off
// when it is all on. The same mode shows as the octal number chmod
// takes, which can be typed to set the tiles, as ls -l writes it, and as
// the chmod commands that set it, both octal and symbolic. Write access
// for others is called out. The editor changes the app's mode in place
// and reports the changes:
//
//	if chmod.View(c, &app.mode, "deploy.sh").Changed() {
//		os.Chmod(path, app.mode)
//	}
package chmod

import (
	"io/fs"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

var (
	whos  = [...]string{"Owner", "Group", "Others"}
	whats = [...]string{"Read", "Write", "Execute"}
)

// bit returns the permission bit of who, 0 the owner, for what, 0 read.
func bit(who, what int) fs.FileMode { return 1 << (8 - 3*who - what) }

// special lists the special bits, as their octal digit's bits, in order.
var special = [...]struct {
	name  string
	mode  fs.FileMode
	octal uint64
}{
	{"Set user ID", fs.ModeSetuid, 04000},
	{"Set group ID", fs.ModeSetgid, 02000},
	{"Sticky", fs.ModeSticky, 01000},
}

// Octal returns the permissions of m as the octal number chmod takes,
// with a fourth digit in front when a special bit is set: 755, 4755.
func Octal(m fs.FileMode) string {
	n := uint64(m.Perm())
	for _, sp := range special {
		if m&sp.mode != 0 {
			n |= sp.octal
		}
	}
	s := strconv.FormatUint(n, 8)
	return strings.Repeat("0", max(0, 3-len(s))) + s
}

// Parse returns m with the permissions of octal number s, of three or
// four digits.
func Parse(s string, m fs.FileMode) (fs.FileMode, bool) {
	if len(s) < 3 || len(s) > 4 {
		return m, false
	}
	n, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return m, false
	}
	out := m&fs.ModeType | fs.FileMode(n&0o777)
	for _, sp := range special {
		if n&sp.octal != 0 {
			out |= sp.mode
		}
	}
	return out, true
}

// List returns m as ls -l writes it: -rwxr-xr-x, with s, S, t and T for
// the special bits.
func List(m fs.FileMode) string {
	b := []byte("----------")
	switch {
	case m.IsDir():
		b[0] = 'd'
	case m&fs.ModeSymlink != 0:
		b[0] = 'l'
	}
	for who := range 3 {
		for what, ch := range "rwx" {
			if m&bit(who, what) != 0 {
				b[1+3*who+what] = byte(ch)
			}
		}
	}
	for i, sp := range special {
		if m&sp.mode == 0 {
			continue
		}
		at := 3 + 3*i
		on := "sst"[i]
		if b[at] == '-' {
			on -= 'a' - 'A'
		}
		b[at] = on
	}
	return string(b)
}

// Symbolic returns the permissions of m as chmod's symbolic form:
// u=rwx,g=rx,o=rx, with s and t for the special bits.
func Symbolic(m fs.FileMode) string {
	parts := make([]string, 3)
	for who, w := range "ugo" {
		s := string(w) + "="
		for what, ch := range "rwx" {
			if m&bit(who, what) != 0 {
				s += string(ch)
			}
		}
		if m&special[who].mode != 0 {
			s += string("sst"[who])
		}
		parts[who] = s
	}
	return strings.Join(parts, ",")
}

// local is what the editor keeps between frames: the octal number as
// typed, and the mode it last showed, to tell a change from elsewhere.
type local struct {
	text string
	last fs.FileMode
}

// View creates the editor of *mode, the mode of the file called name,
// which it changes in place, reporting the changes.
func View(c *ui.Context, mode *fs.FileMode, name string) ui.Element {
	t := c.Theme()
	m := *mode
	root := ui.Column(c).Gap(t.Space(6)).AlignItems(ui.Start).Role(ui.RoleGroup).Label("Permissions of " + name)
	st := ui.Local(root, "chmod.state", func() local { return local{Octal(m), m} })
	changed := false
	set := func(n fs.FileMode) {
		if n != m {
			m, changed = n, true
		}
	}
	all := func(bits ...fs.FileMode) {
		var mask fs.FileMode
		for _, b := range bits {
			mask |= b
		}
		if m&mask == mask {
			set(m &^ mask)
		} else {
			set(m | mask)
		}
	}
	tile := t.Space(12)
	root.Children(func() {
		ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fixed(tile), ui.Fixed(tile), ui.Fixed(tile), ui.Fixed(t.Space(10))).
			GapX(t.Space(2)).GapY(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Box(c)
			for what, w := range whats {
				var col []fs.FileMode
				for who := range whos {
					col = append(col, bit(who, what))
				}
				if header(c.Key(w), w, ui.Center).Clicked() {
					all(col...)
				}
			}
			ui.Box(c)
			for who, w := range whos {
				if header(c.Key(w), w, ui.Start).Clicked() {
					all(bit(who, 0), bit(who, 1), bit(who, 2))
				}
				for what := range whats {
					b := bit(who, what)
					on := m&b != 0
					e := ui.ButtonBase(c.Key(who*3+what)).Size(tile, tile).Radius(t.Space(2)).Center().
						Checked(on).Label(w + " " + whats[what])
					if e.Clicked() {
						set(m ^ b)
					}
					e.Children(func() {
						l := ui.Text(c, string("rwx"[what])).Font("monospace").FontSize(t.FontSize + 4).FontWeight(700)
						if on {
							e.Background(t.Accent)
							l.TextColor(t.AccentText)
						} else {
							e.Background(t.Surface).Border(1, t.Border)
							l.TextColor(t.TextMuted.Alpha(0.6))
						}
					})
				}
				digit := strconv.Itoa(int(m.Perm() >> (6 - 3*who) & 7))
				ui.Text(c.Key(-1-who), digit).Font("monospace").FontSize(t.FontSize + 6).FontWeight(600).TextAlign(ui.Center).TextColor(t.TextMuted)
			}
		})
		ui.Row(c).Gap(t.Space(5)).Children(func() {
			for i, sp := range special {
				on := m&sp.mode != 0
				if ui.Checkbox(c.Key(i), &on, sp.name).Changed() {
					set(m ^ sp.mode)
				}
			}
		})
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
				if m != st.last {
					st.text, st.last = Octal(m), m
				}
				in := ui.TextInput(c, &st.text).Width(t.Space(18)).Font("monospace").FontSize(t.FontSize + 2).Label("Octal")
				if in.Changed() {
					if n, ok := Parse(st.text, m); ok {
						set(n)
						st.last = n
						c.Invalidate()
					}
				}
				if !in.Focused() && st.text != Octal(m) {
					st.text = Octal(m)
				}
				ls := List(m)
				ui.Row(c).Role(ui.RoleText).Label(ls).Children(func() {
					for i, ch := range ls {
						l := ui.Text(c.Key(i), string(ch)).Font("monospace").FontSize(t.FontSize + 4)
						if ch == '-' {
							l.TextColor(t.TextMuted.Alpha(0.6))
						} else {
							l.FontWeight(600)
						}
					}
				})
				ui.Text(c, name).Font("monospace").FontSize(t.FontSize + 4)
			})
			for _, arg := range []string{Octal(m), Symbolic(m)} {
				ui.Text(c.Key(arg), "chmod "+arg+" "+name).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted).Selectable()
			}
			if m&bit(2, 1) != 0 {
				ui.Text(c, "Anyone on this computer can change "+name+".").FontSize(t.FontSize - 1).TextColor(t.Danger)
			}
		})
	})
	if changed {
		*mode = m
		root.SetChanged()
	}
	return root
}

// header creates the name of a row or a column, a click on which turns
// all of it on or off.
func header(c *ui.Context, name string, align ui.Align) ui.Element {
	t := c.Theme()
	e := ui.ButtonBase(c).Padding(t.Space(1), t.Space(1.5)).Radius(t.Space(1)).Justify(align).Label("All " + name)
	e.Children(func() {
		ui.Text(c, name).FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.TextMuted)
	})
	return e
}

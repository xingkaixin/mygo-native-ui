// Bitfield edits registers bit by bit: a Unix file mode, with its octal
// form and the string ls shows for it beside it, the first word of an
// IPv4 header, and the control register of a made-up UART, whose fields
// name their values.
//
//	go run ./examples/bitfield
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/bitfield"
)

// register is a register to edit: what it is, its width and fields, its
// value, and the state of its view.
type register struct {
	about  string
	bits   int
	fields []bitfield.Field
	value  uint64
	state  bitfield.State
}

// app is the state the window shows: the registers and which one shows.
type app struct {
	regs   [3]*register
	chosen int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(20).Gap(16).Children(func() {
		ui.Segmented(c, &a.chosen, "File mode", "IPv4 header", "UART control").Label("Register").AlignSelf(ui.Start)
		r := a.regs[a.chosen]
		ui.Text(c, r.about).TextColor(t.TextMuted)
		ui.Row(c).Gap(32).AlignItems(ui.Start).Children(func() {
			bitfield.View(c.Key(a.chosen), &r.state, &r.value, r.bits, r.fields)
			if a.chosen == 0 {
				ui.Column(c).Gap(16).Children(func() {
					shown(c, "Octal", fmt.Sprintf("%04o", r.value&0o7777))
					shown(c, "ls -l", ls(r.value))
				})
			}
		})
	})
}

// shown shows text large under a label, as the fields label theirs.
func shown(c *ui.Context, label, text string) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, label).FontWeight(500)
		ui.Text(c, text).Font("monospace").FontSize(20).Selectable()
	})
}

// ls returns mode as ls -l writes it: the type, then read, write and
// execute for the user, the group and others, with setuid, setgid and
// sticky in place of an execute, in capitals where that is off.
func ls(mode uint64) string {
	types := map[uint64]byte{0o14: 's', 0o12: 'l', 0o10: '-', 0o06: 'b', 0o04: 'd', 0o02: 'c', 0o01: 'p'}
	b := []byte("?---------")
	if ch, ok := types[mode>>12]; ok {
		b[0] = ch
	}
	for i, ch := range []byte("rwxrwxrwx") {
		if mode&(1<<(8-i)) != 0 {
			b[1+i] = ch
		}
	}
	for _, sp := range []struct {
		bit uint64
		at  int
		ch  byte
	}{{0o4000, 3, 's'}, {0o2000, 6, 's'}, {0o1000, 9, 't'}} {
		switch {
		case mode&sp.bit == 0:
		case b[sp.at] == 'x':
			b[sp.at] = sp.ch
		default:
			b[sp.at] = sp.ch - 'a' + 'A'
		}
	}
	return string(b)
}

// newApp returns the app with the registers to edit.
func newApp() *app {
	rwx := map[uint64]string{}
	for v := range uint64(8) {
		rwx[v] = ls(v)[7:]
	}
	words := map[uint64]string{}
	for v := range uint64(16) {
		if v >= 5 {
			words[v] = fmt.Sprintf("%d bytes", v*4)
		}
	}
	onOff := map[uint64]string{0: "Off", 1: "On"}
	return &app{regs: [3]*register{
		{
			about: "st_mode of a file, as stat(2) returns it: its type, then its permissions.",
			bits:  16,
			value: 0o100755,
			fields: []bitfield.Field{
				{Name: "type", Lsb: 12, Width: 4, Values: map[uint64]string{
					0o01: "FIFO", 0o02: "char device", 0o04: "directory", 0o06: "block device",
					0o10: "regular", 0o12: "symlink", 0o14: "socket",
				}},
				{Name: "setuid", Lsb: 11, Width: 1},
				{Name: "setgid", Lsb: 10, Width: 1},
				{Name: "sticky", Lsb: 9, Width: 1},
				{Name: "user", Lsb: 6, Width: 3, Values: rwx},
				{Name: "group", Lsb: 3, Width: 3, Values: rwx},
				{Name: "other", Lsb: 0, Width: 3, Values: rwx},
			},
		},
		{
			about: "The first 32-bit word of an IPv4 header, as it goes on the wire.",
			bits:  32,
			value: 0x450005DC,
			fields: []bitfield.Field{
				{Name: "Version", Lsb: 28, Width: 4, Values: map[uint64]string{4: "IPv4", 6: "IPv6"}},
				{Name: "IHL", Lsb: 24, Width: 4, Values: words},
				{Name: "DSCP", Lsb: 18, Width: 6, Values: map[uint64]string{
					0: "Default", 8: "CS1", 10: "AF11", 16: "CS2", 18: "AF21", 24: "CS3", 26: "AF31",
					32: "CS4", 34: "AF41", 40: "CS5", 46: "EF", 48: "CS6", 56: "CS7",
				}},
				{Name: "ECN", Lsb: 16, Width: 2, Values: map[uint64]string{0: "Not-ECT", 1: "ECT(1)", 2: "ECT(0)", 3: "CE"}},
				{Name: "Total Length", Lsb: 0, Width: 16},
			},
		},
		{
			about: "UART0_CR, the control register of a made-up UART. Bits of no field are reserved.",
			bits:  32,
			value: 0x401A_10C1,
			fields: []bitfield.Field{
				{Name: "IE", Lsb: 30, Width: 1, Values: onOff},
				{Name: "CLK", Lsb: 28, Width: 2, Values: map[uint64]string{0: "APB", 1: "HSI", 2: "LSE", 3: "PLL"}},
				{Name: "BRD", Lsb: 16, Width: 12},
				{Name: "OVS", Lsb: 12, Width: 2, Values: map[uint64]string{0: "16×", 1: "8×", 2: "4×"}},
				{Name: "LEN", Lsb: 6, Width: 2, Values: map[uint64]string{0: "5 bits", 1: "6 bits", 2: "7 bits", 3: "8 bits"}},
				{Name: "STOP", Lsb: 5, Width: 1, Values: map[uint64]string{0: "1 bit", 1: "2 bits"}},
				{Name: "PAR", Lsb: 3, Width: 2, Values: map[uint64]string{0: "None", 1: "Even", 2: "Odd", 3: "Mark"}},
				{Name: "MODE", Lsb: 1, Width: 2, Values: map[uint64]string{0: "Normal", 1: "Loopback", 2: "IrDA", 3: "RS-485"}},
				{Name: "EN", Lsb: 0, Width: 1, Values: onOff},
			},
		},
	}}
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Registers",
			Width:   860,
			Height:  440,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

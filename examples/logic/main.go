// Logic shows a synthesized SPI capture of a W25Q128 flash at 4 MHz: a
// Read Data of eight bytes from 0x001000, a Read JEDEC ID and a Read
// Status Register, with the bytes on MOSI and MISO decoded below the
// wires. Under it, a WaveJSON diagram of reads on a valid/ready bus draws
// from the text beside it as it is typed, with what is wrong in it. Click
// to place cursor A and Shift+click for B to read the time between them;
// the wheel zooms and dragging pans.
//
//	go run ./examples/logic
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/logic"
)

const handshake = `// Two reads on a valid/ready bus, 10 ns a cycle
{signal: [
  {name: 'clk',   wave: 'p......|......'},
  {name: 'rst_n', wave: '01.....|......'},
  ['Request',
   {name: 'valid', wave: '0.1..0.|1..0..'},
   {name: 'ready', wave: 'x.0.10.|..10..'},
   {name: 'addr',  wave: 'x.3..x.|4..x..',
    data: ['0x40', '0x44']},
  ],
  {},
  ['Response',
   {name: 'rvalid', wave: '0....10|...10.'},
   {name: 'rdata',  wave: 'z....=z|...=z.',
    data: 'BEEF F00D'},
  ],
  {name: 'clk/2', wave: 'p......', period: 2},
]}
`

// sclk is how long a cycle of SPI's clock lasts: 4 MHz.
const sclk = 250e-9

// word is a byte of an SPI exchange: what the controller sends on MOSI,
// and what the flash answers on MISO, or -1 while it listens; with their
// names, where they have one.
type word struct {
	mosi               byte
	miso               int
	mosiName, misoName string
}

// spi is a capture being synthesized: the four wires, and the bytes
// decoded from MOSI and MISO.
type spi struct {
	cs, sck, mosi, miso, out, in logic.Signal
}

// set adds a change to a wire, unless the wire is at it already.
func set(sig *logic.Signal, ch logic.Change) {
	if n := len(sig.Changes); n > 0 && sig.Changes[n-1].Value == ch.Value && sig.Changes[n-1].Level == ch.Level {
		return
	}
	sig.Changes = append(sig.Changes, ch)
}

// exchange adds an exchange from at, in SPI mode 0: CS# falls, words go
// out most significant bit first, each bit set on MOSI as SCLK falls and
// taken as it rises, the flash answering 7 ns after each fall, and CS#
// rises half a cycle after the last. It returns when CS# rises.
func (s *spi) exchange(at float64, words []word) float64 {
	set(&s.cs, logic.Change{At: at})
	t := at
	for _, w := range words {
		s.out.Changes = append(s.out.Changes, logic.Change{At: t, Value: uint64(w.mosi), Label: w.mosiName})
		if w.miso < 0 {
			set(&s.in, logic.Change{At: t, Level: logic.Floating})
		} else {
			s.in.Changes = append(s.in.Changes, logic.Change{At: t, Value: uint64(w.miso), Label: w.misoName})
		}
		for bit := 7; bit >= 0; bit-- {
			set(&s.mosi, logic.Change{At: t, Value: uint64(w.mosi>>bit) & 1})
			if w.miso >= 0 {
				set(&s.miso, logic.Change{At: t + 7e-9, Value: uint64(w.miso>>bit) & 1})
			}
			set(&s.sck, logic.Change{At: t + sclk/2, Value: 1})
			set(&s.sck, logic.Change{At: t + sclk})
			t += sclk
		}
	}
	end := t + sclk/2
	set(&s.cs, logic.Change{At: end, Value: 1})
	set(&s.miso, logic.Change{At: end + 7e-9, Level: logic.Floating})
	set(&s.out, logic.Change{At: end, Level: logic.Floating})
	set(&s.in, logic.Change{At: end, Level: logic.Floating})
	return end
}

// flash synthesizes the capture: the controller drives MOSI 1.5 µs in,
// then reads a firmware header, the flash's JEDEC ID and its status.
func flash() logic.Capture {
	s := spi{
		cs:   logic.Signal{Name: "CS#", Bits: 1},
		sck:  logic.Signal{Name: "SCLK", Bits: 1},
		mosi: logic.Signal{Name: "MOSI", Bits: 1},
		miso: logic.Signal{Name: "MISO", Bits: 1},
		out:  logic.Signal{Name: "MOSI data", Bits: 8},
		in:   logic.Signal{Name: "MISO data", Bits: 8},
	}
	set(&s.cs, logic.Change{Value: 1})
	set(&s.sck, logic.Change{})
	set(&s.mosi, logic.Change{Level: logic.Unknown})
	set(&s.mosi, logic.Change{At: 1.5e-6})
	for _, sig := range []*logic.Signal{&s.miso, &s.out, &s.in} {
		set(sig, logic.Change{Level: logic.Floating})
	}

	read := []word{{0x03, -1, "READ", ""}, {0x00, -1, "", ""}, {0x10, -1, "", ""}, {0x00, -1, "", ""}}
	for _, b := range []byte("MYGO") {
		read = append(read, word{0xFF, int(b), "", "'" + string(b) + "'"})
	}
	read = append(read, word{0xFF, 0x01, "", ""}, word{0xFF, 0x05, "", ""}, word{0xFF, 0x05, "", ""}, word{0xFF, 0x00, "", ""})
	s.exchange(2e-6, read)
	s.exchange(29e-6, []word{{0x9F, -1, "RDID", ""}, {0xFF, 0xEF, "", "Winbond"}, {0xFF, 0x40, "", ""}, {0xFF, 0x18, "", "128 Mbit"}})
	s.exchange(40e-6, []word{{0x05, -1, "RDSR", ""}, {0xFF, 0x00, "", "Ready"}})
	return logic.Capture{End: 46e-6, Signals: []logic.Signal{s.cs, s.sck, s.mosi, s.miso, s.out, s.in}}
}

type app struct {
	spi     logic.Capture
	capture logic.State
	source  string
	diagram logic.Capture
	err     error
	timing  logic.State
	split   float32
}

// parse draws the diagram from the source, keeping the last that parsed
// while it has an error.
func (a *app) parse() {
	d, err := logic.ParseWaveJSON(a.source, 10e-9)
	a.err = err
	if err == nil {
		a.diagram = d
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	heading := func(title, note string) {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).Bold()
			ui.Text(c, note).TextColor(t.TextMuted)
		})
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(3)).Children(func() {
		heading("SPI flash", "W25Q128 at 4 MHz: Read Data from 0x001000, Read JEDEC ID, Read Status Register")
		logic.View(c, &a.capture, a.spi)
		ui.Box(c).Height(t.Space(2))
		heading("WaveJSON", "Edit the diagram; it redraws as you type")
		ui.Split(c, &a.split, func() {
			ui.Column(c).Fill().PaddingEnd(t.Space(3)).Children(func() {
				if ui.TextArea(c, &a.source).Font("monospace").Label("WaveJSON").Grow(1).Changed() {
					a.parse()
				}
			})
		}, func() {
			ui.Column(c).Fill().PaddingStart(t.Space(3)).Gap(t.Space(2)).Children(func() {
				logic.View(c, &a.timing, a.diagram)
				if a.err != nil {
					ui.Text(c, a.err.Error()).TextColor(t.Danger)
				}
			})
		}).Grow(1)
	})
}

func main() {
	a := &app{spi: flash(), source: handshake, split: 480}
	a.parse()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Logic",
			Width:   1440,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

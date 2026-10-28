// Pinout shows two parts: the 40-pin header of a single-board computer,
// as the Raspberry Pi's, and the ATmega328P microcontroller in its
// 28-pin DIP. Hover over a tag or click one in the legend to light a
// bus's pins, and click a pin to read about it below.
//
//	go run ./examples/pinout
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pinout"
)

// pin reads "name/kind/GROUP:Fn GROUP:Fn", kind p for power, g for ground,
// o for other and empty for I/O.
func pin(s string) pinout.Pin {
	f := strings.SplitN(s, "/", 3)
	p := pinout.Pin{Name: f[0]}
	switch f[1] {
	case "p":
		p.Kind = pinout.Power
	case "g":
		p.Kind = pinout.Ground
	case "o":
		p.Kind = pinout.Other
	}
	if len(f) > 2 {
		for _, fn := range strings.Fields(f[2]) {
			g, n, _ := strings.Cut(fn, ":")
			p.Functions = append(p.Functions, pinout.Function{Name: n, Group: g})
		}
	}
	return p
}

func pins(specs ...string) []pinout.Pin {
	out := make([]pinout.Pin, len(specs))
	for i, s := range specs {
		out[i] = pin(s)
	}
	return out
}

var header = pins(
	"3V3/p", "5V/p",
	"GPIO 2//I2C:SDA1", "5V/p",
	"GPIO 3//I2C:SCL1", "GND/g",
	"GPIO 4//Clock:GPCLK0", "GPIO 14//UART:TXD",
	"GND/g", "GPIO 15//UART:RXD",
	"GPIO 17/", "GPIO 18//PCM:CLK PWM:PWM0",
	"GPIO 27/", "GND/g",
	"GPIO 22/", "GPIO 23/",
	"3V3/p", "GPIO 24/",
	"GPIO 10//SPI:MOSI", "GND/g",
	"GPIO 9//SPI:MISO", "GPIO 25/",
	"GPIO 11//SPI:SCLK", "GPIO 8//SPI:CE0",
	"GND/g", "GPIO 7//SPI:CE1",
	"GPIO 0//EEPROM:ID_SD", "GPIO 1//EEPROM:ID_SC",
	"GPIO 5/", "GND/g",
	"GPIO 6/", "GPIO 12//PWM:PWM0",
	"GPIO 13//PWM:PWM1", "GND/g",
	"GPIO 19//PCM:FS SPI:MISO1", "GPIO 16//SPI:CE2",
	"GPIO 26/", "GPIO 20//PCM:DIN SPI:MOSI1",
	"GND/g", "GPIO 21//PCM:DOUT SPI:SCLK1",
)

var atmega = pins(
	"PC6/o/Reset:RESET", "PD0//UART:RXD", "PD1//UART:TXD", "PD2//Interrupt:INT0", "PD3//Interrupt:INT1 PWM:OC2B",
	"PD4//Timer:T0 UART:XCK", "VCC/p", "GND/g", "PB6//Clock:XTAL1", "PB7//Clock:XTAL2",
	"PD5//Timer:T1 PWM:OC0B", "PD6//Analog:AIN0 PWM:OC0A", "PD7//Analog:AIN1", "PB0//Timer:ICP1 Clock:CLKO",
	"PB1//PWM:OC1A", "PB2//SPI:SS PWM:OC1B", "PB3//SPI:MOSI PWM:OC2A", "PB4//SPI:MISO", "PB5//SPI:SCK",
	"AVCC/p", "AREF/o/Analog:AREF", "GND/g", "PC0//Analog:ADC0", "PC1//Analog:ADC1",
	"PC2//Analog:ADC2", "PC3//Analog:ADC3", "PC4//Analog:ADC4 I2C:SDA", "PC5//Analog:ADC5 I2C:SCL",
)

var boards = []struct {
	name  string
	board pinout.Board
}{
	{"40-pin header", pinout.Board{Layout: pinout.Header, Pins: header}},
	{"ATmega328P (DIP-28)", pinout.Board{Layout: pinout.DIP, Pins: atmega}},
}

type app struct {
	chosen int
	states [2]pinout.State
	pins   [2]int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	names := []string{boards[0].name, boards[1].name}
	b := boards[a.chosen].board
	sel := &a.pins[a.chosen]
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.chosen, names...).Label("Part")
			pinout.View(c.Key(a.chosen), &a.states[a.chosen], sel, b)
			text := "Click a pin to read about it"
			if *sel >= 0 {
				p := b.Pins[*sel]
				text = fmt.Sprintf("Pin %d: %s, %s", *sel+1, p.Name, p.Kind)
				for _, f := range p.Functions {
					text += fmt.Sprintf(", %s %s", f.Group, f.Name)
				}
			}
			ui.Text(c, text).TextColor(t.TextMuted)
		})
	})
}

func main() {
	a := &app{pins: [2]int{2, -1}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pinout",
			Width:   820,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

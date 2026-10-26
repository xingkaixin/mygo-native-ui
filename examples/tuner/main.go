// Tuner plucks the six strings of a guitar that has drifted out of tune,
// as the sound it makes itself, and shows the pitch of what plays: click
// a string to hear it and see how far off it is.
//
//	go run ./examples/tuner
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tuner"
)

const rate = 44100

// window is how many samples each reading takes: about 46 ms.
const window = 2048

// guitar is the strings in standard tuning, each off by some cents.
var guitar = []struct {
	name  string
	hz    float64
	cents float64
}{
	{"E2", 82.41, -18}, {"A2", 110.00, 7}, {"D3", 146.83, 0},
	{"G3", 196.00, -32}, {"B3", 246.94, 12}, {"E4", 329.63, -3},
}

// pluck returns 3 seconds of a string at hz plucked, by the
// Karplus-Strong method: a burst of noise that goes round a delay line
// as long as a period, softened a little each time.
func pluck(hz float64) []float32 {
	r := rand.New(rand.NewPCG(uint64(hz), 1))
	period := float64(rate) / hz
	line := make([]float64, int(period)+1)
	g := float64(len(line)) - period
	for i := range line {
		line[i] = r.Float64()*2 - 1
	}
	out := make([]float32, rate*3)
	pos := 0
	for i := range out {
		next := (pos + 1) % len(line)
		// Taking a share g of the next sample shortens the loop by g, which
		// tunes it between whole samples, and damps it.
		v := 0.996 * ((1-g)*line[pos] + g*line[next])
		out[i] = float32(line[pos] * 0.5)
		line[pos] = v
		pos = next
	}
	return out
}

type app struct {
	samples [][]float32
	player  *audio.Player
	playing int
}

func (a *app) hz() float64 {
	if a.player == nil || !a.player.Playing() {
		return 0
	}
	at := int(a.player.Position().Seconds() * rate)
	s := a.samples[a.playing]
	if at+window > len(s) {
		return 0
	}
	return tuner.Detect(s[at:at+window], rate)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.player != nil && a.player.Playing() {
		c.After(time.Second / 30)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		tuner.View(c, a.hz())
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			for i, s := range guitar {
				if ui.Button(c.Key(i), s.name).Clicked() {
					if a.player != nil {
						a.player.Pause()
					}
					a.playing = i
					a.player = audio.NewSound(rate, 1, a.samples[i]).NewPlayer()
					a.player.Play()
				}
			}
		})
	})
}

func newApp() *app {
	a := &app{}
	for _, s := range guitar {
		a.samples = append(a.samples, pluck(s.hz*math.Exp2(s.cents/1200)))
	}
	return a
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tuner",
			Width:   520,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

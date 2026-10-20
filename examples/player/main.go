// Player plays a WAV file through mygo's audio player, or, given none, a
// set of three minutes in six chapters that it synthesizes as it plays,
// under the player's controls. The scrubber shows what has buffered as if
// the audio streamed, at 8 times the speed it plays. Speeds change the
// pitch too, as mygo's players play.
//
//	go run ./examples/player [file.wav]
package main

import (
	"cmp"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/player"
)

const (
	rate = 48000
	// bar is how long a bar is at 120 BPM, in seconds.
	bar = 2.0
)

// part is a chapter of the set, as many bars long, and how loud each
// voice plays in it.
type part struct {
	title                             string
	bars                              int
	pad, bass, arp, kick, hats, snare float64
}

var parts = []part{
	{title: "Intro", bars: 12, pad: 1},
	{title: "Groove", bars: 16, pad: 0.8, bass: 1, kick: 1, hats: 0.6},
	{title: "Arpeggios", bars: 16, pad: 0.6, bass: 0.8, kick: 0.8, hats: 1, arp: 1},
	{title: "Breakdown", bars: 12, pad: 1, hats: 0.5, arp: 0.5},
	{title: "Drop", bars: 24, pad: 0.7, bass: 1, kick: 1, hats: 1, snare: 1, arp: 0.8},
	{title: "Outro", bars: 16, pad: 1, arp: 0.4},
}

var chords = [][]int{{57, 60, 64}, {53, 57, 60}, {48, 55, 64}, {55, 59, 62}}

// hz is the frequency of each MIDI note.
var hz [128]float64

func init() {
	for m := range hz {
		hz[m] = 440 * math.Pow(2, float64(m-69)/12)
	}
}

// set is a decoder making the set as the player reads it, at 120 BPM: a
// pad of Am, F, C and G, a bass on the eighths, an arpeggio on the
// sixteenths, a kick on the beats, a snare on two and four and hats
// between, each voice as loud as its part says. The pad fades in through
// the intro and out through the outro.
type set struct {
	// part is the index of the part of each bar.
	part     []int
	frame    int64
	lp, last float64
	noise    *rand.Rand
}

// newSet returns the set and its chapters.
func newSet() (*set, []player.Chapter) {
	s := &set{noise: rand.New(rand.NewPCG(5, 6))}
	var chapters []player.Chapter
	for k, p := range parts {
		start := time.Duration(float64(len(s.part)) * bar * float64(time.Second))
		chapters = append(chapters, player.Chapter{Title: p.title, Start: start})
		for range p.bars {
			s.part = append(s.part, k)
		}
	}
	return s, chapters
}

func (s *set) SampleRate() int { return rate }
func (s *set) Channels() int   { return 1 }
func (s *set) Len() int64      { return int64(float64(len(s.part)) * bar * rate) }

func (s *set) SetPosition(frame int64) error {
	s.frame = frame
	return nil
}

func (s *set) Read(buf []float32) (int, error) {
	n := int(min(int64(len(buf)), s.Len()-s.frame))
	if n <= 0 {
		return 0, io.EOF
	}
	for i := range n {
		buf[i] = s.sample(float64(s.frame) / rate)
		s.frame++
	}
	return n, nil
}

func saw(f, t float64) float64 { p := f * t; return 2 * (p - math.Floor(p+0.5)) }

// sample returns the sample t seconds in.
func (s *set) sample(t float64) float32 {
	b := int(t / bar)
	p := parts[s.part[b]]
	chord := chords[b/2%4]
	fade := 1.0
	switch p.title {
	case "Intro":
		fade = t / (float64(p.bars) * bar)
	case "Outro":
		fade = max(0, (float64(len(s.part))*bar-t)/(float64(p.bars)*bar))
	}
	pad := 0.0
	for _, m := range chord {
		pad += saw(hz[m]*1.003, t) + saw(hz[m]*0.997, t)
	}
	cut := 600 + 3000*(0.5-0.5*math.Cos(2*math.Pi*t/(8*bar)))
	s.lp += (1 - math.Exp(-2*math.Pi*cut/rate)) * (pad*0.05 - s.lp)
	v := s.lp * p.pad * fade

	eighth := math.Mod(t, 0.25)
	root := hz[chord[0]-12]
	v += p.bass * 0.3 * math.Exp(-eighth/0.12) * (math.Sin(2*math.Pi*root*t) + 0.3*saw(root, t))

	sixteenth := math.Mod(t, 0.125)
	note := chord[int(t/0.125)%len(chord)] + 12*(int(t/0.375)%2+1)
	v += p.arp * 0.12 * math.Exp(-sixteenth/0.08) * math.Sin(2*math.Pi*hz[note]*t) * fade

	beat := math.Mod(t, 0.5)
	v += p.kick * 0.8 * math.Exp(-beat/0.18) * math.Sin(2*math.Pi*(45*beat+75*(1-math.Exp(-beat/0.03))*0.03))
	noise := s.noise.Float64()*2 - 1
	if off := math.Mod(t+0.25, 0.5); off < 0.06 {
		v += p.hats * 0.16 * (noise - s.last) * math.Exp(-off/0.02)
	}
	s.last = noise
	if two := math.Mod(t+0.5, 1); two < 0.2 {
		v += p.snare * 0.3 * noise * math.Exp(-two/0.05)
	}
	return float32(math.Tanh(v))
}

type app struct {
	title    string
	track    *audio.Player
	chapters []player.Chapter
	controls player.State
	position time.Duration
	playing  bool
	buffered []player.Range
	last     time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.position, a.playing = a.track.Position(), a.track.Playing()
	a.buffer(c)
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Justify(ui.Center).Children(func() {
		ui.Text(c, a.title).Bold().FontSize(t.FontSize + 4).SingleLine()
		ctl := player.Controls(c, &a.controls, &a.position, &a.playing, player.Media{
			Duration: a.track.Duration(),
			Chapters: a.chapters,
			Buffered: a.buffered,
		})
		if at, ok := ctl.Seeked(); ok {
			a.track.Seek(at)
		}
	})
	switch {
	case a.playing && !a.track.Playing():
		a.track.Play()
	case !a.playing && a.track.Playing():
		a.track.Pause()
	}
	a.track.SetVolume(a.controls.Volume()).SetSpeed(a.controls.Speed())
	if a.playing {
		c.After(time.Second / 60)
	}
}

// buffer moves on what has buffered, as if the audio streamed at 8 times
// the speed it plays: the stretch the position is in grows to the end,
// and a seek past what has buffered starts another.
func (a *app) buffer(c *ui.Context) {
	now := c.Now()
	var dt time.Duration
	if !a.last.IsZero() {
		dt = now.Sub(a.last)
	}
	a.last = now
	end, pos := a.track.Duration(), a.position
	i := slices.IndexFunc(a.buffered, func(r player.Range) bool { return r.From <= pos && pos <= r.To })
	if i < 0 {
		i, _ = slices.BinarySearchFunc(a.buffered, pos, func(r player.Range, d time.Duration) int { return cmp.Compare(r.From, d) })
		a.buffered = slices.Insert(a.buffered, i, player.Range{From: pos, To: pos})
	}
	r := &a.buffered[i]
	r.To = min(end, r.To+8*dt)
	if i+1 < len(a.buffered) && r.To >= a.buffered[i+1].From {
		r.To = max(r.To, a.buffered[i+1].To)
		a.buffered = slices.Delete(a.buffered, i+1, i+2)
	}
	if r.To < end {
		c.After(time.Second / 20)
	}
}

func main() {
	a := &app{title: "Night Drive"}
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		if a.track, err = audio.OpenStream(f); err != nil {
			log.Fatal(err)
		}
		a.title = filepath.Base(os.Args[1])
	} else {
		s, chapters := newSet()
		a.track, a.chapters = audio.NewPlayer(s), chapters
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Player",
			Width:   720,
			Height:  200,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

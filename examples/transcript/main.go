// Transcript shows a made-up three-minute interview from a podcast, three
// speakers on a city roof talking about bees, as a transcript that
// follows the player's controls. Nothing plays: the playhead moves with
// time while playing, so play, pause and the scrubber move the
// transcript. Click a word to go to it, find text in the field, and
// select words to copy them with their speakers and times. Words heard
// with less than 70% confidence are underlined.
//
//	go run ./examples/transcript
package main

import (
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/player"
	"github.com/xingkaixin/mygo-native-ui/ui/transcript"
)

// lines are the interview, a speaker and what they said. A word ending in
// ~NN was heard with NN% confidence.
var lines = []struct{ speaker, text string }{
	{"Maya Chen", "Welcome back to Field Notes. I'm Maya Chen, and today we're up on a roof in the middle of the city, talking to someone who keeps about two hundred thousand bees up here. Tomás,~58 thanks for having us."},
	{"Tomás Ortega", "Thanks for climbing all those stairs. Most people don't expect to find bees six floors above a bus depot,~64 but honestly, they do really well up here."},
	{"Priya Shah", "Why is that? I'd have guessed a roof is the worst place for them. It's windy, it's hot, there's nothing growing."},
	{"Tomás Ortega", "You'd think so. But a bee will fly two or three miles for food, so the roof is just where they sleep. The city is full of street trees, parks, window boxes, little gardens behind restaurants. There's something flowering from March to October, which you can't say about a lot of farmland."},
	{"Maya Chen", "So city honey tastes different?"},
	{"Tomás Ortega", "Very different. In June it's mostly linden,~52 so it's pale and kind of minty. By August it gets darker, because they're on the, um,~41 the Japanese pagoda~57 trees and whatever people planted on their balconies."},
	{"Priya Shah", "What surprised you most when you started?"},
	{"Tomás Ortega", "How calm they are. People picture a swarm chasing them down the street. Really, if you move slowly and you don't stand in their flight path, they ignore you. The hardest part was the paperwork. You have to register every hive with the city, and the building had to agree to it."},
	{"Maya Chen", "And the neighbors?"},
	{"Tomás Ortega", "One complaint in six years. A man across the street thought the bees were drinking from his pool. They were, to be fair. So now I keep a shallow tray of water up here with some corks floating in it, and they stay home."},
	{"Priya Shah", "We should talk about the varroa~38 mite, because I think that's the part most listeners haven't heard of."},
	{"Tomás Ortega", "Right. It's a tiny mite that rides on the bees and spreads viruses. Every beekeeper is fighting it. I check each hive once a month by counting how many mites fall onto a sticky board, and if the count climbs, I treat them with oxalic~45 acid in the winter, when there's no brood.~66"},
	{"Maya Chen", "If someone listening wants to start, where should they begin?"},
	{"Tomás Ortega", "Find your local beekeeping club before you buy anything. Spend a season helping someone else, open a few hives, get stung once or twice. Then start with two hives, not one, so you can compare them and borrow from the strong one when the weak one struggles."},
	{"Priya Shah", "Two hives. That's the advice."},
	{"Maya Chen", "Tomás~61 Ortega, thank you. That's Field Notes for this week. Next time we're going underground, into the old water tunnels beneath the river."},
}

// interview times the words of lines as if spoken: each takes longer the
// longer it is, a comma pauses, the end of a sentence pauses longer, and
// so does a change of speaker. Words without a confidence of their own
// were heard with 88% to 99%.
func interview() []transcript.Segment {
	r := rand.New(rand.NewPCG(1, 2))
	var segments []transcript.Segment
	at := 800 * time.Millisecond
	for _, l := range lines {
		seg := transcript.Segment{Speaker: l.speaker, Start: at}
		for _, f := range strings.Fields(l.text) {
			sure := 0.88 + 0.11*r.Float64()
			if text, p, ok := strings.Cut(f, "~"); ok {
				n, _ := strconv.Atoi(p)
				f, sure = text, float64(n)/100
			}
			d := 150*time.Millisecond + 42*time.Millisecond*time.Duration(utf8.RuneCountInString(f))
			seg.Words = append(seg.Words, transcript.Word{Text: f, Start: at, End: at + d, Confidence: sure})
			at += d
			switch f[len(f)-1] {
			case ',':
				at += 200 * time.Millisecond
			case '.', '?':
				at += 350 * time.Millisecond
			}
		}
		seg.End = seg.Words[len(seg.Words)-1].End
		segments = append(segments, seg)
		at += 500 * time.Millisecond
	}
	return segments
}

type app struct {
	segments   []transcript.Segment
	duration   time.Duration
	transcript transcript.State
	controls   player.State
	position   time.Duration
	playing    bool
	// last is when the position last moved, zero while paused.
	last time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.advance(c)
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5), t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Bees on the Roof").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Field Notes, episode 42 · Maya Chen and Priya Shah with Tomás Ortega").TextColor(t.TextMuted).SingleLine()
		})
		ui.Column(c).Grow(1).Children(func() {
			// Nothing plays: the position is all there is to move.
			transcript.View(c, &a.transcript, &a.position, a.segments)
		})
		player.Controls(c, &a.controls, &a.position, &a.playing, player.Media{Duration: a.duration})
	})
	if a.playing {
		c.After(time.Second / 60)
	}
}

// advance moves the position on by the time since the last frame, at the
// controls' speed, while playing, and stops at the end.
func (a *app) advance(c *ui.Context) {
	if a.playing && a.position >= a.duration {
		a.position = 0
	}
	if !a.playing {
		a.last = time.Time{}
		return
	}
	now := c.Now()
	if !a.last.IsZero() {
		a.position += time.Duration(float64(now.Sub(a.last)) * a.controls.Speed())
	}
	a.last = now
	if a.position >= a.duration {
		a.position, a.playing = a.duration, false
	}
}

func main() {
	a := &app{segments: interview()}
	a.duration = a.segments[len(a.segments)-1].End + 2*time.Second
	a.transcript.Threshold = 0.7
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Transcript",
			Width:   760,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

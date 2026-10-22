// Fretboard explores scales on a guitar's neck: choose a key, a scale and
// a tuning, read the notes by name or by interval, and click a fret to
// play it and mark it or not. Below, common chords strum when clicked.
//
//	go run ./examples/fretboard
package main

import (
	"log"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/fretboard"
)

var keys = []string{"C", "D♭", "D", "E♭", "E", "F", "F♯", "G", "A♭", "A", "B♭", "B"}

// tuning is a tuning's name and its strings' notes, lowest first, as MIDI
// numbers them.
type tuning struct {
	name  string
	notes []int
}

var tunings = []tuning{
	{"Standard", []int{40, 45, 50, 55, 59, 64}},
	{"Drop D", []int{38, 45, 50, 55, 59, 64}},
	{"DADGAD", []int{38, 45, 50, 55, 57, 62}},
	{"Open G", []int{38, 43, 50, 55, 59, 62}},
	{"Bass", []int{28, 33, 38, 43}},
}

var chords = []fretboard.Chord{
	{Name: "C", Shape: "x32010"},
	{Name: "G", Shape: "320003"},
	{Name: "D", Shape: "xx0232"},
	{Name: "Am", Shape: "x02210"},
	{Name: "Em", Shape: "022000"},
	{Name: "F", Shape: "133211", Base: 1},
	{Name: "B7", Shape: "x21202"},
}

type app struct {
	key, scale, tuning string
	labels             int
	marks              fretboard.Marks
	played             string
}

func (a *app) neck() fretboard.Neck {
	i := slices.IndexFunc(tunings, func(t tuning) bool { return t.name == a.tuning })
	return fretboard.Neck{Tuning: tunings[i].notes}
}

// mark marks the notes of the scale chosen on the neck.
func (a *app) mark() {
	s := slices.IndexFunc(fretboard.Scales, func(f fretboard.Formula) bool { return f.Name == a.scale })
	a.marks = a.neck().Marks(slices.Index(keys, a.key), fretboard.Scales[s])
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Fretboard").Bold().FontSize(t.FontSize + 6).Grow(1)
			ui.Text(c, a.played).TextColor(t.TextMuted)
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Wrap().Children(func() {
			scales := make([]string, len(fretboard.Scales))
			for i, s := range fretboard.Scales {
				scales[i] = s.Name
			}
			changed := false
			ui.Text(c, "Key").TextColor(t.TextMuted)
			changed = ui.Select(c, &a.key, keys).Width(80).Label("Key").Changed() || changed
			ui.Text(c, "Scale").TextColor(t.TextMuted).MarginStart(t.Space(2))
			changed = ui.Select(c, &a.scale, scales).Width(170).Label("Scale").Changed() || changed
			ui.Text(c, "Tuning").TextColor(t.TextMuted).MarginStart(t.Space(2))
			names := make([]string, len(tunings))
			for i, tu := range tunings {
				names[i] = tu.name
			}
			changed = ui.Select(c, &a.tuning, names).Width(120).Label("Tuning").Changed() || changed
			ui.Box(c).Grow(1)
			ui.Segmented(c, &a.labels, "Names", "Intervals").Label("Labels")
			if changed {
				a.mark()
			}
		})
		neck := a.neck()
		if p, ok := fretboard.View(c, neck, &a.marks, fretboard.Labels(a.labels)).Clicked(); ok {
			note := neck.Note(p)
			// Guitarists count the strings from the highest.
			a.played = "Played " + keys[note%12] + strconv.Itoa(note/12-1) + " on string " +
				strconv.Itoa(len(neck.Tuning)-p.String) + ", fret " + strconv.Itoa(p.Fret)
		}
		ui.Text(c, "Chords").Bold()
		ui.Row(c).Gap(t.Space(2)).Wrap().Children(func() {
			for i, ch := range chords {
				fretboard.Diagram(c.Key(i), nil, ch)
			}
		})
		if err := audio.Err(); err != nil {
			ui.Text(c, err.Error()).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		}
	})
}

func main() {
	a := &app{key: "A", scale: "Minor pentatonic", tuning: "Standard", labels: 1}
	a.mark()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Fretboard",
			Width:   1120,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// Ragchunks splits a made-up kettle handbook for retrieval and shows how.
// Drag the chunk size and overlap to split it again, switch between
// windows of characters and paragraphs then sentences, and type a
// question: the chunks rank by a BM25 score computed here, the best four
// retrieved. Rest the pointer on a chunk in the text or the list to find
// it in the other, and click to keep it lit.
//
//	go run ./examples/ragchunks
package main

import (
	"cmp"
	"log"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ragchunks"
)

const handbook = `Brightwell Ember Kettle: Owner's Handbook

Thank you for choosing the Ember, a 1.5 litre variable temperature kettle. Read this handbook before first use and keep it with the kettle. The model number and serial number are printed on the label under the base.

Getting started. Before first use, fill the kettle to the MAX line, boil the water and pour it away. Repeat this twice to rinse out any dust from manufacturing. Place the base on a dry, level surface at least 20 cm from the edge of the counter and away from cupboards, as steam escapes through the lid while the kettle boils.

Setting the temperature. Press the dial to wake the display, then turn it to choose a temperature from 40 °C to 100 °C in steps of 5 degrees. Green tea brews best at 80 °C, oolong at 90 °C and black tea or coffee at 95 °C. Press the dial again to start heating. The ring around the dial glows amber while the kettle heats and turns white when the water is ready.

Keep warm. Hold the dial for two seconds to keep the water at the chosen temperature for up to 30 minutes. The kettle beeps once when keep warm starts and twice when it ends. Lifting the kettle from the base pauses keep warm, and it resumes when you put the kettle back within two minutes.

Cleaning. Unplug the base and let the kettle cool before cleaning. Wipe the outside with a soft damp cloth; never put the kettle or the base in a dishwasher or under a running tap. The mesh filter in the spout lifts out for rinsing, and you should rinse it every week.

Descaling. Hard water leaves limescale inside the kettle, which slows heating and makes the display read the wrong temperature. Descale every four weeks in hard water areas, or every three months where water is soft. Fill the kettle halfway with equal parts water and white vinegar, boil it, and leave it to stand for an hour. Pour it away, then boil fresh water twice and discard it before making drinks. Citric acid works as well as vinegar and leaves no smell.

Troubleshooting. If the display shows E1, the kettle was switched on empty: fill it and press the dial. E2 means the temperature sensor is covered in scale, so descale the kettle. If the kettle switches off before the water boils, check that the lid is closed firmly. A clicking sound while heating is normal and comes from the metal base expanding.

Warranty. The Ember is guaranteed against defects in materials and workmanship for two years from the date of purchase. The warranty does not cover damage from limescale, dropping the kettle or using it with a base from another model. To make a claim, contact Brightwell support with your serial number and proof of purchase, and we will repair or replace the kettle free of charge.

Safety. Never fill the kettle above the MAX line, as boiling water may spill from the spout. Keep the kettle and its cord out of reach of children. Do not open the lid while the water is boiling. Use the kettle only with the base supplied, and stop using it if the cord or plug is damaged.`

// separators split the handbook into paragraphs, then lines, sentences
// and words.
var separators = []string{"\n\n", "\n", ". ", " "}

type app struct {
	size, overlap float64
	by            int
	query         string
	inspector     ragchunks.State

	// split is the size, overlap and way the chunks were split by, and
	// asked the query the hits answer.
	split  [3]int
	asked  string
	chunks []ragchunks.Chunk
	hits   []ragchunks.Hit
}

// word is a word or a piece of punctuation, which stands in for a token.
var word = regexp.MustCompile(`\w+|[^\w\s]`)

// stop are words too common to tell chunks apart.
var stop = map[string]bool{
	"a": true, "an": true, "and": true, "the": true, "of": true, "to": true, "in": true, "is": true,
	"it": true, "i": true, "my": true, "do": true, "does": true, "how": true, "what": true, "for": true,
	"should": true, "can": true, "on": true, "with": true, "when": true, "why": true,
}

// terms returns the words of text worth matching, in lower case.
func terms(text string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if !stop[w] {
			out = append(out, w)
		}
	}
	return out
}

// bm25 scores the chunks of text against query by Okapi BM25 and returns
// the best four that match at all; nil for an empty query.
func bm25(text, query string, chunks []ragchunks.Chunk) []ragchunks.Hit {
	q := terms(query)
	if len(q) == 0 {
		return nil
	}
	runes := []rune(text)
	tfs := make([]map[string]int, len(chunks))
	lens := make([]int, len(chunks))
	df := map[string]int{}
	total := 0
	for i, ch := range chunks {
		tfs[i] = map[string]int{}
		for _, w := range terms(string(runes[ch.Start:ch.End])) {
			if tfs[i][w] == 0 {
				df[w]++
			}
			tfs[i][w]++
			lens[i]++
		}
		total += lens[i]
	}
	const k1, b = 1.2, 0.75
	n, avg := float64(len(chunks)), float64(total)/float64(max(len(chunks), 1))
	hits := []ragchunks.Hit{}
	for i, tf := range tfs {
		score := 0.0
		for _, term := range q {
			f := float64(tf[term])
			idf := math.Log(1 + (n-float64(df[term])+0.5)/(float64(df[term])+0.5))
			score += idf * f * (k1 + 1) / (f + k1*(1-b+b*float64(lens[i])/avg))
		}
		if score > 0 {
			hits = append(hits, ragchunks.Hit{Chunk: i, Score: score})
		}
	}
	slices.SortFunc(hits, func(x, y ragchunks.Hit) int { return cmp.Compare(y.Score, x.Score) })
	return hits[:min(4, len(hits))]
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	size, overlap := int(a.size)/10*10, int(a.overlap)/10*10
	split := [3]int{size, overlap, a.by}
	resplit := split != a.split || a.chunks == nil
	if resplit {
		a.split = split
		if a.by == 0 {
			a.chunks = ragchunks.Split(handbook, size, overlap)
		} else {
			a.chunks = ragchunks.Split(handbook, size, overlap, separators...)
		}
	}
	if resplit || a.query != a.asked {
		a.asked = a.query
		a.hits = bm25(handbook, a.query, a.chunks)
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Gap(t.Space(3)).Padding(t.Space(4), t.Space(5)).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Children(func() {
				ui.Text(c, "Chunk inspector").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Brightwell Ember Kettle handbook, split for retrieval").TextColor(t.TextMuted)
			})
			ui.Row(c).Wrap().GapX(t.Space(6)).GapY(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "Split by").TextColor(t.TextMuted)
					ui.Segmented(c, &a.by, "Characters", "Paragraphs, sentences").Label("Split by")
				})
				slider(c, "Chunk size", &a.size, 100, 1000, size)
				slider(c, "Overlap", &a.overlap, 0, 300, overlap)
			})
			ui.SearchField(c, &a.query).Placeholder("Ask the handbook a question").Label("Query")
		})
		ui.Divider(c)
		ragchunks.View(c, &a.inspector, handbook, a.chunks, a.hits).Grow(1)
	})
}

// slider shows a slider of *v from lo to hi named name, and shown, the
// value it stands for.
func slider(c *ui.Context, name string, v *float64, lo, hi float64, shown int) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, name).TextColor(t.TextMuted)
		ui.Slider(c, v, lo, hi).Width(t.Space(36)).Label(name)
		ui.Text(c, strconv.Itoa(shown)).FontFeatures("tnum").Width(t.Space(9))
	})
}

func main() {
	a := &app{size: 300, overlap: 80, by: 1, query: "how often should I descale"}
	a.inspector.Tokens = func(text string) int { return len(word.FindAllString(text, -1)) }
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "RAG chunks",
			Width:   1200,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

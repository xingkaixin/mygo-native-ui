// Logprobs shows three made-up completions, an answer, a function and the
// first line of a ghost story, with how likely the model found each token.
// Rest the pointer on a token for its alternatives, click it to pin its
// card and step through the tokens with the arrow keys, tint the tokens
// by probability or by entropy, and underline those below a threshold.
//
//	go run ./examples/logprobs
package main

import (
	"cmp"
	"log"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/logprobs"
)

// completion is a prompt and the tokens a model answered it with.
type completion struct {
	name, prompt string
	tokens       []logprobs.Token
}

type app struct {
	completions []completion
	shown, mode int
	viewer      logprobs.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	cm := a.completions[a.shown]
	a.viewer.Mode = logprobs.Mode(a.mode)
	names := make([]string, len(a.completions))
	for i, cm := range a.completions {
		names[i] = cm.name
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "Logprobs").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Made-up completions, each token tinted by how unsure the model was").TextColor(t.TextMuted).SingleLine()
			})
			ui.Segmented(c, &a.shown, names...).Label("Completion")
		})
		ui.Row(c).Wrap().GapX(t.Space(6)).GapY(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Tint by").TextColor(t.TextMuted)
				ui.Segmented(c, &a.mode, "Probability", "Entropy").Label("Tint by")
			})
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Underline below").TextColor(t.TextMuted)
				ui.Slider(c, &a.viewer.Threshold, 0, 0.5).Width(t.Space(40)).Label("Underline below")
				ui.Text(c, strconv.Itoa(int(math.Round(100*a.viewer.Threshold)))+"%").FontFeatures("tnum").Width(t.Space(8))
			})
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Boundaries").TextColor(t.TextMuted)
				ui.Switch(c, &a.viewer.Boundaries).Label("Boundaries")
			})
		})
		ui.Column(c).Gap(t.Space(1)).Padding(t.Space(3), t.Space(4)).Radius(t.Space(2.5)).Background(t.Surface).Children(func() {
			ui.Text(c, "Prompt").FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
			ui.Text(c, cm.prompt)
		})
		var ev logprobs.Events
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(0, t.Space(1), t.Space(2)).Children(func() {
				ev = logprobs.View(c, &a.viewer, cm.tokens)
			})
		})
		status := "Click a token to pin its card, then step with the arrow keys; Esc unpins it"
		if i, ok := ev.Pinned(); ok {
			tk := cm.tokens[i]
			status = "Pinned token " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(cm.tokens)) + ", " +
				strconv.Quote(tk.Text) + " at " + strconv.FormatFloat(100*math.Exp(tk.Logprob), 'f', 1, 64) + "%"
		}
		ui.Text(c, status).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

// tok makes a token the model chose with probability p, and its top
// alternatives, likeliest first, of it and alts, pairs of a text and its
// probability.
func tok(text string, p float64, alts ...any) logprobs.Token {
	top := []logprobs.Alternative{{Text: text, Logprob: math.Log(p)}}
	for i := 0; i+1 < len(alts); i += 2 {
		top = append(top, logprobs.Alternative{Text: alts[i].(string), Logprob: math.Log(alts[i+1].(float64))})
	}
	slices.SortStableFunc(top, func(a, b logprobs.Alternative) int { return cmp.Compare(b.Logprob, a.Logprob) })
	return logprobs.Token{Text: text, Logprob: math.Log(p), Top: top}
}

// answer is a factual answer, sure of its facts and less of its words.
var answer = []logprobs.Token{
	tok("The", .87, "Canberra", .09, "Australia", .03),
	tok(" capital", .96, " Australian", .02),
	tok(" of", .99, " city", .006),
	tok(" Australia", .995, " the", .003),
	tok(" is", .99, " has", .004),
	tok(" Canberra", .93, " Sydney", .04, " not", .02),
	tok(",", .55, ".", .43),
	tok(" not", .81, " a", .08, " which", .06),
	tok(" Sydney", .97, " Melbourne", .02),
	tok(".", .96, ",", .03),
	tok(" Canberra", .34, " The", .29, " It", .21, " Although", .06, " When", .04),
	tok(" was", .78, " is", .12, " became", .07),
	tok(" chosen", .41, " selected", .33, " founded", .1, " built", .06, " established", .05),
	tok(" in", .58, " as", .36),
	tok(" 1908", .38, " 1913", .31, " 1901", .17, " 1911", .06),
	tok(" as", .92, ",", .05),
	tok(" a", .99, " the", .008),
	tok(" compromise", .97, " neutral", .015),
	tok(" between", .96, " site", .02),
	tok(" Sydney", .88, " the", .07, " Melbourne", .04),
	tok(" and", .995, ",", .003),
	tok(" Melbourne", .998, " Canberra", .001),
	tok(",", .74, ".", .2, " —", .03),
	tok(" the", .83, " which", .1, " then", .03),
	tok(" two", .9, " country", .05, " nation", .03),
	tok(" largest", .52, " biggest", .2, " rival", .16, " main", .06),
	tok(" cities", .95, " and", .02),
	tok(",", .62, " of", .12, ".", .11, " in", .08),
	tok(" which", .48, " each", .2, " both", .17, " who", .05),
	tok(" both", .6, " each", .21, " had", .08),
	tok(" wanted", .31, " claimed", .26, " vied", .15, " sought", .12, " competed", .07),
	tok(" the", .72, " to", .22),
	tok(" title", .45, " honour", .2, " role", .15, " status", .1),
	tok(".", .97, ",", .02),
	tok(" Parliament", .29, " The", .26, " Construction", .16, " Building", .09, " It", .08),
	tok(" first", .44, " moved", .38, " met", .12),
	tok(" sat", .67, " met", .29),
	tok(" there", .82, " in", .14),
	tok(" in", .99, " on", .005),
	tok(" 1927", .83, " 1913", .05, " 1923", .04, " 1928", .03),
	tok(".", .9, ",", .06),
	tok("\n\n", .41, "<|endoftext|>", .52),
	tok("The", .46, "Its", .2, "Canberra", .17, "Today", .08),
	tok(" city", .74, " site", .12, " plan", .07),
	tok(" was", .93, " itself", .03),
	tok(" designed", .7, " planned", .25),
	tok(" by", .99, " in", .006),
	tok(" the", .38, " American", .45, " Walter", .12),
	tok(" American", .55, " Chicago", .2, " architects", .14, " husband", .05),
	tok(" architects", .63, " architect", .25, " couple", .06),
	tok(" Walter", .96, " Marion", .03),
	tok(" Bur", .995, " Griffin", .004),
	tok("ley", .9998),
	tok(" Griffin", .999, " G", .0005),
	tok(" and", .97, ",", .02),
	tok(" Marion", .88, " his", .1),
	tok(" Mah", .93, " Griffin", .05),
	tok("ony", .9998),
	tok(" Griffin", .98, ",", .01),
	tok(",", .43, ".", .51),
	tok(" who", .64, " winners", .14, " after", .12),
	tok(" won", .9, " had", .05),
	tok(" an", .62, " the", .33),
	tok(" international", .96, " open", .02),
	tok(" competition", .99, " design", .007),
	tok(" in", .73, ".", .23),
	tok(" 1912", .91, " 1911", .05),
	tok(".", .97, ",", .02),
}

// function is a Python function, sure of its syntax and torn over names.
var function = []logprobs.Token{
	tok("def", .93, "```", .05),
	tok(" fib", .34, " fibonacci", .41, " nth", .12, " fibo", .05),
	tok("(n", .9, "(", .08),
	tok(":", .63, "):", .35),
	tok(" int", .99, " Int", .004),
	tok(")", .995, ",", .003),
	tok(" ->", .97, ":", .02),
	tok(" int", .995, " Int", .002),
	tok(":\n", .996, ":", .003),
	tok("   ", .98, "\t", .01),
	tok(` """`, .46, " if", .21, " a", .2, " #", .05),
	tok("Return", .58, "Returns", .27, "Compute", .06, "Calculate", .05),
	tok(" the", .97, " n", .01),
	tok(" n", .62, " nth", .3),
	tok("th", .995, "-th", .004),
	tok(" Fibonacci", .97, " number", .02),
	tok(" number", .96, " term", .02, " numbers", .01),
	tok(`."""`+"\n", .77, ".\n", .11, " (", .06, `"""`+"\n", .05),
	tok("   ", .99, "\t", .005),
	tok(" a", .79, " if", .15),
	tok(",", .97, " =", .02),
	tok(" b", .99, " _", .004),
	tok(" =", .998, ",", .001),
	tok(" ", .998, " (", .001),
	tok("0", .93, "1", .07),
	tok(",", .999),
	tok(" ", .999),
	tok("1", .97, "0", .02),
	tok("\n", .99, ";", .003),
	tok("   ", .98, "\t", .01),
	tok(" for", .91, " while", .04, " if", .03),
	tok(" _", .58, " i", .38),
	tok(" in", .999),
	tok(" range", .998, " xrange", .001),
	tok("(n", .97, "(", .02),
	tok("):\n", .99, "):", .006),
	tok("       ", .99, "\t", .005),
	tok(" a", .99, " b", .005),
	tok(",", .99),
	tok(" b", .99),
	tok(" =", .99, " +=", .004),
	tok(" b", .99, " a", .006),
	tok(",", .99),
	tok(" a", .99, " b", .004),
	tok(" +", .995, "+", .003),
	tok(" b", .998),
	tok("\n", .97, ";", .01),
	tok("   ", .98, "\t", .01),
	tok(" return", .99, " print", .005),
	tok(" a", .96, " b", .03),
	tok("\n", .58, "<|endoftext|>", .35),
}

// story is the first line of a ghost story, sampled where the model had
// many ways to go.
var story = []logprobs.Token{
	tok("The", .38, "It", .17, "Every", .12, "No", .07, "Nobody", .05),
	tok(" lighthouse", .08, " house", .21, " old", .17, " first", .09, " night", .07),
	tok(" keeper", .86, " on", .05, " had", .03),
	tok(" had", .45, " was", .4, "'s", .06),
	tok(" been", .97, " died", .02),
	tok(" dead", .55, " gone", .15, " missing", .13, " drowned", .04),
	tok(" for", .96, " eleven", .01),
	tok(" eleven", .07, " three", .22, " ten", .18, " twenty", .12, " seven", .09),
	tok(" years", .97, " days", .02),
	tok(",", .95, " when", .02, " before", .015),
	tok(" but", .88, " and", .08),
	tok(" every", .41, " the", .38, " someone", .09, " his", .04),
	tok(" night", .82, " evening", .1),
	tok(" at", .16, " the", .46, " his", .14, ",", .08),
	tok(" nine", .11, " midnight", .52, " dusk", .14, " exactly", .06, " eight", .05),
	tok(" the", .43, " o", .23, ",", .19, " sharp", .05),
	tok(" lamp", .38, " light", .45, " beam", .08),
	tok(" still", .44, " was", .17, " came", .14, " turned", .12),
	tok(" turned", .29, " burned", .31, " came", .12, " shone", .1, " swept", .07),
	tok(",", .81, ".", .12),
	tok(" and", .9, " slow", .03),
	tok(" every", .55, " each", .2, " in", .1),
	tok(" morning", .93, " dawn", .03),
	tok(" there", .31, " the", .3, " someone", .09, " fresh", .06),
	tok(" was", .83, " were", .14),
	tok(" fresh", .12, " a", .27, " sand", .1, " salt", .08, " water", .07),
	tok(" salt", .19, " sand", .22, " seaweed", .14, " water", .12, " oil", .06),
	tok(" on", .48, " water", .17, " in", .12, " crusted", .04),
	tok(" the", .97, " his", .02),
	tok(" stairs", .33, " steps", .27, " floor", .15, " glass", .06),
	tok(".", .86, " —", .05),
}

func main() {
	a := &app{completions: []completion{
		{"Answer", "What is the capital of Australia, and why isn’t it Sydney?", answer},
		{"Code", "Write a Python function that returns the nth Fibonacci number.", function},
		{"Story", "Write the opening line of a ghost story.", story},
	}}
	a.viewer.Threshold = 0.1
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Logprobs",
			Width:   920,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

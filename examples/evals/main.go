// Evals shows a made-up eval run: three models classifying support emails
// and giving short answers, fifteen cases graded pass or fail, the answers
// scored. Filter to the cases that fail or where the models disagree,
// click a cell or move with the arrow keys to read the output against
// another model's, and rate outputs with a thumb up or down.
//
//	go run ./examples/evals
package main

import (
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/evals"
)

// model is how a made-up model answers: how long it takes before its
// first token and for each character, and its price for a token, 0 when
// it runs locally.
type model struct {
	name          string
	first, perOut time.Duration
	price         float64
}

var models = []model{
	{"kestrel-pro", 820 * time.Millisecond, 14 * time.Millisecond, 0.000006},
	{"kestrel-mini", 310 * time.Millisecond, 5 * time.Millisecond, 0.0000008},
	{"basalt-8b", 160 * time.Millisecond, 3 * time.Millisecond, 0},
}

// results gives the models' outputs to prompt their latency and cost,
// made up the same each run, and grades them by pass.
func results(prompt string, outputs [3]string, pass func(k int, output string) bool) []evals.Result {
	timing := rand.New(rand.NewPCG(uint64(len(prompt)), 14))
	rs := make([]evals.Result, len(models))
	for k, m := range models {
		out := outputs[k]
		took := m.first + time.Duration(len(out))*m.perOut
		took += time.Duration(timing.Float64() * 0.3 * float64(took))
		rs[k] = evals.Result{
			Output:  out,
			Pass:    pass(k, out),
			Latency: took.Round(time.Millisecond),
			Cost:    float64(len(prompt)+len(out)) / 4 * m.price,
		}
	}
	return rs
}

// label is a support email to put in a category, which the output must
// start with.
func label(id, email, want string, outputs [3]string) evals.Case {
	return evals.Case{
		ID:       id,
		Prompt:   email,
		Expected: want,
		Results: results(email, outputs, func(_ int, out string) bool {
			return strings.HasPrefix(out, want+":")
		}),
	}
}

// answer is a question whose answers a grader scored by the criteria,
// passing from 0.7.
func answer(id, question, criteria string, outputs [3]string, scores [3]float64) evals.Case {
	rs := results(question, outputs, func(k int, _ string) bool { return scores[k] >= 0.7 })
	for k := range rs {
		rs[k].Score, rs[k].Scored = scores[k], true
	}
	return evals.Case{ID: id, Prompt: question, Expected: criteria, Results: rs}
}

func run() evals.Run {
	names := make([]string, len(models))
	for k, m := range models {
		names[k] = m.name
	}
	return evals.Run{Columns: names, Cases: []evals.Case{
		label("sup-01", "Hi, I was charged twice for my March invoice (#4471). Can you refund the duplicate charge?", "billing", [3]string{
			"billing: Duplicate charge on the March invoice; asks for a refund.",
			"billing: Charged twice for invoice #4471 and wants the extra charge refunded.",
			"billing: Customer wants a refund for a double charge.",
		}),
		label("sup-02", "Since yesterday's update the desktop app crashes whenever I open a shared folder. Reinstalling didn't help.", "technical", [3]string{
			"technical: The app crashes opening shared folders after the latest update.",
			"technical: Crash when opening shared folders since the update.",
			"technical: App crashes after the update; reinstalling did not fix it.",
		}),
		label("sup-03", "How do I change the email address on my account? I no longer have access to the old one.", "account", [3]string{
			"account: Wants to change the account's email; has lost access to the old address.",
			"account: Needs to update the email on the account.",
			"technical: The user cannot access their old email.",
		}),
		label("sup-04", "My card was declined when upgrading to the Team plan, but the bank says there's no problem on their side. Is something wrong on your end?", "billing", [3]string{
			"billing: Card declined while upgrading to the Team plan; the bank sees no issue.",
			"technical: Card payment fails on upgrade, possibly a checkout error.",
			"technical: Payment error during the upgrade.",
		}),
		label("sup-05", "Please delete my account and all of my data. I'm moving to another service.", "account", [3]string{
			"account: Asks to delete the account and all of its data.",
			"account: Account and data deletion request.",
			"account: Wants the account deleted.",
		}),
		label("sup-06", "Love the new dark mode! Any chance you'll add a sepia theme too?", "other", [3]string{
			"other: Praise for dark mode and a request for a sepia theme.",
			"other: Feature request for a sepia theme.",
			"technical: Asks about adding a sepia theme.",
		}),
		label("sup-07", "I can't log in: the code from my authenticator app is always rejected. I've already checked the clock on my phone.", "account", [3]string{
			"technical: Two-factor codes are rejected even with the phone's clock right.",
			"account: Can't sign in; authenticator codes are rejected.",
			"account: Locked out because 2FA codes are rejected.",
		}),
		label("sup-08", "We were billed for 12 seats but only have 9 people on the team. Please adjust it and credit us the difference.", "billing", [3]string{
			"billing: Charged for 12 seats with 9 members; wants a credit for 3.",
			"account: Wants the seat count lowered from 12 to 9.",
			"billing: Overcharged for seats; asks for a credit.",
		}),
		answer("qa-01", "What is the capital of Australia?", "Names Canberra. Does not say Sydney or Melbourne is the capital.", [3]string{
			"Canberra is the capital of Australia.",
			"The capital of Australia is Canberra.",
			"Sydney is the capital of Australia.",
		}, [3]float64{1, 1, 0}),
		answer("qa-02", "How many bytes are in a kibibyte?", "Says 1,024 bytes. May note that a kilobyte is 1,000.", [3]string{
			"A kibibyte is 1,024 bytes (2^10), unlike a kilobyte, which is 1,000.",
			"1,024 bytes.",
			"A kibibyte has 1,000 bytes.",
		}, [3]float64{1, 0.95, 0.1}),
		answer("qa-03", "Who wrote Pride and Prejudice?", "Jane Austen.", [3]string{
			"Jane Austen wrote Pride and Prejudice, published in 1813.",
			"Jane Austen.",
			"It was written by Jane Austen in 1813.",
		}, [3]float64{1, 1, 0.95}),
		answer("qa-04", "Why is the sky blue? Answer in one sentence.", "Gives Rayleigh scattering, or that shorter wavelengths scatter more, in one sentence.", [3]string{
			"Air molecules scatter the short blue wavelengths of sunlight far more than red ones, so blue light reaches us from all over the sky.",
			"Because of Rayleigh scattering: the atmosphere scatters blue light more than other colors.",
			"The sky is blue because it reflects the color of the oceans.",
		}, [3]float64{0.9, 0.95, 0.05}),
		answer("qa-05", "What does HTTP status 404 mean?", "The server cannot find the resource asked for.", [3]string{
			"Not Found: the server can't find the requested resource.",
			"It means the page was not found on the server.",
			"404 means the server is down.",
		}, [3]float64{1, 0.85, 0.15}),
		answer("qa-06", "Convert 100 °F to Celsius, rounded to one decimal place.", "37.8 °C.", [3]string{
			"100 °F is 37.8 °C.",
			"About 37.7 °C.",
			"37.8 °C",
		}, [3]float64{1, 0.4, 1}),
		answer("qa-07", "How many times does the letter r appear in \"strawberry\"?", "3.", [3]string{
			"The letter r appears twice in \"strawberry\".",
			"2",
			"There are 2 r's in strawberry.",
		}, [3]float64{0, 0, 0}),
	}}
}

type app struct {
	run     evals.Run
	evals   evals.State
	ratings map[evals.Cell]evals.Rating
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "support-triage, run 14").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Made-up outputs of three models over support emails and short questions").TextColor(t.TextMuted)
		})
		evals.View(c, &a.evals, &a.ratings, a.run).Grow(1).AlignSelf(ui.Stretch)
		up, down := 0, 0
		for _, r := range a.ratings {
			if r == evals.Up {
				up++
			} else {
				down++
			}
		}
		outputs := len(a.run.Cases) * len(a.run.Columns)
		ui.Text(c, strconv.Itoa(up+down)+" of "+strconv.Itoa(outputs)+" outputs rated, "+strconv.Itoa(up)+" up and "+
			strconv.Itoa(down)+" down · Click a cell or use the arrow keys, Esc to close").
			FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{run: run(), ratings: map[evals.Cell]evals.Rating{
		{Case: 3, Column: 0}:  evals.Up,
		{Case: 6, Column: 0}:  evals.Up,
		{Case: 11, Column: 0}: evals.Up,
		{Case: 9, Column: 1}:  evals.Down,
		{Case: 8, Column: 2}:  evals.Down,
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Evals",
			Width:   1440,
			Height:  900,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

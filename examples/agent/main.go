// Agent plays a coding agent fixing a failing test, as Claude Code and
// Cursor show one at work: its thinking, a plan, tool calls and a
// subagent's, the stack of the panic the tests hit, a card asking to run
// a command, the tests passing, and its reply, and beside them the files
// it changed, with the diff of the one chosen, and how full its context
// is. Send the message in the field to start it.
//
//	go run ./examples/agent
package main

import (
	"encoding/json"
	"log"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/xingkaixin/mygo-native-ui/ui/changes"
	"github.com/xingkaixin/mygo-native-ui/ui/confirm"
	"github.com/xingkaixin/mygo-native-ui/ui/contextmeter"
	"github.com/xingkaixin/mygo-native-ui/ui/plan"
	"github.com/xingkaixin/mygo-native-ui/ui/promptinput"
	"github.com/xingkaixin/mygo-native-ui/ui/stacktrace"
	"github.com/xingkaixin/mygo-native-ui/ui/testresults"
	"github.com/xingkaixin/mygo-native-ui/ui/trace"
)

// entry is a tool call of a run, or with sub, a subagent's task and the
// calls it made; a run of the tests shows the stack it panicked with, or
// the tests.
type entry struct {
	call    trace.Call
	open    bool
	sub     []*entry
	stack   *stacktrace.Trace
	tests   []testresults.Suite
	results testresults.State
}

// run is the agent's answer to a message, which script plays.
type run struct {
	prompt    string
	thought   trace.Thought
	thinking  bool
	reasoning string
	steps     []plan.Step
	showDone  bool
	entries   []*entry
	// into is where calls go: the run's entries, or a subagent's.
	into     *[]*entry
	asked    bool
	decision confirm.Decision
	reply    string
	replying bool
	next     int
	due      time.Time
}

func (r *run) done() bool { return r.next >= len(script) }
func (r *run) end()       { r.next = len(script) }

// step is a step of script: it changes the run, and returns how long
// until the next frame, and whether it is through, or runs again then,
// or with 0, at the next frame there is, as it waits for the user.
type step func(a *app, r *run) (time.Duration, bool)

var script []step

type app struct {
	runs    []*run
	list    ui.ListState
	prompt  promptinput.State
	model   string
	usage   contextmeter.Usage
	always  bool
	files   []changes.File
	diffs   map[string]*editor.DiffState
	changes changes.State
}

var models = []string{"Opus 5.5", "Sonnet 5.5", "Haiku 5.5"}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	var r *run
	if len(a.runs) > 0 {
		r = a.runs[len(a.runs)-1]
		a.advance(c, r)
	}
	busy := r != nil && !r.done()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			if len(a.runs) == 0 {
				ui.Column(c).Grow(1).Center().Children(func() {
					ui.Text(c, "Send the message below to watch the agent fix a failing test.").TextColor(t.TextMuted)
				})
			} else {
				ui.List(c, &a.list, len(a.runs), func(i int) { a.turn(c, a.runs[i]) }).Grow(1).Padding(t.Space(5), t.Space(6), 0)
			}
			ui.Box(c).Padding(t.Space(2), t.Space(6), t.Space(5)).Children(func() {
				promptinput.View(c, &a.prompt, "Ask the agent", busy, func() {
					ui.Select(c, &a.model, models).Label("Model")
					ui.Box(c).Grow(1)
					contextmeter.View(c, a.usage)
				})
			})
			if m, ok := a.prompt.Sent(); ok {
				a.runs = append(a.runs, &run{prompt: m.Text})
				a.spend(len(m.Text)/4, 0)
				c.Invalidate()
			}
			if a.prompt.Stopped() && busy {
				stop(r)
			}
		})
		ui.Box(c).Width(1).Background(t.Border)
		ui.Column(c).Width(400).Children(func() {
			changes.View(c, &a.changes, a.files).Height(t.Space(56))
			ui.Divider(c)
			if f, ok := a.changes.Chosen(); ok {
				editor.DiffView(c, a.diffs[f.Path]).Grow(1)
			} else {
				ui.Column(c).Grow(1).Center().Children(func() {
					ui.Text(c, "Choose a file to see its diff").TextColor(t.TextMuted)
				})
			}
		})
	})
}

// turn shows the message of r and the agent's answer.
func (a *app) turn(c *ui.Context, r *run) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(3)).Padding(0, 0, t.Space(6)).Children(func() {
		ui.Row(c).Justify(ui.End).Children(func() {
			ui.Text(c, r.prompt).Selectable().Shrink(1).MaxWidth(t.Space(120)).
				Padding(t.Space(2), t.Space(3)).Radius(t.Space(3)).Background(t.Surface)
		})
		if r.thinking || r.reasoning != "" {
			trace.Reasoning(c, &r.thought, r.reasoning, r.thinking)
		}
		if r.steps != nil {
			plan.View(c, "Fix the failing test", r.steps, &r.showDone)
		}
		ui.Column(c).Children(func() {
			for i, e := range r.entries {
				show(c.Key(i), e)
			}
		})
		if r.asked {
			ask := confirm.Ask(c, &r.decision, "Run this command?", "Always allow go test", func() {
				ui.Text(c, "$ "+testCommand).Font("monospace").FontSize(t.FontSize - 1).
					Padding(t.Space(2)).Radius(t.Space(1.5)).Background(t.Surface)
			})
			if ask.Changed() {
				c.Invalidate()
			}
		}
		if r.reply != "" {
			if r.replying {
				markdown.Stream(c, r.reply, nil)
			} else {
				markdown.View(c, r.reply, nil)
			}
		}
	})
}

// show shows e, and the calls of a subagent nested in it.
func show(c *ui.Context, e *entry) {
	switch {
	case e.stack != nil:
		trace.Step(c, e.call.Status, e.call.Name, command(e.call), &e.open, func() {
			if f, ok := stacktrace.View(c, *e.stack).Chosen(); ok {
				c.Toast("Open " + filepath.Base(f.File) + " at line " + strconv.Itoa(f.Line))
			}
		})
	case e.tests != nil:
		trace.Step(c, e.call.Status, e.call.Name, command(e.call), &e.open, func() {
			testresults.View(c, &e.results, e.tests)
		})
	case e.sub == nil:
		trace.Tool(c, e.call, &e.open)
	default:
		trace.Step(c, e.call.Status, e.call.Name, strconv.Itoa(len(e.sub))+" tools", &e.open, func() {
			for i, s := range e.sub {
				show(c.Key(i), s)
			}
		})
	}
}

// command returns the command call runs.
func command(call trace.Call) string {
	var in struct{ Command string }
	json.Unmarshal([]byte(call.Input), &in)
	return in.Command
}

// advance plays the steps of r due by now, and asks for a frame when the
// next is due.
func (a *app) advance(c *ui.Context, r *run) {
	now := c.Now()
	if r.into == nil {
		r.into = &r.entries
	}
	for !r.done() && !now.Before(r.due) {
		wait, through := script[r.next](a, r)
		if through {
			r.next++
		}
		r.due = now.Add(wait)
		if !through && wait == 0 {
			return
		}
	}
	if !r.done() {
		c.After(r.due.Sub(now))
	}
}

// stop ends r where it is, failing the calls under way.
func stop(r *run) {
	r.end()
	r.thinking, r.replying = false, false
	if r.asked && r.decision == confirm.Undecided {
		r.decision = confirm.Deny
	}
	var fail func([]*entry)
	fail = func(es []*entry) {
		for _, e := range es {
			if e.call.Status == trace.Running || e.call.Status == trace.Pending {
				e.call.Status, e.call.Err = trace.Failed, "Stopped by the user."
			}
			fail(e.sub)
		}
	}
	fail(r.entries)
	for i := range r.steps {
		if r.steps[i].Status == trace.Running {
			r.steps[i].Status = trace.Pending
		}
	}
	if r.reply == "" {
		r.reply = "*Stopped.*"
	}
}

// spend counts tokens read and written into the context, at Sonnet's
// prices.
func (a *app) spend(in, out int) {
	u := &a.usage
	u.Input += in
	u.Output += out
	u.Used += in + out
	u.Cost = (3*float64(u.Input) + 15*float64(u.Output) + 0.3*float64(u.Cached)) / 1e6
}

// change records the file at path changed from before to after.
func (a *app) change(path, before, after string, how changes.Change) {
	d := &editor.DiffState{Old: editor.NewBuffer(before), New: editor.NewBuffer(after), Layout: editor.DiffUnified, Wrap: true}
	added, removed := d.Stats()
	a.diffs[path] = d
	f := changes.File{Path: path, Change: how, Added: added, Removed: removed}
	if i := slices.IndexFunc(a.files, func(g changes.File) bool { return g.Path == path }); i >= 0 {
		a.files[i] = f
	} else {
		a.files = append(a.files, f)
	}
}

const ms = time.Millisecond

// do returns a step doing f, then waiting d.
func do(d time.Duration, f func(a *app, r *run)) step {
	return func(a *app, r *run) (time.Duration, bool) {
		f(a, r)
		return d, true
	}
}

// stream returns a step writing text into the string to returns, a word
// at a time, as a model streams it.
func stream(text string, to func(r *run) *string) step {
	return func(a *app, r *run) (time.Duration, bool) {
		s := to(r)
		rest := text[len(*s):]
		i := strings.IndexAny(rest, " \n")
		if i < 0 {
			*s = text
			return 200 * ms, true
		}
		*s += rest[:i+1]
		return 30 * ms, false
	}
}

// start returns a step starting a call to name with input, status
// Running or Pending, then waiting d.
func start(name, input string, status trace.Status, d time.Duration) step {
	return do(d, func(a *app, r *run) {
		*r.into = append(*r.into, &entry{call: trace.Call{Name: name, Input: input, Status: status}})
	})
}

// finish returns a step ending the call last started, with output, or
// failing with err, and doing then.
func finish(output, err string, then func(a *app)) step {
	return do(300*ms, func(a *app, r *run) {
		e := (*r.into)[len(*r.into)-1]
		e.call.Status, e.call.Output, e.call.Err = trace.Done, output, err
		if err != "" {
			e.call.Status = trace.Failed
		}
		a.spend(1_500+len(output+err)/4, 120)
		if then != nil {
			then(a)
		}
	})
}

// tool returns the steps of a call taking d.
func tool(name, input, output, err string, d time.Duration, then func(a *app)) []step {
	return []step{start(name, input, trace.Running, d), finish(output, err, then)}
}

// progress returns a step marking step i-1 of the plan done and i under
// way.
func progress(i int) step {
	return do(0, func(a *app, r *run) {
		r.steps[i-1].Status = trace.Done
		if i < len(r.steps) {
			r.steps[i].Status = trace.Running
		}
	})
}

// subagent returns the steps of a subagent doing task in steps.
func subagent(task string, steps ...[]step) []step {
	begin := do(400*ms, func(a *app, r *run) {
		e := &entry{call: trace.Call{Name: task, Status: trace.Running}, sub: []*entry{}, open: true}
		r.entries = append(r.entries, e)
		r.into = &e.sub
	})
	end := do(300*ms, func(a *app, r *run) {
		r.entries[len(r.entries)-1].call.Status = trace.Done
		r.into = &r.entries
	})
	return slices.Concat(append(append([][]step{{begin}}, steps...), []step{end})...)
}

// ask waits for the user to allow the command the call last started runs,
// unless they always allow it; denied, the run ends there.
func ask(a *app, r *run) (time.Duration, bool) {
	if !a.always {
		r.asked = true
		switch r.decision {
		case confirm.Undecided:
			return 0, false
		case confirm.Deny:
			e := r.entries[len(r.entries)-1]
			e.call.Status, e.call.Err = trace.Failed, "The user denied running it."
			r.steps[len(r.steps)-1].Status = trace.Pending
			r.reply = "I fixed `Save` but didn't run the tests. Run `" + testCommand + "` to check it."
			r.end()
			return 0, false
		case confirm.AlwaysAllow:
			a.always = true
		}
	}
	r.entries[len(r.entries)-1].call.Status = trace.Running
	return 500 * ms, true
}

// runTests runs the tests of passing one at a time, as go test -v reports
// them, in the call last started.
func runTests(a *app, r *run) (time.Duration, bool) {
	e := r.entries[len(r.entries)-1]
	if e.tests == nil {
		e.tests, e.open = []testresults.Suite{{Name: "example.com/notes"}}, true
	}
	s := &e.tests[0]
	if n := len(s.Tests); n > 0 && s.Tests[n-1].Status == testresults.Running {
		s.Tests[n-1] = passing[n-1]
	}
	if len(s.Tests) == len(passing) {
		e.call.Status, e.call.Output = trace.Done, "ok  \texample.com/notes\t0.011s"
		a.spend(1_200, 120)
		return 300 * ms, true
	}
	s.Tests = append(s.Tests, testresults.Test{Name: passing[len(s.Tests)].Name})
	return 220 * ms, false
}

var passing = []testresults.Test{
	{Name: "TestSaveKeepsLast", Status: testresults.Passed, Duration: 2 * ms},
	{Name: "TestSaveEmpty", Status: testresults.Passed, Duration: 1 * ms},
	{Name: "TestLoad", Status: testresults.Passed, Duration: 1 * ms},
	{Name: "TestLoadMissingFile", Status: testresults.Passed, Duration: 0},
	{Name: "TestLoadCorrupt", Status: testresults.Skipped, Output: "load_test.go:41: needs the corrupt fixture; run with -fixtures"},
}

const testCommand = "go test ./notes/... -count=1"

const reasoning = "The notes tests fail, so something goes wrong between `Save` and `Load`. " +
	"I'll run them to see how, read the test, then find where notes are saved.\n\n" +
	"While I'm there, `Save` writes the file in place, so a crash halfway would leave it broken; " +
	"writing to a temporary file and renaming it fixes that too."

const reply = "Fixed. `Save` copied the notes with a loop that ran one past the end, so it panicked with an index out of range:\n\n" +
	"- **notes/store.go**: marshal the notes as they are, and write them with `fsutil.WriteFileAtomic`.\n" +
	"- **internal/fsutil/atomic.go**: new; writes to a temporary file and renames it over the old one, so a crash can't leave half a file.\n" +
	"- **notes/store_test.go**: adds `TestSaveEmpty`.\n\n" +
	"`" + testCommand + "` passes."

const testPanic = `--- FAIL: TestSaveKeepsLast (0.00s)
panic: runtime error: index out of range [3] with length 3 [recovered]
	panic: runtime error: index out of range [3] with length 3

goroutine 7 [running]:
testing.tRunner.func1.2({0x1003e5f60, 0x14000116048})
	/usr/local/go/src/testing/testing.go:1632 +0x1bc
testing.tRunner.func1()
	/usr/local/go/src/testing/testing.go:1635 +0x334
panic({0x1003e5f60?, 0x14000116048?})
	/usr/local/go/src/runtime/panic.go:785 +0x124
example.com/notes.Save({0x1400011e0c0, 0x22}, {0x1400010c0f0, 0x3, 0x3})
	/Users/dev/notes/notes/store.go:18 +0x1d4
example.com/notes.TestSaveKeepsLast(0x14000130340)
	/Users/dev/notes/notes/store_test.go:11 +0x10c
testing.tRunner(0x14000130340, 0x1004ecf08)
	/usr/local/go/src/testing/testing.go:1690 +0xe4
created by testing.(*T).Run in goroutine 1
	/usr/local/go/src/testing/testing.go:1743 +0x314
exit status 2
FAIL	example.com/notes	0.012s`

func init() {
	script = slices.Concat(
		[]step{
			do(0, func(a *app, r *run) { r.thinking = true }),
			stream(reasoning, func(r *run) *string { return &r.reasoning }),
			do(300*ms, func(a *app, r *run) {
				r.thinking = false
				r.steps = []plan.Step{
					{Title: "Run the tests and read the failure", Status: trace.Running},
					{Title: "Find where notes are saved", Status: trace.Pending},
					{Title: "Fix the off-by-one in Save", Status: trace.Pending},
					{Title: "Run the tests again", Status: trace.Pending},
				}
				a.spend(400, 380)
			}),
		},
		[]step{
			start("bash", `{"command":"go test ./notes/..."}`, trace.Running, 1100*ms),
			do(300*ms, func(a *app, r *run) {
				e := r.entries[len(r.entries)-1]
				tr := stacktrace.Parse(testPanic)
				e.call.Status, e.call.Err, e.stack, e.open = trace.Failed, testPanic, &tr, true
				a.spend(1_500+len(testPanic)/4, 120)
			}),
		},
		tool("read_file", `{"path":"notes/store_test.go"}`, testBefore, "", 500*ms, nil),
		[]step{progress(1)},
		subagent("Explore where notes are saved",
			tool("grep", `{"pattern":"func (Save|Load)","path":"."}`,
				"notes/store.go:15: func Save(path string, notes []Note) error\nnotes/load.go:9: func Load(path string) ([]Note, error)", "", 700*ms, nil),
			tool("read_file", `{"path":"notes/store.go"}`, storeBefore, "", 500*ms, nil),
			tool("read_file", `{"path":"notes/load.go"}`, "package notes\n\n// Load reads the notes Save wrote to path.\nfunc Load(path string) ([]Note, error) { … }", "", 400*ms, nil),
		),
		[]step{progress(2)},
		tool("write_file", `{"path":"internal/fsutil/atomic.go"}`, "Created internal/fsutil/atomic.go", "", 700*ms, func(a *app) {
			a.change("internal/fsutil/atomic.go", "", atomicAfter, changes.Added)
		}),
		tool("edit_file", `{"path":"notes/store.go"}`, "Edited notes/store.go", "", 600*ms, func(a *app) {
			a.change("notes/store.go", storeBefore, storeAfter, changes.Modified)
		}),
		tool("edit_file", `{"path":"notes/store_test.go"}`, "Edited notes/store_test.go", "", 500*ms, func(a *app) {
			a.change("notes/store_test.go", testBefore, testAfter, changes.Modified)
		}),
		[]step{
			progress(3),
			start("bash", `{"command":"`+testCommand+`"}`, trace.Pending, 0),
			ask,
			runTests,
			progress(4),
			do(0, func(a *app, r *run) { r.replying = true }),
			stream(reply, func(r *run) *string { return &r.reply }),
			do(0, func(a *app, r *run) {
				r.replying = false
				a.spend(0, 260)
			}),
		},
	)
}

func main() {
	a := &app{
		model:  models[1],
		prompt: promptinput.State{Text: "The notes tests fail since yesterday. Can you fix them?"},
		usage:  contextmeter.Usage{Used: 18_400, Window: 200_000, Input: 6_400, Cached: 12_000},
		diffs:  map[string]*editor.DiffState{},
	}
	a.spend(0, 0)
	a.list.FollowEnd = true
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Agent",
			Width:   1180,
			Height:  780,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

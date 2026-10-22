package flamechart

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"slices"
	"strings"
	"time"
)

// Trace is a run traced: its threads, each a lane of the view, and when
// the last call ended, from the start of the first.
type Trace struct {
	Threads []*Thread
	End     time.Duration
	calls   int
}

// Thread is a thread of a trace: its name and its calls, a row for each
// depth of the stack, by when they started.
type Thread struct {
	Name string
	rows [][]Call
}

// Call is a call on a thread: its name and category, when it started,
// from the start of the trace, how long it took, and how long of that it
// ran itself, not in the calls it made; how deep in the stack it ran, 0
// at the bottom, and the arguments the trace gave it.
type Call struct {
	Name     string
	Category string
	Start    time.Duration
	Duration time.Duration
	Self     time.Duration
	Depth    int
	Args     map[string]any
	hue      float32
	shade    float32
}

// End returns when c ended, from the start of the trace.
func (c *Call) End() time.Duration { return c.Start + c.Duration }

// Calls returns how many calls the trace has.
func (tr *Trace) Calls() int { return tr.calls }

// event is an event of the Trace Event Format.
type event struct {
	Name string          `json:"name"`
	Cat  string          `json:"cat"`
	Ph   string          `json:"ph"`
	Ts   float64         `json:"ts"`
	Dur  float64         `json:"dur"`
	Pid  json.RawMessage `json:"pid"`
	Tid  json.RawMessage `json:"tid"`
	Args map[string]any  `json:"args"`
}

// Parse reads a trace in the Trace Event Format, as Chrome's tracing,
// Perfetto and many profilers write it: a JSON array of events, or an
// object with them in "traceEvents". It takes complete events ("X"),
// begin and end pairs ("B" and "E") and the names of threads
// ("thread_name" metadata), and leaves the rest. It returns a thread for
// each that made a call, in the order they first come, named "Thread"
// and its id when the trace does not name it.
func Parse(r io.Reader) (*Trace, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("flamechart: %w", err)
	}
	var events []event
	if data = bytes.TrimSpace(data); len(data) > 0 && data[0] == '{' {
		var doc struct {
			TraceEvents []event `json:"traceEvents"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("flamechart: %w", err)
		}
		events = doc.TraceEvents
	} else {
		// A tracer stopped mid-write leaves the array unclosed, or ending
		// with a comma, which the format allows.
		data = bytes.TrimRight(data, ", \t\r\n")
		if !bytes.HasSuffix(data, []byte("]")) {
			data = append(data, ']')
		}
		if err := json.Unmarshal(data, &events); err != nil {
			return nil, fmt.Errorf("flamechart: %w", err)
		}
	}

	type thread struct {
		name  string
		calls []Call
		open  []event
	}
	threads := map[string]*thread{}
	var order []string
	lane := func(e event) *thread {
		key := string(e.Pid) + "/" + string(e.Tid)
		th := threads[key]
		if th == nil {
			th = &thread{}
			threads[key] = th
			order = append(order, key)
		}
		return th
	}
	us := func(v float64) time.Duration { return time.Duration(math.Round(v * 1000)) }
	first := time.Duration(math.MaxInt64)
	for _, e := range events {
		th := lane(e)
		switch e.Ph {
		case "X":
			th.calls = append(th.calls, Call{Name: e.Name, Category: e.Cat, Start: us(e.Ts), Duration: us(e.Dur), Args: e.Args})
		case "B":
			th.open = append(th.open, e)
			continue
		case "E":
			if len(th.open) == 0 {
				continue
			}
			b := th.open[len(th.open)-1]
			th.open = th.open[:len(th.open)-1]
			if b.Name == "" {
				b.Name = e.Name
			}
			if len(e.Args) > 0 && b.Args == nil {
				b.Args = map[string]any{}
			}
			for k, v := range e.Args {
				b.Args[k] = v
			}
			th.calls = append(th.calls, Call{Name: b.Name, Category: b.Cat, Start: us(b.Ts), Duration: us(e.Ts - b.Ts), Args: b.Args})
		case "M":
			if name, ok := e.Args["name"].(string); ok && e.Name == "thread_name" {
				th.name = name
			}
			continue
		default:
			continue
		}
		first = min(first, th.calls[len(th.calls)-1].Start)
	}

	tr := &Trace{}
	categories := map[string]int{}
	for _, key := range order {
		th := threads[key]
		for _, c := range th.calls {
			if _, ok := categories[c.Category]; !ok && c.Category != "" {
				categories[c.Category] = len(categories)
			}
		}
	}
	for _, key := range order {
		th := threads[key]
		if len(th.calls) == 0 {
			continue
		}
		name := th.name
		if name == "" {
			_, tid, _ := strings.Cut(key, "/")
			name = "Thread " + strings.Trim(tid, `"`)
		}
		for i := range th.calls {
			th.calls[i].Start -= first
		}
		t := &Thread{Name: name, rows: nest(th.calls)}
		for _, row := range t.rows {
			for i := range row {
				c := &row[i]
				c.hue, c.shade = tone(c, categories)
				tr.End = max(tr.End, c.End())
			}
			tr.calls += len(row)
		}
		tr.Threads = append(tr.Threads, t)
	}
	return tr, nil
}

// nest stacks the calls of a thread by when they ran: a call inside
// another a row deeper, and returns the rows, each by when its calls
// started, with how deep each is and its self time.
func nest(calls []Call) [][]Call {
	slices.SortStableFunc(calls, func(a, b Call) int {
		if a.Start != b.Start {
			return cmp.Compare(a.Start, b.Start)
		}
		return cmp.Compare(b.Duration, a.Duration)
	})
	var stack []int
	depth := 0
	for i := range calls {
		c := &calls[i]
		for len(stack) > 0 && calls[stack[len(stack)-1]].End() <= c.Start {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			parent := &calls[stack[len(stack)-1]]
			// A call that outlives its parent, as clocks of two sources
			// can make it, would overlap the next call in its row.
			c.Duration = min(c.Duration, parent.End()-c.Start)
			parent.Self -= c.Duration
		}
		c.Self = c.Duration
		c.Depth = len(stack)
		depth = max(depth, c.Depth+1)
		stack = append(stack, i)
	}
	rows := make([][]Call, depth)
	for _, c := range calls {
		rows[c.Depth] = append(rows[c.Depth], c)
	}
	return rows
}

// tone returns the hue of c and how much lighter or darker than its
// category's its bar is: a hue for each category, by the order they come,
// or for a call without one, a hue for its name, shaded by its name, so
// that calls of one category tell apart and keep their colors.
func tone(c *Call, categories map[string]int) (hue, shade float32) {
	h := fnv.New32a()
	h.Write([]byte(c.Name))
	v := h.Sum32()
	shade = float32(v>>8%7)/100 - 0.03
	if k, ok := categories[c.Category]; ok && len(categories) > 1 {
		return float32((200+k*137)%360) + float32(v%21) - 10, shade
	}
	return float32(v % 360), shade
}

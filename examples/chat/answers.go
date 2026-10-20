package main

// The answers the model gives, and the sources it read for them.

import (
	"strings"

	"github.com/xingkaixin/mygo-native-ui/ui/sources"
)

// answer is a reply's paragraphs of Markdown, each citing sources by
// their index.
type answer struct {
	paras   []para
	sources []sources.Source
}

type para struct {
	text  string
	cites []int
}

func (a *answer) words() int {
	n := 0
	for _, p := range a.paras {
		n += len(strings.SplitAfter(p.text, " "))
	}
	return n
}

func (a *answer) text() string {
	var b strings.Builder
	for i, p := range a.paras {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(p.text)
	}
	return b.String()
}

// cut returns a, its first n words, as a reply stopped there.
func (a *answer) cut(n int) *answer {
	out := &answer{sources: a.sources}
	for _, p := range a.paras {
		words := strings.SplitAfter(p.text, " ")
		if n < len(words) {
			if n > 0 {
				out.paras = append(out.paras, para{text: strings.Join(words[:n], "")})
			}
			break
		}
		out.paras = append(out.paras, p)
		n -= len(words)
	}
	return out
}

// answerFor returns the answer to prompt, worded as its nth reply.
func answerFor(prompt string, n int) *answer {
	p := strings.ToLower(prompt)
	var choices []*answer
	switch {
	case strings.Contains(p, "iterat") || strings.Contains(p, "range"):
		choices = iterators
	case strings.Contains(p, "log"):
		choices = logging
	default:
		return unknown
	}
	return choices[n%len(choices)]
}

var iteratorSources = []sources.Source{
	{Title: "Range Over Function Types", URL: "https://go.dev/blog/range-functions",
		Snippet: "Go 1.23 lets for-range loops range over functions, so loops can range over iterators of any kind."},
	{Title: "Go 1.23 Release Notes", URL: "https://go.dev/doc/go1.23",
		Snippet: "The range clause in a for-range loop now accepts iterator functions."},
	{Title: "iter package", URL: "https://pkg.go.dev/iter",
		Snippet: "Package iter provides basic definitions and operations related to iterators over sequences."},
}

var iterators = []*answer{
	{sources: iteratorSources, paras: []para{
		{"Since Go 1.23, a `for range` loop can range over a function. An iterator is a function that takes a `yield` callback and calls it once for each value: `iter.Seq[V]` is `func(yield func(V) bool)`, and `iter.Seq2[K, V]` yields two values at a time.", []int{0, 2}},
		{"`yield` returns false once the loop stops early, as with `break`, and the iterator must then return without calling it again.", []int{0}},
		{"The standard library returns iterators from `slices.All`, `slices.Values`, `maps.Keys` and `maps.Values`, and `slices.Collect` turns one back into a slice.", []int{1}},
	}},
	{sources: iteratorSources, paras: []para{
		{"Write a function that returns an `iter.Seq`, and range over what it returns as over a slice:", []int{2}},
		{"```go\nfunc Countdown(n int) iter.Seq[int] {\n\treturn func(yield func(int) bool) {\n\t\tfor i := n; i > 0; i-- {\n\t\t\tif !yield(i) {\n\t\t\t\treturn\n\t\t\t}\n\t\t}\n\t}\n}\n\nfor i := range Countdown(3) {\n\tfmt.Println(i)\n}\n```", nil},
		{"Check what `yield` returns: false means the loop broke out early, and the iterator should stop.", []int{0, 1}},
	}},
}

var loggingSources = []sources.Source{
	{Title: "Structured Logging with slog", URL: "https://go.dev/blog/slog",
		Snippet: "The new log/slog package in Go 1.21 brings structured logging to the standard library."},
	{Title: "slog package", URL: "https://pkg.go.dev/log/slog",
		Snippet: "Package slog provides structured logging, in which log records include a message, a severity level, and various other attributes expressed as key-value pairs."},
	{Title: "Go 1.21 Release Notes", URL: "https://go.dev/doc/go1.21",
		Snippet: "The new log/slog package provides structured logging with levels."},
}

var logging = []*answer{
	{sources: loggingSources, paras: []para{
		{"Use `log/slog`, in the standard library since Go 1.21. Its functions take a message and then keys and values in turn: `slog.Info(\"saved\", \"path\", path, \"bytes\", n)`.", []int{0, 2}},
		{"A handler decides the format: `slog.NewTextHandler` writes `key=value` pairs and `slog.NewJSONHandler` writes JSON. Make one the default with `slog.SetDefault(slog.New(h))`.", []int{1}},
		{"To add the same attributes to every record, derive a logger with `logger.With(\"request\", id)`.", []int{0}},
	}},
	{sources: loggingSources, paras: []para{
		{"`log/slog` is the standard library's structured logger, since Go 1.21:", []int{2}},
		{"```go\nlogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))\nlogger.Info(\"saved\", \"path\", path, \"bytes\", n)\n```", nil},
		{"This prints a JSON object with `time`, `level`, `msg`, `path` and `bytes` fields, one a line.", []int{1}},
	}},
}

var unknown = &answer{paras: []para{
	{"This demo answers two questions: how Go's iterators work, and how to log structured data. Ask about either, or edit your message.", nil},
}}

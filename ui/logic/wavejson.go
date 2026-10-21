package logic

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseWaveJSON reads a timing diagram in WaveDrom's WaveJSON, each of
// its cycles lasting cycle seconds:
//
//	{signal: [
//		{name: "clk",  wave: "p.....|..."},
//		{name: "data", wave: "x.=.=.|=.x", data: ["head", "body", "tail"]},
//		{},
//		["Slow", {name: "ack", wave: "0.1.0", period: 2}],
//	]}
//
// Each character of a wave lasts a cycle, or period cycles: 0 and 1, or l
// and h, are low and high, x unknown and z high impedance; p and n are a
// cycle of a clock rising or falling first; = and 2 to 9 are a value of a
// bus, labeled with the next of data, a list or words between spaces; .
// goes on with what is before it, a clock too, and | does the same with a
// break drawn in it. A signal with any bus value is a bus of 0 Bits,
// which writes only its labels. An empty object is an empty row, and a
// list a group, whose signals follow its name in order. Keys need no
// quotes, strings may take single quotes, and comments and commas before
// a closing bracket are allowed, as in WaveDrom's own files. The error
// says on which line what is wrong.
func ParseWaveJSON(src string, cycle float64) (capture Capture, err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(syntaxError)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	p := &parser{src: src}
	v := p.value()
	p.space()
	if p.pos < len(src) {
		p.fail("%s after the end of the diagram", p.what())
	}
	root, ok := v.(*object)
	if !ok {
		p.failAt(0, "WaveJSON is an object, as {signal: [...]}")
	}
	list, ok := root.fields["signal"].([]any)
	if !ok {
		p.failAt(root.at, `the diagram has no "signal" list`)
	}
	p.signals(list, cycle, &capture)
	return capture, nil
}

// signals adds to capture the signals of list and of the groups in it.
func (p *parser) signals(list []any, cycle float64, capture *Capture) {
	for i, item := range list {
		switch v := item.(type) {
		case *object:
			sig, length := p.signal(v, cycle)
			capture.Signals = append(capture.Signals, sig)
			capture.End = max(capture.End, length)
		case []any:
			p.signals(v, cycle, capture)
		case string:
			if i > 0 {
				p.failAt(-1, "a group's name %q comes first in its list", v)
			}
		default:
			p.failAt(-1, "a signal is an object, as {name: \"clk\", wave: \"p...\"}")
		}
	}
}

// signal reads a signal's object, and returns it and how long it lasts.
func (p *parser) signal(o *object, cycle float64) (Signal, float64) {
	str := func(key string) string {
		switch v := o.fields[key].(type) {
		case nil:
			return ""
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'g', -1, 64)
		}
		p.failAt(o.at, "%s is not text", key)
		return ""
	}
	sig := Signal{Name: str("name"), Bits: 1}
	wave := str("wave")
	period := 1.0
	switch v := o.fields["period"].(type) {
	case nil:
	case float64:
		if v <= 0 {
			p.failAt(o.at, "the period of %q is not above 0", sig.Name)
		}
		period = v
	default:
		p.failAt(o.at, "the period of %q is not a number", sig.Name)
	}
	var data []string
	switch v := o.fields["data"].(type) {
	case nil:
	case string:
		data = strings.Fields(v)
	case []any:
		for _, d := range v {
			switch d := d.(type) {
			case string:
				data = append(data, d)
			case float64:
				data = append(data, strconv.FormatFloat(d, 'g', -1, 64))
			default:
				p.failAt(o.at, "the data of %q are not text", sig.Name)
			}
		}
	default:
		p.failAt(o.at, "the data of %q are not a list", sig.Name)
	}

	span := period * cycle
	add := func(at float64, ch Change) {
		ch.At = at
		sig.Changes = append(sig.Changes, ch)
	}
	// clock is the clock, 'p' or 'n', that a . goes on with.
	var clock byte
	tick := func(at float64) {
		first, then := uint64(1), uint64(0)
		if clock == 'n' {
			first, then = 0, 1
		}
		add(at, Change{Value: first})
		add(at+span/2, Change{Value: then})
	}
	bus, next := false, 0
	for i := 0; i < len(wave); i++ {
		at := float64(i) * span
		switch r := wave[i]; r {
		case '0', 'l', 'L':
			clock = 0
			add(at, Change{})
		case '1', 'h', 'H':
			clock = 0
			add(at, Change{Value: 1})
		case 'x':
			clock = 0
			add(at, Change{Level: Unknown})
		case 'z':
			clock = 0
			add(at, Change{Level: Floating})
		case 'p', 'P':
			clock = 'p'
			tick(at)
		case 'n', 'N':
			clock = 'n'
			tick(at)
		case '=', '2', '3', '4', '5', '6', '7', '8', '9':
			clock, bus = 0, true
			ch := Change{}
			if next < len(data) {
				ch.Label = data[next]
			}
			next++
			add(at, ch)
		case '.', '|':
			switch {
			case clock != 0:
				tick(at)
			case i == 0:
				add(at, Change{Level: Unknown})
			}
			if r == '|' {
				sig.Gaps = append(sig.Gaps, at+span/2)
			}
		default:
			p.failAt(o.at, "%q in the wave of %q is not one of 0 1 x z . = 2–9 p n h l |", r, sig.Name)
		}
	}
	if bus {
		sig.Bits = 0
	}
	// A bit holding its value, as 1 after h, makes no edge; a bus value
	// does, even when it is the same as the one before.
	kept := sig.Changes[:0]
	for _, ch := range sig.Changes {
		if k := len(kept); k > 0 && (!bus || ch.Level != Driven) && kept[k-1].Value == ch.Value && kept[k-1].Level == ch.Level {
			continue
		}
		kept = append(kept, ch)
	}
	sig.Changes = kept
	return sig, float64(len(wave)) * span
}

// object is a JSON object as parsed, and where it begins in the source.
type object struct {
	fields map[string]any
	at     int
}

// syntaxError is what is wrong with a source, where the parse stopped.
type syntaxError string

func (e syntaxError) Error() string { return string(e) }

// parser reads JSON5, the JSON that WaveDrom takes: keys without quotes,
// single quotes, comments and trailing commas. Objects come as *object,
// lists as []any, strings as string, numbers as float64, and true, false
// and null as bool and nil.
type parser struct {
	src string
	pos int
}

// fail stops the parse with the error, on the line the parser is on.
func (p *parser) fail(format string, args ...any) { p.failAt(p.pos, format, args...) }

// failAt stops the parse with the error, on the line of byte at, or on
// none for an at below 0.
func (p *parser) failAt(at int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if at >= 0 {
		msg = fmt.Sprintf("Line %d: %s", 1+strings.Count(p.src[:min(at, len(p.src))], "\n"), msg)
	}
	panic(syntaxError(msg))
}

// what names what is at the parser, for errors.
func (p *parser) what() string {
	if p.pos >= len(p.src) {
		return "the end"
	}
	return strconv.QuoteRune(rune(p.src[p.pos]))
}

// space skips spaces and comments.
func (p *parser) space() {
	for p.pos < len(p.src) {
		switch rest := p.src[p.pos:]; {
		case strings.HasPrefix(rest, "//"):
			if i := strings.IndexByte(rest, '\n'); i >= 0 {
				p.pos += i
			} else {
				p.pos = len(p.src)
			}
		case strings.HasPrefix(rest, "/*"):
			i := strings.Index(rest[2:], "*/")
			if i < 0 {
				p.fail("a comment has no */")
			}
			p.pos += i + 4
		case strings.ContainsRune(" \t\r\n", rune(rest[0])):
			p.pos++
		default:
			return
		}
	}
}

// value reads the value at the parser, after spaces and comments.
func (p *parser) value() any {
	p.space()
	if p.pos >= len(p.src) {
		p.fail("the diagram ends early")
	}
	switch ch := p.src[p.pos]; {
	case ch == '{':
		return p.object()
	case ch == '[':
		return p.list()
	case ch == '"' || ch == '\'':
		return p.string()
	case ch == '-' || ch == '+' || ch == '.' || ch >= '0' && ch <= '9':
		return p.number()
	case isIdent(ch):
		switch w := p.ident(); w {
		case "true", "false":
			return w == "true"
		case "null":
			return nil
		default:
			p.failAt(p.pos-len(w), "%s is not a value; text goes in quotes", w)
		}
	}
	p.fail("%s does not begin a value", p.what())
	return nil
}

// object reads an object from its {.
func (p *parser) object() *object {
	o := &object{fields: map[string]any{}, at: p.pos}
	p.pos++
	for {
		p.space()
		if p.pos >= len(p.src) {
			p.fail("an object has no }")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			return o
		}
		var key string
		switch ch := p.src[p.pos]; {
		case ch == '"' || ch == '\'':
			key = p.string()
		case isIdent(ch):
			key = p.ident()
		default:
			p.fail("%s is not a key", p.what())
		}
		p.space()
		if p.pos >= len(p.src) || p.src[p.pos] != ':' {
			p.fail("a : goes after the key %s", key)
		}
		p.pos++
		o.fields[key] = p.value()
		p.space()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
		} else if p.pos >= len(p.src) || p.src[p.pos] != '}' {
			p.fail("a , or } goes after the value of %s, not %s", key, p.what())
		}
	}
}

// list reads a list from its [.
func (p *parser) list() []any {
	p.pos++
	out := []any{}
	for {
		p.space()
		if p.pos >= len(p.src) {
			p.fail("a list has no ]")
		}
		if p.src[p.pos] == ']' {
			p.pos++
			return out
		}
		out = append(out, p.value())
		p.space()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
		} else if p.pos >= len(p.src) || p.src[p.pos] != ']' {
			p.fail("a , or ] goes between the items of a list, not %s", p.what())
		}
	}
}

// string reads a string in single or double quotes, with JSON's escapes.
func (p *parser) string() string {
	quote := p.src[p.pos]
	start := p.pos
	p.pos++
	var b strings.Builder
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		switch {
		case ch == quote:
			p.pos++
			return b.String()
		case ch == '\n':
			p.failAt(start, "a string has no closing quote")
		case ch == '\\' && p.pos+1 < len(p.src):
			p.pos++
			switch e := p.src[p.pos]; e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'u':
				if p.pos+5 > len(p.src) {
					p.fail("\\u needs four hexadecimal digits")
				}
				r, err := strconv.ParseUint(p.src[p.pos+1:p.pos+5], 16, 32)
				if err != nil {
					p.fail("\\u needs four hexadecimal digits")
				}
				b.WriteRune(rune(r))
				p.pos += 4
			default:
				b.WriteByte(e)
			}
			p.pos++
		default:
			b.WriteByte(ch)
			p.pos++
		}
	}
	p.failAt(start, "a string has no closing quote")
	return ""
}

// number reads a number, as JSON writes them.
func (p *parser) number() float64 {
	start := p.pos
	for p.pos < len(p.src) && strings.IndexByte("+-.0123456789eE", p.src[p.pos]) >= 0 {
		p.pos++
	}
	v, err := strconv.ParseFloat(p.src[start:p.pos], 64)
	if err != nil {
		p.failAt(start, "%s is not a number", p.src[start:p.pos])
	}
	return v
}

// ident reads a name without quotes, as a key or true.
func (p *parser) ident() string {
	start := p.pos
	for p.pos < len(p.src) && (isIdent(p.src[p.pos]) || p.src[p.pos] >= '0' && p.src[p.pos] <= '9') {
		p.pos++
	}
	return p.src[start:p.pos]
}

// isIdent reports whether ch may begin a name without quotes.
func isIdent(ch byte) bool {
	return ch == '_' || ch == '$' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

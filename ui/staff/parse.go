package staff

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// whole is how many ticks a whole note lasts: enough for 64ths, their
// dots and thirds of them.
const whole = 3840

type problem struct {
	line int
	msg  string
}

type bar uint8

const (
	noBar    bar = iota
	thin         // |
	double       // ||
	final        // |]
	heavy        // [|
	repStart     // |:
	repEnd       // :|
	repBoth      // ::
)

var bars = []struct {
	text string
	kind bar
}{
	{":||:", repBoth}, {":|:", repBoth}, {"::", repBoth}, {":|", repEnd}, {"|:", repStart},
	{"|]", final}, {"||", double}, {"[|", heavy}, {"|", thin},
}

type accidental uint8

const (
	noAcc accidental = iota
	sharp
	flat
	natural
	doubleSharp
	doubleFlat
)

var alters = map[accidental]int{sharp: 1, flat: -1, natural: 0, doubleSharp: 2, doubleFlat: -2}

// head is a note of an event: its step, counted in letters from C0 so
// that middle C is 28, the semitones the key, the bar or its accidental
// move it by, the accidental written and whether a tie goes on from it.
type head struct {
	step, alter int
	acc         accidental
	tie         bool
}

var semitones = [7]int{0, 2, 4, 5, 7, 9, 11}

func (h head) pitch() int { return 12*(h.step/7+1) + semitones[h.step%7] + h.alter }

func (h head) name() string {
	return string("CDEFGAB"[h.step%7]) + map[int]string{-2: "♭♭", -1: "♭", 1: "♯", 2: "♯♯"}[h.alter] + strconv.Itoa(h.step/7)
}

// event is a note, a chord or a rest: where it starts in the source, its
// heads lowest first, none for a rest, how long it lasts as written and
// as played, in ticks, and whether it is written against the event
// before it, which beams them.
type event struct {
	line, col int
	heads     []head
	written   int
	ticks     int
	triplet   bool
	joined    bool
}

// measure is the events from to to, and the bars around them: open only
// before the first measure.
type measure struct {
	from, to    int
	open, close bar
}

type meter struct {
	num, den int
	// symbol is 'C' for common time, '|' for cut time, 0 for numbers.
	symbol byte
}

type tune struct {
	title, composer string
	meter           meter
	// beat is the ticks of the tempo's beat, bpm how many a minute; tempo
	// tells whether the source gives them.
	beat, bpm int
	tempo     bool
	fifths    int
	bass      bool
	events    []event
	measures  []measure
	triplets  [][2]int
	errs      []problem
}

var (
	sharps = [7]int{3, 0, 4, 1, 5, 2, 6} // F C G D A E B
	flats  = [7]int{6, 2, 5, 1, 4, 0, 3} // B E A D G C F
)

// keyAlter returns what the key does to letter, C 0 to B 6.
func (t *tune) keyAlter(letter int) int {
	switch {
	case t.fifths > 0 && slices.Index(sharps[:], letter) < t.fifths:
		return 1
	case t.fifths < 0 && slices.Index(flats[:], letter) < -t.fifths:
		return -1
	}
	return 0
}

// value returns the note an event of ticks shows, 1 for a whole, 2 for a
// half and so on to 64, and its dots.
func value(ticks int) (kind, dots int, ok bool) {
	for kind = 1; kind <= 64; kind *= 2 {
		b := whole / kind
		switch ticks {
		case b:
			return kind, 0, true
		case b * 3 / 2:
			return kind, 1, true
		case b * 7 / 4:
			return kind, 2, true
		}
	}
	return 0, 0, false
}

type parser struct {
	t    *tune
	line int
	// failed are the lines with a problem, so that each lists its first.
	failed map[int]bool
	body   bool
	unit   int
	// accs are the accidentals of the bar so far, by step.
	accs    map[int]int
	from    int
	open    bar
	joined  bool
	left    int // events still to come in a triplet
	first   int // the triplet's first event
	broken  int // > is 1, >> 2, < -1
	tied    bool
	tieLine int
}

func (p *parser) fail(format string, args ...any) { p.failAt(p.line, format, args...) }

func (p *parser) failAt(line int, format string, args ...any) {
	if p.failed[line] {
		return
	}
	p.failed[line] = true
	p.t.errs = append(p.t.errs, problem{line, fmt.Sprintf(format, args...)})
}

// parse reads src, ABC notation, into a tune, with the problems of the
// lines it could not read.
func parse(src string) *tune {
	t := &tune{beat: whole / 4, bpm: 120}
	p := &parser{t: t, accs: map[int]int{}, failed: map[int]bool{}}
	for n, line := range strings.Split(src, "\n") {
		p.line = n + 1
		if i := strings.IndexByte(line, '%'); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) >= 2 && line[1] == ':' && (line[0] >= 'A' && line[0] <= 'Z' || line[0] >= 'a' && line[0] <= 'z') {
			p.field(line[0], strings.TrimSpace(line[2:]))
			continue
		}
		p.music(line)
	}
	if p.left > 0 {
		t.triplets = append(t.triplets, [2]int{p.first, len(t.events) - 1})
	}
	if len(t.events) > p.from {
		t.measures = append(t.measures, measure{from: p.from, to: len(t.events), open: p.open})
	}
	for i := range t.events {
		e := &t.events[i]
		e.ticks = e.written
		if e.triplet {
			e.ticks = e.written * 2 / 3
		}
	}
	slices.SortStableFunc(t.errs, func(a, b problem) int { return cmp.Compare(a.line, b.line) })
	return t
}

func (p *parser) field(name byte, v string) {
	t := p.t
	switch name {
	case 'T':
		if t.title == "" {
			t.title = v
		}
	case 'C':
		if t.composer == "" {
			t.composer = v
		}
	case 'M':
		if p.body {
			p.fail("the meter changes only in the header")
			return
		}
		m, ok := parseMeter(v)
		if !ok {
			p.fail("%q is not a meter: 4/4, 6/8, C, C| or none", v)
			return
		}
		t.meter = m
	case 'L':
		n, d, ok := fraction(v)
		if !ok || whole*n%d != 0 || n <= 0 {
			p.fail("%q is not a note length, as 1/8", v)
			return
		}
		p.unit = whole * n / d
	case 'Q':
		if p.body {
			p.fail("the tempo changes only in the header")
			return
		}
		beat, bpm, ok := parseTempo(v)
		if !ok {
			p.fail("%q is not a tempo, as 1/4=120", v)
			return
		}
		t.beat, t.bpm, t.tempo = beat, bpm, true
	case 'K':
		if p.body {
			p.fail("the key changes only in the header")
			return
		}
		p.key(v)
	}
}

func parseMeter(v string) (meter, bool) {
	switch v {
	case "C":
		return meter{4, 4, 'C'}, true
	case "C|":
		return meter{2, 2, '|'}, true
	case "none", "":
		return meter{}, true
	}
	n, d, ok := fraction(v)
	if !ok || n < 1 || n > 32 || d > 32 || d&(d-1) != 0 {
		return meter{}, false
	}
	return meter{num: n, den: d}, true
}

// fraction reads n/d.
func fraction(v string) (n, d int, ok bool) {
	a, b, found := strings.Cut(strings.TrimSpace(v), "/")
	if !found {
		return 0, 0, false
	}
	n, err1 := strconv.Atoi(strings.TrimSpace(a))
	d, err2 := strconv.Atoi(strings.TrimSpace(b))
	return n, d, err1 == nil && err2 == nil && n > 0 && d > 0
}

// parseTempo reads 1/4=120, or 120 for quarters, skipping text in quotes.
func parseTempo(v string) (beat, bpm int, ok bool) {
	for {
		i := strings.IndexByte(v, '"')
		if i < 0 {
			break
		}
		j := strings.IndexByte(v[i+1:], '"')
		if j < 0 {
			return 0, 0, false
		}
		v = v[:i] + v[i+1+j+1:]
	}
	v = strings.TrimSpace(v)
	beat = whole / 4
	if l, r, found := strings.Cut(v, "="); found {
		n, d, ok := fraction(l)
		if !ok || whole*n%d != 0 {
			return 0, 0, false
		}
		beat, v = whole*n/d, strings.TrimSpace(r)
	}
	bpm, err := strconv.Atoi(v)
	return beat, bpm, err == nil && bpm >= 20 && bpm <= 400
}

var fifths = map[byte]int{'F': -1, 'C': 0, 'G': 1, 'D': 2, 'A': 3, 'E': 4, 'B': 5}

func (p *parser) key(v string) {
	t := p.t
	words := strings.Fields(v)
	tonic := ""
	minor := false
	for i, w := range words {
		lw := strings.ToLower(w)
		switch {
		case lw == "clef=bass" || lw == "bass":
			t.bass = true
		case lw == "clef=treble" || lw == "treble":
			t.bass = false
		case i == 1 && (lw == "m" || lw == "min" || lw == "minor"):
			minor = true
		case i == 1 && (lw == "maj" || lw == "major"):
		case i == 0 && w[0] >= 'A' && w[0] <= 'G':
			tonic = w
		default:
			p.fail("%q is not a major or minor key, or clef=bass", w)
			return
		}
	}
	if tonic == "" {
		return
	}
	k := fifths[tonic[0]]
	rest := tonic[1:]
	switch {
	case strings.HasPrefix(rest, "#"):
		k, rest = k+7, rest[1:]
	case strings.HasPrefix(rest, "b"):
		k, rest = k-7, rest[1:]
	}
	switch strings.ToLower(rest) {
	case "":
	case "m", "min", "minor":
		minor = true
	case "maj", "major":
	default:
		p.fail("%q is not a major or minor key", tonic)
		return
	}
	if minor {
		k -= 3
	}
	if k < -7 || k > 7 {
		p.fail("%s has no key signature", v)
		return
	}
	t.fifths = k
}

// music reads a line of notes.
func (p *parser) music(s string) {
	t := p.t
	if !p.body {
		p.body = true
		if p.unit == 0 {
			p.unit = whole / 8
			if m := t.meter; m.num > 0 && 4*m.num < 3*m.den {
				p.unit = whole / 16
			}
		}
	}
	p.joined = false
	for i := 0; i < len(s); {
		c := s[i]
		col := utf8.RuneCountInString(s[:i]) + 1
		switch {
		case c == ' ' || c == '\t' || c == '\r':
			p.joined = false
			i++
		case c == '`' || c == '\\' || c == 'y':
			i++
		case c == '|' || c == ':' || c == '[' && i+1 < len(s) && s[i+1] == '|':
			i = p.bar(s, i)
		case c == '[' && i+2 < len(s) && s[i+2] == ':':
			p.fail("fields in brackets are not read")
			i = skip(s, i, ']')
		case c == '[':
			i = p.chord(s, i, col)
		case c == '(' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9':
			if s[i+1] != '3' || i+2 < len(s) && s[i+2] == ':' {
				p.fail("only triplets, (3, are read")
			} else {
				p.left, p.first = 3, len(t.events)
			}
			i += 2
		case c == '(' || c == ')':
			p.fail("slurs are not read")
			i++
		case c == '>' || c == '<':
			n := 0
			for i < len(s) && s[i] == c && n < 3 {
				n, i = n+1, i+1
			}
			if len(t.events) == 0 {
				p.fail("%c goes between two notes", c)
				continue
			}
			if c == '<' {
				n = -n
			}
			p.broken = n
		case c == '-':
			p.fail("a tie follows a note")
			i++
		case c == 'z':
			i = p.rest(s, i, col)
		case strings.IndexByte("^_=ABCDEFGabcdefg", c) >= 0:
			h, n, d, j, ok := p.note(s, i)
			i = j
			if !ok {
				continue
			}
			if i < len(s) && s[i] == '-' {
				h.tie = true
				i++
			}
			p.add(event{line: p.line, col: col, heads: []head{h}}, n, d)
		case c == '"':
			p.fail("chord symbols and text in quotes are not read")
			i = skip(s, i, '"')
		case c == '!' || c == '+':
			p.fail("decorations are not read")
			i = skip(s, i, c)
		case c == '{':
			p.fail("grace notes are not read")
			i = skip(s, i, '}')
		case strings.IndexByte(".~HLMOPSTuv", c) >= 0:
			p.fail("decorations are not read")
			i++
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			p.fail("%q is not a note, a rest or a bar", r)
			i += size
		}
	}
}

// skip returns the index after the next end in s after i, or the end of s.
func skip(s string, i int, end byte) int {
	if j := strings.IndexByte(s[i+1:], end); j >= 0 {
		return i + 1 + j + 1
	}
	return len(s)
}

func (p *parser) bar(s string, i int) int {
	for _, b := range bars {
		if !strings.HasPrefix(s[i:], b.text) {
			continue
		}
		i += len(b.text)
		if i < len(s) && s[i] >= '0' && s[i] <= '9' {
			p.fail("endings are not read")
			for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == ',' || s[i] == '-') {
				i++
			}
		}
		p.push(b.kind)
		return i
	}
	p.fail(": belongs to a repeat, :| or |:")
	return i + 1
}

// push closes the measure with b, or adds b to the bar before when the
// measure is empty.
func (p *parser) push(b bar) {
	t := p.t
	clear(p.accs)
	p.joined = false
	switch {
	case len(t.events) > p.from:
		t.measures = append(t.measures, measure{from: p.from, to: len(t.events), open: p.open, close: b})
		p.from, p.open = len(t.events), noBar
	case len(t.measures) > 0:
		last := &t.measures[len(t.measures)-1]
		switch {
		case last.close == thin:
			last.close = b
		case last.close == repEnd && b == repStart:
			last.close = repBoth
		default:
			last.close = b
		}
	default:
		p.open = b
	}
}

// length reads the length after a note: 2, /2, /, //, 3/2 and the like,
// as a fraction.
func length(s string, i int) (n, d, j int) {
	n, d = 1, 1
	j = i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j > i {
		n, _ = strconv.Atoi(s[i:j])
	}
	for j < len(s) && s[j] == '/' {
		j++
		k := j
		for k < len(s) && s[k] >= '0' && s[k] <= '9' {
			k++
		}
		if k > j {
			v, _ := strconv.Atoi(s[j:k])
			d *= max(v, 1)
			j = k
		} else {
			d *= 2
		}
	}
	return n, d, j
}

// note reads a note at s[i]: its head and its length, n/d of the unit.
func (p *parser) note(s string, i int) (h head, n, d, j int, ok bool) {
	start := i
	switch {
	case strings.HasPrefix(s[i:], "^^"):
		h.acc, i = doubleSharp, i+2
	case strings.HasPrefix(s[i:], "__"):
		h.acc, i = doubleFlat, i+2
	case s[i] == '^':
		h.acc, i = sharp, i+1
	case s[i] == '_':
		h.acc, i = flat, i+1
	case s[i] == '=':
		h.acc, i = natural, i+1
	}
	if i >= len(s) || strings.IndexByte("ABCDEFGabcdefg", s[i]) < 0 {
		p.fail("an accidental goes before a note")
		return h, 0, 0, i, false
	}
	letter := strings.IndexByte("CDEFGAB", s[i]&^0x20)
	octave := 4
	if s[i] >= 'a' {
		octave = 5
	}
	i++
	for i < len(s) && (s[i] == ',' || s[i] == '\'') {
		if s[i] == ',' {
			octave--
		} else {
			octave++
		}
		i++
	}
	if octave < 1 || octave > 8 {
		p.fail("%s is out of range", s[start:i])
		return h, 0, 0, i, false
	}
	h.step = 7*octave + letter
	switch alter, seen := p.accs[h.step]; {
	case h.acc != noAcc:
		h.alter = alters[h.acc]
		p.accs[h.step] = h.alter
	case seen:
		h.alter = alter
	default:
		h.alter = p.t.keyAlter(letter)
	}
	n, d, i = length(s, i)
	return h, n, d, i, true
}

func (p *parser) rest(s string, i, col int) int {
	n, d, j := length(s, i+1)
	p.add(event{line: p.line, col: col}, n, d)
	return j
}

func (p *parser) chord(s string, i, col int) int {
	e := event{line: p.line, col: col}
	n, d := 0, 1
	i++
	for i < len(s) && s[i] != ']' {
		if strings.IndexByte("^_=ABCDEFGabcdefg", s[i]) < 0 {
			p.fail("a chord holds notes only")
			return skip(s, i-1, ']')
		}
		h, hn, hd, j, ok := p.note(s, i)
		i = j
		if !ok {
			return skip(s, i-1, ']')
		}
		if i < len(s) && s[i] == '-' {
			h.tie = true
			i++
		}
		if len(e.heads) == 0 {
			n, d = hn, hd
		}
		e.heads = append(e.heads, h)
	}
	if i >= len(s) {
		p.fail("a chord has no ]")
		return i
	}
	if len(e.heads) == 0 {
		p.fail("a chord has no notes")
		return i + 1
	}
	cn, cd, j := length(s, i+1)
	i = j
	if i < len(s) && s[i] == '-' {
		for k := range e.heads {
			e.heads[k].tie = true
		}
		i++
	}
	slices.SortStableFunc(e.heads, func(a, b head) int { return cmp.Compare(a.step, b.step) })
	p.add(e, n*cn, d*cd)
	return i
}

// add adds e, n/d of the unit long, with the broken rhythm and the
// triplet it is in.
func (p *parser) add(e event, n, d int) {
	t := p.t
	if p.unit*n%d != 0 {
		p.fail("a note is shorter than a 64th")
		return
	}
	e.written = p.unit * n / d
	if _, _, ok := value(e.written); !ok {
		g := gcd(e.written, whole)
		p.fail("no one note lasts %d/%d: tie two to make it", e.written/g, whole/g)
		return
	}
	e.joined = p.joined && len(t.events) > 0
	p.joined = true
	if p.broken != 0 && len(t.events) > 0 {
		prev := &t.events[len(t.events)-1]
		k := 1 << abs(p.broken)
		long, short := prev, &e
		if p.broken < 0 {
			long, short = short, long
		}
		l, s := long.written*(2*k-1)/k, short.written/k
		_, _, ok1 := value(l)
		_, _, ok2 := value(s)
		if ok1 && ok2 && long.written*(2*k-1)%k == 0 && short.written%k == 0 {
			long.written, short.written = l, s
		} else {
			p.fail("a broken rhythm here makes no one note")
		}
	}
	p.broken = 0
	if p.tied {
		p.tied = false
		if prev := t.events[len(t.events)-1]; !matches(prev, e) {
			p.failAt(p.tieLine, "a tie joins notes of the same pitch")
		}
	}
	for _, h := range e.heads {
		if h.tie {
			p.tied, p.tieLine = true, p.line
		}
	}
	if p.left > 0 {
		e.triplet = true
		if p.left--; p.left == 0 {
			t.triplets = append(t.triplets, [2]int{p.first, len(t.events)})
		}
	}
	t.events = append(t.events, e)
}

// matches reports whether every tied head of a has a head of b at its
// pitch.
func matches(a, b event) bool {
	for _, h := range a.heads {
		if h.tie && !slices.ContainsFunc(b.heads, func(g head) bool { return g.pitch() == h.pitch() }) {
			return false
		}
	}
	return true
}

func abs(v int) int { return max(v, -v) }

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

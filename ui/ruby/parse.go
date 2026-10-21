package ruby

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Parse reads runs from markup: {東京|とうきょう}, a reading over a word;
// {漢字|かん|じ}, a reading over each character, where one left empty
// leaves its character plain; and 漢(hàn)字(zì), a reading without spaces
// in parentheses after a Chinese character. A brace whose readings are
// neither one nor one for each character puts them together over the
// word. Everything else is plain text, braces and parentheses that hold no
// reading included.
func Parse(markup string) []Run {
	var runs []Run
	var plain strings.Builder
	add := func(r Run) {
		if r.Reading == "" {
			plain.WriteString(r.Base)
			return
		}
		if plain.Len() > 0 {
			runs = append(runs, Run{Base: plain.String()})
			plain.Reset()
		}
		runs = append(runs, r)
	}
	for s := markup; s != ""; {
		switch s[0] {
		case '{':
			if group, n := braced(s); n > 0 {
				for _, r := range group {
					add(r)
				}
				s = s[n:]
				continue
			}
		case '(':
			last, size := utf8.DecodeLastRuneInString(plain.String())
			if reading, n := parenthesized(s); n > 0 && unicode.Is(unicode.Han, last) {
				base := plain.String()
				plain.Reset()
				plain.WriteString(base[:len(base)-size])
				add(Run{Base: base[len(base)-size:], Reading: reading})
				s = s[n:]
				continue
			}
		}
		_, n := utf8.DecodeRuneInString(s)
		plain.WriteString(s[:n])
		s = s[n:]
	}
	if plain.Len() > 0 {
		runs = append(runs, Run{Base: plain.String()})
	}
	return runs
}

// braced reads {base|reading…} at the start of s: the runs it holds and
// how long it is, or 0 when it is not one.
func braced(s string) ([]Run, int) {
	end := strings.IndexAny(s, "}\n")
	if end < 0 || s[end] != '}' {
		return nil, 0
	}
	parts := strings.Split(s[1:end], "|")
	base, readings := parts[0], parts[1:]
	if base == "" || len(readings) == 0 {
		return nil, 0
	}
	if len(readings) > 1 && len(readings) == utf8.RuneCountInString(base) {
		runs := make([]Run, 0, len(readings))
		for i, r := range []rune(base) {
			runs = append(runs, Run{Base: string(r), Reading: readings[i]})
		}
		return runs, end + 1
	}
	return []Run{{Base: base, Reading: strings.Join(readings, "")}}, end + 1
}

// parenthesized reads (reading) at the start of s: the reading and how
// long it is with its parentheses, or 0 when it is not one.
func parenthesized(s string) (string, int) {
	end := strings.IndexByte(s, ')')
	if end < 2 || strings.ContainsFunc(s[1:end], func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("(){}|", r)
	}) {
		return "", 0
	}
	return s[1:end], end + 1
}

// text returns runes from up to to of the paragraph s.runs make, in which
// a run with a reading is one rune: its base, followed, with readings, by
// its reading in parentheses.
func (s *State) text(from, to int, readings bool) string {
	var b strings.Builder
	for i, r := range s.runs {
		a, z := s.at[i], s.at[i+1]
		if z <= from || a >= to {
			continue
		}
		if r.Reading == "" {
			b.WriteString(runes(r.Base, max(from, a)-a, min(to, z)-a))
			continue
		}
		b.WriteString(r.Base)
		if readings {
			b.WriteString("(" + r.Reading + ")")
		}
	}
	return b.String()
}

// runes returns runes from up to to of s.
func runes(s string, from, to int) string {
	start, n := len(s), 0
	for i := range s {
		if n == from {
			start = i
		}
		if n == to {
			return s[start:i]
		}
		n++
	}
	return s[start:]
}

// wide reports whether s is all kana, zhuyin or Chinese characters, which
// spread apart over a wider base.
func wide(s string) bool {
	for _, r := range s {
		if !unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Bopomofo) && !strings.ContainsRune("ーˊˇˋ˙", r) {
			return false
		}
	}
	return true
}

// language returns the language runs read in, for the forms of their
// Chinese characters: Japanese with kana in them, traditional Chinese with
// zhuyin, simplified Chinese with other Chinese characters, else "", the
// app's.
func language(runs []Run) string {
	lang := ""
	for _, r := range runs {
		for _, c := range r.Base + r.Reading {
			switch {
			case unicode.In(c, unicode.Hiragana, unicode.Katakana):
				return "ja"
			case unicode.Is(unicode.Bopomofo, c):
				lang = "zh-Hant"
			case lang == "" && unicode.Is(unicode.Han, c):
				lang = "zh-Hans"
			}
		}
	}
	return lang
}

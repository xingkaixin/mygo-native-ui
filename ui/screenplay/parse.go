package screenplay

import (
	"strings"
	"unicode"
)

// Kind is what a line of a screenplay is.
type Kind int

const (
	Action Kind = iota
	Heading
	Character
	Parenthetical
	Dialogue
	Transition
	Centered
)

// Line is a paragraph of a screenplay: an action, a scene heading, a
// character's cue and their lines, a transition, or centered text.
type Line struct {
	Kind Kind
	Text string
	// Scene numbers the scene a heading starts, from 1.
	Scene int
}

// Script is a screenplay: its title page and its lines.
type Script struct {
	Title, Credit, Author string
	Lines                 []Line
}

// Parse reads a screenplay written in Fountain: a title page of Key:
// value lines, then scene headings starting INT., EXT. or a period,
// character cues in capitals followed by their dialogue and
// parentheticals, transitions in capitals ending TO: or starting >, text
// centered between > and <, and action, which is everything else or
// starts with !. Notes in [[ ]], boneyards in /* */, sections starting #
// and synopses starting = are left out.
func Parse(src string) Script {
	var sc Script
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strip(src, "/*", "*/")
	src = strip(src, "[[", "]]")
	lines := strings.Split(src, "\n")
	i := 0
	if k, _, ok := strings.Cut(lines[0], ":"); ok && (!strings.Contains(k, " ") || strings.EqualFold(k, "draft date")) {
		for ; i < len(lines) && strings.TrimSpace(lines[i]) != ""; i++ {
			k, v, ok := strings.Cut(lines[i], ":")
			if !ok {
				continue
			}
			v = strings.TrimSpace(v)
			switch strings.ToLower(strings.TrimSpace(k)) {
			case "title":
				sc.Title = v
			case "credit":
				sc.Credit = v
			case "author", "authors":
				sc.Author = v
			}
		}
	}
	blank := func(j int) bool { return j < 0 || j >= len(lines) || strings.TrimSpace(lines[j]) == "" }
	// action adds a line of action, to the paragraph before when no blank
	// line parts them.
	action := func(at int, text string) {
		if n := len(sc.Lines); n > 0 && !blank(at-1) && sc.Lines[n-1].Kind == Action {
			sc.Lines[n-1].Text += "\n" + text
			return
		}
		sc.Lines = append(sc.Lines, Line{Kind: Action, Text: text})
	}
	scene := 0
	inDialogue := false
	for ; i < len(lines); i++ {
		raw := lines[i]
		text := strings.TrimSpace(raw)
		if text == "" {
			inDialogue = false
			continue
		}
		if inDialogue {
			kind := Dialogue
			if strings.HasPrefix(text, "(") && strings.HasSuffix(text, ")") {
				kind = Parenthetical
			}
			if n := len(sc.Lines); kind == Dialogue && sc.Lines[n-1].Kind == Dialogue {
				sc.Lines[n-1].Text += "\n" + text
			} else {
				sc.Lines = append(sc.Lines, Line{Kind: kind, Text: text})
			}
			continue
		}
		switch {
		case strings.HasPrefix(text, "#"), strings.HasPrefix(text, "="):
			continue
		case strings.HasPrefix(text, "!"):
			action(i, text[1:])
		case strings.HasPrefix(text, ">") && strings.HasSuffix(text, "<"):
			sc.Lines = append(sc.Lines, Line{Kind: Centered, Text: strings.TrimSpace(text[1 : len(text)-1])})
		case strings.HasPrefix(text, ">"):
			sc.Lines = append(sc.Lines, Line{Kind: Transition, Text: strings.TrimSpace(text[1:])})
		case blank(i-1) && (strings.HasPrefix(text, ".") && !strings.HasPrefix(text, "..") || isHeading(text)):
			scene++
			sc.Lines = append(sc.Lines, Line{Kind: Heading, Text: strings.ToUpper(strings.TrimPrefix(text, ".")), Scene: scene})
		case blank(i-1) && blank(i+1) && isUpper(text) && strings.HasSuffix(text, "TO:"):
			sc.Lines = append(sc.Lines, Line{Kind: Transition, Text: text})
		case blank(i-1) && !blank(i+1) && (strings.HasPrefix(text, "@") || isUpper(text)):
			sc.Lines = append(sc.Lines, Line{Kind: Character, Text: strings.TrimPrefix(text, "@")})
			inDialogue = true
		default:
			action(i, strings.TrimRight(raw, " "))
		}
	}
	return sc
}

func isHeading(s string) bool {
	u := strings.ToUpper(s)
	for _, p := range []string{"INT.", "EXT.", "INT/EXT", "INT./EXT.", "I/E", "EST."} {
		if strings.HasPrefix(u, p+" ") || u == p {
			return true
		}
	}
	return false
}

// isUpper reports whether s has letters and none in lower case, leaving
// out an extension in parentheses such as (V.O.) or (cont'd).
func isUpper(s string) bool {
	if i := strings.Index(s, "("); i > 0 {
		s = s[:i]
	}
	letters := false
	for _, r := range s {
		if unicode.IsLower(r) {
			return false
		}
		letters = letters || unicode.IsLetter(r)
	}
	return letters
}

func strip(s, open, close string) string {
	for {
		i := strings.Index(s, open)
		if i < 0 {
			return s
		}
		j := strings.Index(s[i+len(open):], close)
		if j < 0 {
			return s[:i]
		}
		s = s[:i] + s[i+len(open)+j+len(close):]
	}
}

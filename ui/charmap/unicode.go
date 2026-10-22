package charmap

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/runenames"
)

// block is a block of Unicode the viewer shows: the section and the group
// of the sidebar it shows in, its name, and its first and last code points.
type block struct {
	section, group, name string
	first, last          rune
}

// blocks are the blocks the viewer shows, in the order of the sidebar,
// the blocks of a group one after another. The standard library has no
// table of blocks; these are of Unicode's Blocks.txt.
var blocks = []block{
	{"Scripts", "Basic Latin", "Basic Latin", 0x0000, 0x007F},
	{"Scripts", "Latin-1 Supplement", "Latin-1 Supplement", 0x0080, 0x00FF},
	{"Scripts", "Latin Extended", "Latin Extended-A", 0x0100, 0x017F},
	{"Scripts", "Latin Extended", "Latin Extended-B", 0x0180, 0x024F},
	{"Scripts", "Latin Extended", "Latin Extended Additional", 0x1E00, 0x1EFF},
	{"Scripts", "Greek", "Greek and Coptic", 0x0370, 0x03FF},
	{"Scripts", "Greek", "Greek Extended", 0x1F00, 0x1FFF},
	{"Scripts", "Cyrillic", "Cyrillic", 0x0400, 0x04FF},
	{"Scripts", "Cyrillic", "Cyrillic Supplement", 0x0500, 0x052F},
	{"Scripts", "Hebrew", "Hebrew", 0x0590, 0x05FF},
	{"Scripts", "Arabic", "Arabic", 0x0600, 0x06FF},
	{"Scripts", "Devanagari", "Devanagari", 0x0900, 0x097F},
	{"Scripts", "Thai", "Thai", 0x0E00, 0x0E7F},
	{"Scripts", "Hiragana", "Hiragana", 0x3040, 0x309F},
	{"Scripts", "Katakana", "Katakana", 0x30A0, 0x30FF},
	{"Scripts", "CJK Unified Ideographs", "CJK Unified Ideographs", 0x4E00, 0x9FFF},
	{"Scripts", "Hangul", "Hangul Jamo", 0x1100, 0x11FF},
	{"Scripts", "Hangul", "Hangul Compatibility Jamo", 0x3130, 0x318F},
	{"Scripts", "Hangul", "Hangul Syllables", 0xAC00, 0xD7AF},
	{"Symbols", "General Punctuation", "General Punctuation", 0x2000, 0x206F},
	{"Symbols", "CJK Symbols and Punctuation", "CJK Symbols and Punctuation", 0x3000, 0x303F},
	{"Symbols", "Superscripts and Subscripts", "Superscripts and Subscripts", 0x2070, 0x209F},
	{"Symbols", "Currency Symbols", "Currency Symbols", 0x20A0, 0x20CF},
	{"Symbols", "Letterlike Symbols", "Letterlike Symbols", 0x2100, 0x214F},
	{"Symbols", "Number Forms", "Number Forms", 0x2150, 0x218F},
	{"Symbols", "Arrows", "Arrows", 0x2190, 0x21FF},
	{"Symbols", "Arrows", "Supplemental Arrows-A", 0x27F0, 0x27FF},
	{"Symbols", "Arrows", "Supplemental Arrows-B", 0x2900, 0x297F},
	{"Symbols", "Mathematical Operators", "Mathematical Operators", 0x2200, 0x22FF},
	{"Symbols", "Mathematical Operators", "Supplemental Mathematical Operators", 0x2A00, 0x2AFF},
	{"Symbols", "Miscellaneous Technical", "Miscellaneous Technical", 0x2300, 0x23FF},
	{"Symbols", "Enclosed Alphanumerics", "Enclosed Alphanumerics", 0x2460, 0x24FF},
	{"Symbols", "Box Drawing", "Box Drawing", 0x2500, 0x257F},
	{"Symbols", "Block Elements", "Block Elements", 0x2580, 0x259F},
	{"Symbols", "Geometric Shapes", "Geometric Shapes", 0x25A0, 0x25FF},
	{"Symbols", "Miscellaneous Symbols", "Miscellaneous Symbols", 0x2600, 0x26FF},
	{"Symbols", "Dingbats", "Dingbats", 0x2700, 0x27BF},
	{"Symbols", "Braille", "Braille Patterns", 0x2800, 0x28FF},
}

// group is a group of the sidebar: its section, its name, and the
// characters of its blocks that show, unassigned code points and those
// drawing nothing, as controls, left out.
type group struct {
	section, name string
	chars         []rune
}

var load = sync.OnceValue(func() []group {
	var groups []group
	for _, b := range blocks {
		if len(groups) == 0 || groups[len(groups)-1].name != b.group {
			groups = append(groups, group{section: b.section, name: b.group})
		}
		g := &groups[len(groups)-1]
		for r := b.first; r <= b.last; r++ {
			if unicode.IsGraphic(r) {
				g.chars = append(g.chars, r)
			}
		}
	}
	return groups
})

// blockOf returns the name of the block of r, "" for one the viewer does
// not show.
func blockOf(r rune) string {
	for _, b := range blocks {
		if r >= b.first && r <= b.last {
			return b.name
		}
	}
	return ""
}

// The parts of the names of Hangul syllables: the leading consonant, the
// vowel and the trailing consonant.
var (
	leads  = strings.Split("G GG N D DD R M B BB S SS  J JJ C K T P H", " ")
	vowels = strings.Split("A AE YA YAE EO E YEO YE O WA WAE OE YO U WEO WE WI YU EU YI I", " ")
	tails  = strings.Split(" G GG GS N NJ NH D L LG LM LB LS LT LP LH M B BS S SS NG J C K T P H", " ")
)

// name returns the name of r, as Unicode derives it for CJK ideographs and
// Hangul syllables, whose names the tables leave out.
func name(r rune) string {
	n := runenames.Name(r)
	switch {
	case strings.HasPrefix(n, "<CJK Ideograph"):
		return fmt.Sprintf("CJK UNIFIED IDEOGRAPH-%04X", r)
	case n == "<Hangul Syllable>":
		s := r - 0xAC00
		return "HANGUL SYLLABLE " + leads[s/588] + vowels[s%588/28] + tails[s%28]
	}
	return n
}

// categories are the general categories of the characters that show.
var categories = []struct{ code, name string }{
	{"Lu", "Uppercase Letter"}, {"Ll", "Lowercase Letter"}, {"Lt", "Titlecase Letter"},
	{"Lm", "Modifier Letter"}, {"Lo", "Other Letter"},
	{"Mn", "Nonspacing Mark"}, {"Mc", "Spacing Mark"}, {"Me", "Enclosing Mark"},
	{"Nd", "Decimal Number"}, {"Nl", "Letter Number"}, {"No", "Other Number"},
	{"Pc", "Connector Punctuation"}, {"Pd", "Dash Punctuation"}, {"Ps", "Open Punctuation"},
	{"Pe", "Close Punctuation"}, {"Pi", "Initial Punctuation"}, {"Pf", "Final Punctuation"},
	{"Po", "Other Punctuation"},
	{"Sm", "Math Symbol"}, {"Sc", "Currency Symbol"}, {"Sk", "Modifier Symbol"}, {"So", "Other Symbol"},
	{"Zs", "Space Separator"},
}

// category returns the general category of r, as "Other Symbol (So)".
func category(r rune) string {
	for _, c := range categories {
		if unicode.Is(unicode.Categories[c.code], r) {
			return c.name + " (" + c.code + ")"
		}
	}
	return ""
}

// glyph returns the text showing r: on a dotted circle when it is a mark,
// which draws over the character before it.
func glyph(r rune) string {
	if unicode.Is(unicode.M, r) {
		return "◌" + string(r)
	}
	return string(r)
}

// entry is a character and its name, lowercase, for searching.
type entry struct {
	r    rune
	name string
}

// index is the names of the characters of the groups, made once in the
// background: names holds them once built is closed.
var index = struct {
	once  sync.Once
	built chan struct{}
	names []entry
}{built: make(chan struct{})}

// nameIndex returns the index of names, or false while it is made: the
// first call starts making it.
func nameIndex() ([]entry, bool) {
	index.once.Do(func() {
		go func() {
			var names []entry
			for _, g := range load() {
				for _, r := range g.chars {
					names = append(names, entry{r, strings.ToLower(name(r))})
				}
			}
			index.names = names
			close(index.built)
		}()
	})
	select {
	case <-index.built:
		return index.names, true
	default:
		return nil, false
	}
}

// search returns the characters query finds: the character it is, or
// whose code point it is, then those of names with a word starting with
// each of its words, the name that is query first, then those starting
// with it.
func search(query string, names []entry) []rune {
	var found, exact, starts, others []rune
	at, ok := lookup(query)
	if ok {
		found = append(found, at)
	}
	words := strings.Fields(strings.ToLower(query))
	q := strings.Join(words, " ")
	for _, e := range names {
		switch {
		case ok && e.r == at, !matches(e.name, words):
		case e.name == q:
			exact = append(exact, e.r)
		case strings.HasPrefix(e.name, q):
			starts = append(starts, e.r)
		default:
			others = append(others, e.r)
		}
	}
	return slices.Concat(found, exact, starts, others)
}

// lookup returns the character query is, as ✓, or whose code point it
// is, as U+2713, or 2713 without U+: four hex digits or more, a decimal
// digit among them.
func lookup(query string) (rune, bool) {
	q := strings.TrimSpace(query)
	if r, size := utf8.DecodeRuneInString(q); size > 0 && size == len(q) && unicode.IsGraphic(r) {
		return r, true
	}
	hex, plus := strings.CutPrefix(strings.ToUpper(q), "U+")
	// Words such as face and cafe are hex too, but searched for by name.
	if !plus && (len(hex) < 4 || !strings.ContainsAny(hex, "0123456789")) {
		return 0, false
	}
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil || n > unicode.MaxRune || !unicode.IsGraphic(rune(n)) {
		return 0, false
	}
	return rune(n), true
}

// matches reports whether each of words starts a word of name.
func matches(name string, words []string) bool {
	for _, w := range words {
		if !strings.HasPrefix(name, w) && !strings.Contains(name, " "+w) && !strings.Contains(name, "-"+w) {
			return false
		}
	}
	return len(words) > 0
}

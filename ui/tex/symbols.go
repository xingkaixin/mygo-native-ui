package tex

import (
	"strings"
	"unicode/utf8"
)

// class is what an atom is to TeX, which sets the room around it.
type class uint8

const (
	ord class = iota
	op
	bin
	rel
	open
	close
	punct
	inner
)

// symbol is a command that stands for a character.
type symbol struct {
	r     rune
	class class
}

var (
	symbols = map[string]symbol{}
	// classes are the classes of characters typed as they are.
	classes = map[rune]class{
		'+': bin, '−': bin, '∗': bin, '=': rel, '<': rel, '>': rel, ':': rel, '|': ord, '‖': ord,
		'(': open, '[': open, ')': close, ']': close, '!': close, '?': close,
		',': punct, ';': punct,
	}
	delimiters = map[string]rune{
		"(": '(', ")": ')', "[": '[', "]": ']', "|": '|', "/": '/', "<": '⟨', ">": '⟩',
	}
)

func init() {
	for cl, list := range [...]string{
		ord: `alpha α beta β gamma γ delta δ epsilon ϵ varepsilon ε zeta ζ eta η theta θ
			vartheta ϑ iota ι kappa κ varkappa ϰ lambda λ mu μ nu ν xi ξ omicron ο pi π
			varpi ϖ rho ρ varrho ϱ sigma σ varsigma ς tau τ upsilon υ phi ϕ varphi φ
			chi χ psi ψ omega ω Gamma Γ Delta Δ Theta Θ Lambda Λ Xi Ξ Pi Π Sigma Σ
			Upsilon Υ Phi Φ Psi Ψ Omega Ω
			infty ∞ partial ∂ nabla ∇ emptyset ∅ varnothing ∅ forall ∀ exists ∃ nexists ∄
			neg ¬ lnot ¬ prime ′ hbar ℏ hslash ℏ ell ℓ Re ℜ Im ℑ wp ℘ aleph ℵ beth ℶ
			angle ∠ measuredangle ∡ triangle △ top ⊤ bot ⊥ ldots … dots … cdots ⋯ vdots ⋮
			ddots ⋱ clubsuit ♣ diamondsuit ♢ heartsuit ♡ spadesuit ♠ flat ♭ natural ♮
			sharp ♯ backslash \ vert | Vert ‖ | ‖ # # $ $ % % & & _ _ imath ı jmath ȷ
			complement ∁ degree ° checkmark ✓ square □ blacksquare ■ lozenge ◊ surd √`,
		op: `sum ∑ prod ∏ coprod ∐ int ∫ iint ∬ iiint ∭ oint ∮ oiint ∯ oiiint ∰ bigcup ⋃
			bigcap ⋂ bigvee ⋁ bigwedge ⋀ bigoplus ⨁ bigotimes ⨂ bigodot ⨀ biguplus ⨄
			bigsqcup ⨆`,
		bin: `pm ± mp ∓ times × div ÷ cdot ⋅ ast ∗ star ⋆ circ ∘ bullet ∙ oplus ⊕
			ominus ⊖ otimes ⊗ oslash ⊘ odot ⊙ cup ∪ cap ∩ sqcup ⊔ sqcap ⊓ vee ∨ lor ∨
			wedge ∧ land ∧ setminus ∖ smallsetminus ∖ wr ≀ diamond ⋄ bigtriangleup △
			bigtriangledown ▽ triangleleft ◁ triangleright ▷ uplus ⊎ amalg ⨿ dagger †
			ddagger ‡ ltimes ⋉ rtimes ⋊ boxplus ⊞ boxtimes ⊠ cdotp ⋅`,
		rel: `leq ≤ le ≤ geq ≥ ge ≥ neq ≠ ne ≠ approx ≈ equiv ≡ sim ∼ simeq ≃ cong ≅
			propto ∝ ll ≪ gg ≫ prec ≺ succ ≻ preceq ⪯ succeq ⪰ subset ⊂ supset ⊃
			subseteq ⊆ supseteq ⊇ subsetneq ⊊ supsetneq ⊋ in ∈ ni ∋ notin ∉ perp ⊥
			parallel ∥ mid ∣ nmid ∤ models ⊨ vdash ⊢ dashv ⊣ doteq ≐ asymp ≍ bowtie ⋈
			smile ⌣ frown ⌢ leqslant ⩽ geqslant ⩾ lesssim ≲ gtrsim ≳ approxeq ≊
			coloneqq ≔ triangleq ≜ sqsubseteq ⊑ sqsupseteq ⊒ to → rightarrow →
			leftarrow ← gets ← leftrightarrow ↔ Rightarrow ⇒ Leftarrow ⇐
			Leftrightarrow ⇔ iff ⟺ implies ⟹ impliedby ⟸ mapsto ↦ longmapsto ⟼
			longrightarrow ⟶ longleftarrow ⟵ longleftrightarrow ⟷ Longrightarrow ⟹
			Longleftarrow ⟸ Longleftrightarrow ⟺ uparrow ↑ downarrow ↓ updownarrow ↕
			Uparrow ⇑ Downarrow ⇓ Updownarrow ⇕ nearrow ↗ searrow ↘ swarrow ↙ nwarrow ↖
			hookrightarrow ↪ hookleftarrow ↩ rightharpoonup ⇀ leftharpoonup ↼
			rightleftharpoons ⇌ leadsto ⇝ nleq ≰ ngeq ≱ nsim ≁
			ncong ≇ nsubseteq ⊈ nsupseteq ⊉`,
		open:  `{ { lbrace { langle ⟨ lfloor ⌊ lceil ⌈ lvert | lVert ‖ lbrack [`,
		close: `} } rbrace } rangle ⟩ rfloor ⌋ rceil ⌉ rvert | rVert ‖ rbrack ]`,
		punct: `colon :`,
	} {
		f := strings.Fields(list)
		for i := 0; i+1 < len(f); i += 2 {
			r, _ := utf8.DecodeRuneInString(f[i+1])
			symbols[f[i]] = symbol{r, class(cl)}
			if _, ok := classes[r]; !ok && class(cl) != ord {
				classes[r] = class(cl)
			}
			if strings.ContainsRune("()[]{}|‖/\\⟨⟩⌊⌋⌈⌉↑↓↕⇑⇓⇕", r) {
				delimiters[`\`+f[i]] = r
			}
		}
	}
}

// negations are the characters \not makes of others.
var negations = map[rune]rune{
	'=': '≠', '<': '≮', '>': '≯', '∈': '∉', '∋': '∌', '≤': '≰', '≥': '≱', '≡': '≢',
	'∼': '≁', '≃': '≄', '≈': '≉', '≅': '≇', '⊂': '⊄', '⊃': '⊅', '⊆': '⊈', '⊇': '⊉',
	'∣': '∤', '∥': '∦', '∃': '∄', '≺': '⊀', '≻': '⊁', '⊢': '⊬', '⊨': '⊭',
}

// integrals take their scripts beside them, as TeX's \int does.
const integrals = "∫∬∭∮∯∰∱∲∳⨌"

// functions are the names set upright as operators, true for those
// taking limits above and below in display style.
var functions = map[string]bool{
	"arcsin": false, "arccos": false, "arctan": false, "arg": false, "cos": false,
	"cosh": false, "cot": false, "coth": false, "csc": false, "deg": false, "det": true,
	"dim": false, "exp": false, "gcd": true, "hom": false, "inf": true, "ker": false,
	"lg": false, "lim": true, "liminf": true, "limsup": true, "ln": false, "log": false,
	"max": true, "min": true, "Pr": true, "sec": false, "sin": false, "sinh": false,
	"sup": true, "tan": false, "tanh": false,
}

// accents are the accents over what follows, true for those as wide as it.
var accents = map[string]struct {
	r    rune
	wide bool
}{
	"hat": {0x302, false}, "widehat": {0x302, true}, "check": {0x30C, false},
	"widecheck": {0x30C, true}, "tilde": {0x303, false}, "widetilde": {0x303, true},
	"acute": {0x301, false}, "grave": {0x300, false}, "dot": {0x307, false},
	"ddot": {0x308, false}, "dddot": {0x20DB, false}, "breve": {0x306, false},
	"bar": {0x304, false}, "vec": {0x20D7, false}, "mathring": {0x30A, false},
	"overrightarrow": {0x20D7, true}, "overleftarrow": {0x20D6, true},
	"overleftrightarrow": {0x20E1, true},
}

// spaces are the widths of spacing commands, in mu, an 18th of an em.
var spaces = map[string]float32{
	`\,`: 3, `\:`: 4, `\>`: 4, `\;`: 5, `\!`: -3, `\ `: 6, "~": 6, `\quad`: 18, `\qquad`: 36,
	`\enspace`: 9, `\thinspace`: 3, `\medspace`: 4, `\thickspace`: 5, `\negthinspace`: -3,
	`\negmedspace`: -4, `\negthickspace`: -5,
}

// alphabet is a style of letters, as \mathbf's.
type alphabet uint8

const (
	// normal is math's: italic letters, but for capital Greek ones, and
	// upright digits.
	normal alphabet = iota
	roman
	bold
	italic
	boldItalic
	script
	fraktur
	doubleStruck
	sans
	mono
)

var alphabets = map[string]alphabet{
	"mathrm": roman, "mathbf": bold, "mathit": italic, "mathbb": doubleStruck,
	"mathcal": script, "mathscr": script, "mathfrak": fraktur, "mathsf": sans,
	"mathtt": mono, "boldsymbol": boldItalic, "bm": boldItalic, "mathnormal": normal,
}

var textAlphabets = map[string]alphabet{
	"text": roman, "textrm": roman, "mbox": roman, "textnormal": roman, "textup": roman,
	"textbf": bold, "textit": italic, "textsf": sans, "texttt": mono,
}

// starts are where each alphabet starts in Unicode's Mathematical
// Alphanumeric Symbols: its capital A, its small a, its digit 0 and its
// capital Alpha, 0 where it has none.
var starts = [...][4]rune{
	bold:         {0x1D400, 0x1D41A, 0x1D7CE, 0x1D6A8},
	italic:       {0x1D434, 0x1D44E, 0, 0x1D6E2},
	boldItalic:   {0x1D468, 0x1D482, 0, 0x1D71C},
	script:       {0x1D49C, 0x1D4B6, 0, 0},
	fraktur:      {0x1D504, 0x1D51E, 0, 0},
	doubleStruck: {0x1D538, 0x1D552, 0x1D7D8, 0},
	sans:         {0x1D5A0, 0x1D5BA, 0x1D7E2, 0},
	mono:         {0x1D670, 0x1D68A, 0x1D7F6, 0},
}

// holes are the letters of those alphabets that Unicode had before them,
// elsewhere.
var holes = map[rune]rune{
	0x1D455: 'ℎ', 0x1D49D: 'ℬ', 0x1D4A0: 'ℰ', 0x1D4A1: 'ℱ', 0x1D4A3: 'ℋ', 0x1D4A4: 'ℐ',
	0x1D4A7: 'ℒ', 0x1D4A8: 'ℳ', 0x1D4AD: 'ℛ', 0x1D4BA: 'ℯ', 0x1D4BC: 'ℊ', 0x1D4C4: 'ℴ',
	0x1D506: 'ℭ', 0x1D50B: 'ℌ', 0x1D50C: 'ℑ', 0x1D515: 'ℜ', 0x1D51D: 'ℨ', 0x1D53A: 'ℂ',
	0x1D53F: 'ℍ', 0x1D545: 'ℕ', 0x1D547: 'ℙ', 0x1D548: 'ℚ', 0x1D549: 'ℝ', 0x1D551: 'ℤ',
}

// greekVariants are the Greek letters after ω in the math alphabets.
const greekVariants = "ϵϑϰϕϱϖ"

// styled returns r in alphabet a: italic for a letter in math's, Unicode's
// math alphanumerics for the others, and r itself where a has none.
func styled(r rune, a alphabet) rune {
	if a == normal {
		if r < 'A' || r > 'Z' && r < 'a' || r > 'z' && r < 'α' || r > 'ω' && !strings.ContainsRune(greekVariants, r) {
			return r
		}
		a = italic
	}
	if a == roman || int(a) >= len(starts) {
		return r
	}
	s := starts[a]
	var out rune
	switch {
	case r >= 'A' && r <= 'Z':
		out = s[0] + r - 'A'
	case r >= 'a' && r <= 'z':
		out = s[1] + r - 'a'
	case r >= '0' && r <= '9' && s[2] != 0:
		out = s[2] + r - '0'
	case s[3] == 0:
		return r
	case r >= 'Α' && r <= 'Ω' && r != 0x3A2:
		out = s[3] + r - 'Α'
	case r >= 'α' && r <= 'ω':
		out = s[3] + 26 + r - 'α'
	case strings.ContainsRune(greekVariants, r):
		out = s[3] + 52 + rune(strings.IndexRune(greekVariants, r)/2)
	default:
		return r
	}
	if h, ok := holes[out]; ok {
		return h
	}
	return out
}

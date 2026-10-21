package tex

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// fontFiles are where systems keep fonts with a MATH table, the first
// found taken.
var fontFiles = []string{
	"/System/Library/Fonts/Supplemental/STIXTwoMath.otf",
	"/Library/Fonts/STIXTwoMath-Regular.otf",
	"C:/Windows/Fonts/cambria.ttc",
	"/usr/share/fonts/*/STIXTwoMath*.otf",
	"/usr/share/fonts/*/*/STIXTwoMath*.otf",
	"/usr/share/fonts/*/*/latinmodern-math.otf",
	"/usr/share/texmf/fonts/opentype/public/lm-math/latinmodern-math.otf",
	"/usr/share/fonts/*/*/texgyrepagella-math.otf",
	"/usr/share/fonts/*/*/DejaVuMathTeXGyre.ttf",
	"/usr/share/fonts/*/*/NotoSansMath-Regular.ttf",
}

// The values of a MATH table's constants, in its order.
const (
	scriptPercentScaleDown = iota
	scriptScriptPercentScaleDown
	delimitedSubFormulaMinHeight
	displayOperatorMinHeight
	mathLeading
	axisHeight
	accentBaseHeight
	flattenedAccentBaseHeight
	subscriptShiftDown
	subscriptTopMax
	subscriptBaselineDropMin
	superscriptShiftUp
	superscriptShiftUpCramped
	superscriptBottomMin
	superscriptBaselineDropMax
	subSuperscriptGapMin
	superscriptBottomMaxWithSubscript
	spaceAfterScript
	upperLimitGapMin
	upperLimitBaselineRiseMin
	lowerLimitGapMin
	lowerLimitBaselineDropMin
	stackTopShiftUp
	stackTopDisplayStyleShiftUp
	stackBottomShiftDown
	stackBottomDisplayStyleShiftDown
	stackGapMin
	stackDisplayStyleGapMin
	stretchStackTopShiftUp
	stretchStackBottomShiftDown
	stretchStackGapAboveMin
	stretchStackGapBelowMin
	fractionNumeratorShiftUp
	fractionNumeratorDisplayStyleShiftUp
	fractionDenominatorShiftDown
	fractionDenominatorDisplayStyleShiftDown
	fractionNumeratorGapMin
	fractionNumDisplayStyleGapMin
	fractionRuleThickness
	fractionDenominatorGapMin
	fractionDenomDisplayStyleGapMin
	skewedFractionHorizontalGap
	skewedFractionVerticalGap
	overbarVerticalGap
	overbarRuleThickness
	overbarExtraAscender
	underbarVerticalGap
	underbarRuleThickness
	underbarExtraDescender
	radicalVerticalGap
	radicalDisplayStyleVerticalGap
	radicalRuleThickness
	radicalExtraAscender
	radicalKernBeforeDegree
	radicalKernAfterDegree
	radicalDegreeBottomRaisePercent
	numConstants
)

// defaults are the constants, in ems, without a math font: STIX Two
// Math's. The percentages are fractions.
var defaults = [numConstants]float32{
	0.7, 0.55, 1.325, 1.8, 0.15, 0.258, 0.48, 0.656, 0.21, 0.368, 0.16, 0.36, 0.252, 0.12,
	0.23, 0.15, 0.38, 0.04, 0.135, 0.3, 0.135, 0.67, 0.47, 0.78, 0.385, 0.69, 0.15, 0.3, 0.8,
	0.59, 0.068, 0.068, 0.585, 0.64, 0.585, 0.64, 0.068, 0.15, 0.068, 0.068, 0.15, 0.35, 0.068,
	0.175, 0.068, 0.068, 0.175, 0.068, 0.068, 0.085, 0.17, 0.068, 0.078, 0.065, -0.335, 0.55,
}

// outline is a glyph of the math font: its curves, its advance and the
// box of its ink, in ems, y down.
type outline struct {
	ops            []sfnt.SegmentOp
	pts            [][2]float32
	adv            float32
	x0, y0, x1, y1 float32
	// italic is its italic correction, accent where an accent above it
	// centers.
	italic, accent float32
}

// part is a piece of a glyph assembly: its glyph, how long it is in ems,
// and whether it repeats to make the assembly longer.
type part struct {
	g        *outline
	full     float32
	extender bool
}

// construction is how a glyph grows: its larger variants by the size
// they reach, and the parts that build it larger still.
type construction struct {
	variants []*outline
	sizes    []float32
	parts    []part
}

// mathFont is the font math is set in, all its lengths in ems.
type mathFont struct {
	mu          sync.Mutex
	f           *sfnt.Font
	buf         sfnt.Buffer
	upem        float32
	k           [numConstants]float32
	italics     map[sfnt.GlyphIndex]float32
	accents     map[sfnt.GlyphIndex]float32
	vert, horiz map[sfnt.GlyphIndex]int
	cons        []byte
	glyphs      map[sfnt.GlyphIndex]*outline
}

var (
	loadOnce sync.Once
	face     *mathFont
)

// mathFace returns the system's math font, nil when it has none.
func mathFace() *mathFont {
	loadOnce.Do(func() {
		for _, pattern := range fontFiles {
			files, _ := filepath.Glob(pattern)
			for _, name := range files {
				if data, err := os.ReadFile(name); err == nil {
					if face = parseFont(data); face != nil {
						return
					}
				}
			}
		}
	})
	return face
}

func u16(b []byte, i int) int {
	if i < 0 || i+2 > len(b) {
		return 0
	}
	return int(binary.BigEndian.Uint16(b[i:]))
}

func i16(b []byte, i int) float32 { return float32(int16(u16(b, i))) }

func sub(b []byte, off int) []byte {
	if off <= 0 || off >= len(b) {
		return nil
	}
	return b[off:]
}

// parseFont reads the first font of data, a font or a collection, that
// has a MATH table.
func parseFont(data []byte) *mathFont {
	col, err := sfnt.ParseCollection(data)
	if err != nil {
		return nil
	}
	dirs := []int{0}
	if string(data[:min(4, len(data))]) == "ttcf" {
		dirs = dirs[:0]
		for i := range col.NumFonts() {
			dirs = append(dirs, int(binary.BigEndian.Uint32(data[12+4*i:])))
		}
	}
	for i, dir := range dirs {
		table := mathTable(data, dir)
		if table == nil {
			continue
		}
		f, err := col.Font(i)
		if err != nil {
			continue
		}
		m := &mathFont{f: f, upem: float32(f.UnitsPerEm()), glyphs: map[sfnt.GlyphIndex]*outline{}}
		m.read(table)
		return m
	}
	return nil
}

// mathTable returns the MATH table of the font whose table directory is
// at dir.
func mathTable(data []byte, dir int) []byte {
	n := u16(data, dir+4)
	for i := range n {
		rec := dir + 12 + 16*i
		if rec+16 > len(data) {
			return nil
		}
		if string(data[rec:rec+4]) == "MATH" {
			off, size := int(binary.BigEndian.Uint32(data[rec+8:])), int(binary.BigEndian.Uint32(data[rec+12:]))
			if off+size > len(data) {
				return nil
			}
			return data[off : off+size]
		}
	}
	return nil
}

func (m *mathFont) read(t []byte) {
	em := func(v float32) float32 { return v / m.upem }
	k := sub(t, u16(t, 4))
	m.k[scriptPercentScaleDown] = i16(k, 0) / 100
	m.k[scriptScriptPercentScaleDown] = i16(k, 2) / 100
	m.k[delimitedSubFormulaMinHeight] = em(i16(k, 4))
	m.k[displayOperatorMinHeight] = em(i16(k, 6))
	for i := mathLeading; i < radicalDegreeBottomRaisePercent; i++ {
		m.k[i] = em(i16(k, 8+(i-mathLeading)*4))
	}
	m.k[radicalDegreeBottomRaisePercent] = i16(k, 8+(radicalDegreeBottomRaisePercent-mathLeading)*4) / 100

	info := sub(t, u16(t, 6))
	m.italics = m.values(sub(info, u16(info, 0)))
	m.accents = m.values(sub(info, u16(info, 2)))

	v := sub(t, u16(t, 8))
	m.cons = v
	nv, nh := u16(v, 6), u16(v, 8)
	m.vert, m.horiz = map[sfnt.GlyphIndex]int{}, map[sfnt.GlyphIndex]int{}
	for i, g := range coverage(sub(v, u16(v, 2))) {
		if i < nv {
			m.vert[g] = u16(v, 10+2*i)
		}
	}
	for i, g := range coverage(sub(v, u16(v, 4))) {
		if i < nh {
			m.horiz[g] = u16(v, 10+2*nv+2*i)
		}
	}
}

// values reads a table of a coverage and a value for each glyph it
// covers, as of italic corrections.
func (m *mathFont) values(t []byte) map[sfnt.GlyphIndex]float32 {
	out := map[sfnt.GlyphIndex]float32{}
	n := u16(t, 2)
	for i, g := range coverage(sub(t, u16(t, 0))) {
		if i < n {
			out[g] = i16(t, 4+4*i) / m.upem
		}
	}
	return out
}

// coverage returns the glyphs of a coverage table, by coverage index.
func coverage(t []byte) []sfnt.GlyphIndex {
	var out []sfnt.GlyphIndex
	switch u16(t, 0) {
	case 1:
		for i := range u16(t, 2) {
			out = append(out, sfnt.GlyphIndex(u16(t, 4+2*i)))
		}
	case 2:
		for i := range u16(t, 2) {
			r := 4 + 6*i
			for g := u16(t, r); g <= u16(t, r+2); g++ {
				out = append(out, sfnt.GlyphIndex(g))
			}
		}
	}
	return out
}

// rune returns the glyph of r, nil when the font has none.
func (m *mathFont) rune(r rune) *outline {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, err := m.f.GlyphIndex(&m.buf, r)
	if err != nil || g == 0 {
		return nil
	}
	return m.glyph(g)
}

// glyph returns glyph g, loading it the first time; m.mu is held.
func (m *mathFont) glyph(g sfnt.GlyphIndex) *outline {
	if o := m.glyphs[g]; o != nil {
		return o
	}
	ppem := fixed.I(int(m.upem))
	o := &outline{}
	if segs, err := m.f.LoadGlyph(&m.buf, g, ppem, nil); err == nil {
		for _, s := range segs {
			o.ops = append(o.ops, s.Op)
			for _, p := range s.Args[:[4]int{1, 1, 2, 3}[s.Op]] {
				o.pts = append(o.pts, [2]float32{float32(p.X) / 64 / m.upem, float32(p.Y) / 64 / m.upem})
			}
		}
	}
	adv, _ := m.f.GlyphAdvance(&m.buf, g, ppem, font.HintingNone)
	o.adv = float32(adv) / 64 / m.upem
	o.bound()
	o.italic = m.italics[g]
	if a, ok := m.accents[g]; ok {
		o.accent = a
	}
	m.glyphs[g] = o
	return o
}

// bound sets the box of o's ink, and where accents center over it.
func (o *outline) bound() {
	for i, p := range o.pts {
		if i == 0 {
			o.x0, o.y0, o.x1, o.y1 = p[0], p[1], p[0], p[1]
		}
		o.x0, o.y0, o.x1, o.y1 = min(o.x0, p[0]), min(o.y0, p[1]), max(o.x1, p[0]), max(o.y1, p[1])
	}
	o.accent = (o.x0 + o.x1) / 2
}

// construction returns how the glyph of r grows, up when vertical, else
// across; nil when it does not.
func (m *mathFont) construction(r rune, vertical bool) *construction {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, err := m.f.GlyphIndex(&m.buf, r)
	if err != nil || g == 0 {
		return nil
	}
	off, ok := m.horiz[g]
	if vertical {
		off, ok = m.vert[g]
	}
	t := sub(m.cons, off)
	if !ok || t == nil {
		return nil
	}
	c := &construction{}
	for i := range u16(t, 2) {
		r := 4 + 4*i
		c.variants = append(c.variants, m.glyph(sfnt.GlyphIndex(u16(t, r))))
		c.sizes = append(c.sizes, float32(u16(t, r+2))/m.upem)
	}
	if a := sub(t, u16(t, 0)); a != nil {
		for i := range u16(a, 4) {
			r := 6 + 10*i
			c.parts = append(c.parts, part{
				g:    m.glyph(sfnt.GlyphIndex(u16(a, r))),
				full: float32(u16(a, r+6)) / m.upem, extender: u16(a, r+8)&1 != 0,
			})
		}
	}
	return c
}

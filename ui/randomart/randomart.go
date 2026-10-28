// Package randomart draws the random art of a key's fingerprint, as
// ssh-keygen -lv and ssh -o VisualHostKey=yes print it, so that a glance
// tells two keys apart where their fingerprints' letters blur together: a
// bishop starts in the middle of a field of 17 by 9 squares and, two bits
// of the fingerprint at a time, steps diagonally, each square marked by
// how often it was visited, from . to ^, S where the walk started and E
// where it ended. The field shows in its frame, the key's type and size
// at the top and the hash at the bottom, with the squares tinted as often
// as they were visited, and the fingerprint below.
//
//	fp, err := randomart.ParseFingerprint("SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8")
//	randomart.View(c, "ED25519 256", fp)
package randomart

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	cols, rows = 17, 9
	// marks are what a square shows for how often it was visited; the last
	// two mark the start and the end.
	marks = " .o+=*BOX@%&#/^SE"
)

// Field returns how often the bishop visited each square, rows first,
// the start and end as 15 and 16 as OpenSSH marks them.
func Field(fp []byte) [rows][cols]int {
	var f [rows][cols]int
	x, y := cols/2, rows/2
	for _, b := range fp {
		for range 4 {
			if b&1 != 0 {
				x++
			} else {
				x--
			}
			if b&2 != 0 {
				y++
			} else {
				y--
			}
			x, y = max(0, min(cols-1, x)), max(0, min(rows-1, y))
			if f[y][x] < len(marks)-3 {
				f[y][x]++
			}
			b >>= 2
		}
	}
	f[rows/2][cols/2] = len(marks) - 2
	f[y][x] = len(marks) - 1
	return f
}

// Art returns the random art of fp as ssh-keygen prints it, framed, its
// title such as "ED25519 256" at the top and SHA256 at the bottom.
func Art(title string, fp []byte) string {
	f := Field(fp)
	var b strings.Builder
	b.WriteString(border("["+title+"]") + "\n")
	for _, row := range f {
		b.WriteByte('|')
		for _, v := range row {
			b.WriteByte(marks[v])
		}
		b.WriteString("|\n")
	}
	b.WriteString(border("[SHA256]"))
	return b.String()
}

// border returns a line of the frame with label in its middle.
func border(label string) string {
	if len(label) > cols {
		label = label[:cols]
	}
	left := (cols - len(label)) / 2
	return "+" + strings.Repeat("-", left) + label + strings.Repeat("-", cols-left-len(label)) + "+"
}

// ParseFingerprint returns the digest of a SHA256 fingerprint as
// ssh-keygen -l writes it.
func ParseFingerprint(s string) ([]byte, error) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(s), "SHA256:")
	if !ok {
		return nil, errors.New("randomart: not a SHA256 fingerprint")
	}
	fp, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(rest, "="))
	if err != nil || len(fp) != sha256.Size {
		return nil, errors.New("randomart: fingerprint is not 32 bytes of base64")
	}
	return fp, nil
}

// Fingerprint returns the SHA256 fingerprint of a public key, the bytes
// its base64 in an authorized_keys line decodes to.
func Fingerprint(key []byte) []byte {
	h := sha256.Sum256(key)
	return h[:]
}

// View creates the random art of fp, title naming its key's type and
// size.
func View(c *ui.Context, title string, fp []byte) ui.Element {
	t := c.Theme()
	f := Field(fp)
	art := Art(title, fp)
	size := t.FontSize + 1
	root := ui.Column(c).Gap(t.Space(2)).AlignItems(ui.Start).Role(ui.RoleImage).Label("Random art of " + title)
	root.Children(func() {
		text := ui.Text(c, art).Font("monospace").FontSize(size).LineHeight(1.25).Selectable()
		text.Draw(func(p *ui.Painter, r ui.Rect) {
			// The text is 19 characters by 11 lines; the field is inside its
			// frame.
			cw, lh := r.W/(cols+2), r.H/(rows+2)
			for y, row := range f {
				for x, v := range row {
					if v == 0 {
						continue
					}
					a := min(1, float32(v)/8) * 0.35
					col := t.Accent.Alpha(a)
					if v >= len(marks)-2 {
						col = t.Accent.Alpha(0.6)
					}
					p.Fill(ui.Rect{X: r.X + cw*float32(x+1), Y: r.Y + lh*float32(y+1), W: cw, H: lh}, col, 2)
				}
			}
		})
		ui.Text(c, "SHA256:"+base64.RawStdEncoding.EncodeToString(fp)).Font("monospace").FontSize(t.FontSize - 2).
			TextColor(t.TextMuted).Selectable()
	})
	return root
}

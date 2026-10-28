// Package identicon draws an avatar from any string, as GitHub draws the
// default avatars of accounts without a picture: a five by five grid of
// squares mirrored left to right, filled in one color on a pale ground,
// both picked from a hash of the string, so that the same name always
// draws the same picture and different names almost never do.
//
//	identicon.View(c, user.Login, 40)
package identicon

import (
	"crypto/sha256"

	"github.com/egoist/mygo/ui"
)

// Pattern returns which squares of the grid of value are filled, rows
// first, and their color.
func Pattern(value string) ([5][5]bool, ui.Color) {
	h := sha256.Sum256([]byte(value))
	var grid [5][5]bool
	// Each of the 15 squares of the left three columns takes a nibble;
	// the right two mirror the left two.
	for i := range 15 {
		row, col := i/3, i%3
		nib := h[i/2] >> (4 * (i % 2)) & 0xF
		on := nib%2 == 0
		grid[row][col], grid[row][4-col] = on, on
	}
	hue := float32(uint16(h[28])<<8|uint16(h[29])) / 65536 * 360
	chroma := 0.1 + float32(h[30])/255*0.06
	light := 0.55 + float32(h[31])/255*0.13
	return grid, ui.Oklch(light, chroma, hue)
}

// View creates the identicon of value, size DIPs wide.
func View(c *ui.Context, value string, size float32) ui.Element {
	t := c.Theme()
	grid, col := Pattern(value)
	ground := ui.RGB(240, 240, 240)
	if t.Dark {
		ground = ui.RGB(48, 48, 52)
	}
	e := ui.Box(c).Size(size, size).Background(ground).Radius(size * 0.12).Role(ui.RoleImage).Label(value)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		// Half a square of margin all round, as GitHub's have.
		cell := r.W / 6
		for row := range 5 {
			for k := range 5 {
				if grid[row][k] {
					x, y := r.X+cell/2+float32(k)*cell, r.Y+cell/2+float32(row)*cell
					// A hair more than a square, so that neighbors meet
					// without seams.
					p.Fill(ui.Rect{X: x, Y: y, W: cell + 0.5, H: cell + 0.5}, col, 0)
				}
			}
		}
	})
	return e
}

// Exif shows the metadata of a photo it paints of a sunset and saves as a
// JPEG with the Exif and GPS tags a phone would write. Pass a JPEG or a
// camera's raw TIFF file to read its metadata instead.
//
//	go run ./examples/exif [photo.jpg]
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/exif"
)

// entry is a TIFF directory entry to write: an ASCII string, or numbers
// of a type, rationals as numerator and denominator pairs.
type entry struct {
	id   uint16
	typ  uint16
	text string
	nums []int64
	dir  []entry
}

func ascii(id uint16, s string) entry       { return entry{id: id, typ: 2, text: s} }
func short(id uint16, v ...int64) entry     { return entry{id: id, typ: 3, nums: v} }
func rational(id uint16, v ...int64) entry  { return entry{id: id, typ: 5, nums: v} }
func srational(id uint16, v ...int64) entry { return entry{id: id, typ: 10, nums: v} }
func pointer(id uint16, dir ...entry) entry { return entry{id: id, typ: 4, dir: dir} }
func undefined(id uint16, b ...byte) entry {
	nums := make([]int64, len(b))
	for i, x := range b {
		nums[i] = int64(x)
	}
	return entry{id: id, typ: 7, nums: nums}
}

// size returns the bytes a directory takes with its values and the
// directories it points to.
func size(dir []entry) int {
	n := 2 + 12*len(dir) + 4
	for _, e := range dir {
		if v := len(value(e, 0)); v > 4 {
			n += v
		}
		if e.dir != nil {
			n += size(e.dir)
		}
	}
	return n
}

// value returns the bytes of e's value, sub its directory's offset.
func value(e entry, sub uint32) []byte {
	le := binary.LittleEndian
	var b []byte
	switch e.typ {
	case 2:
		b = append([]byte(e.text), 0)
	case 3:
		for _, v := range e.nums {
			b = le.AppendUint16(b, uint16(v))
		}
	case 4:
		b = le.AppendUint32(b, sub)
	case 5, 10:
		for _, v := range e.nums {
			b = le.AppendUint32(b, uint32(int32(v)))
		}
	case 7:
		for _, v := range e.nums {
			b = append(b, byte(v))
		}
	}
	return b
}

// write writes dir at offset at of the TIFF structure, then its values
// and the directories it points to.
func write(out []byte, dir []entry, at int) []byte {
	le := binary.LittleEndian
	data := at + 2 + 12*len(dir) + 4
	var tail []byte
	out = le.AppendUint16(out, uint16(len(dir)))
	for _, e := range dir {
		var sub uint32
		if e.dir != nil {
			sub = uint32(data + len(tail))
			tail = write(tail, e.dir, int(sub))
		}
		v := value(e, sub)
		count := len(v)
		switch e.typ {
		case 3:
			count /= 2
		case 4:
			count = 1
		case 5, 10:
			count /= 8
		}
		out = le.AppendUint16(out, e.id)
		out = le.AppendUint16(out, e.typ)
		out = le.AppendUint32(out, uint32(count))
		if len(v) <= 4 {
			out = append(out, append(v, make([]byte, 4-len(v))...)...)
		} else {
			out = le.AppendUint32(out, uint32(data+len(tail)))
			tail = append(tail, v...)
		}
	}
	out = le.AppendUint32(out, 0)
	return append(out, tail...)
}

func exifData(w, h int) []byte {
	gps := []entry{
		undefined(0x0000, 2, 3, 0, 0),
		ascii(0x0001, "N"), rational(0x0002, 37, 1, 48, 1, 3012, 100),
		ascii(0x0003, "W"), rational(0x0004, 122, 1, 28, 1, 3456, 100),
		undefined(0x0005, 0), rational(0x0006, 4250, 100),
		ascii(0x0010, "T"), rational(0x0011, 26400, 100),
	}
	sub := []entry{
		rational(0x829A, 1, 250), rational(0x829D, 18, 10), short(0x8822, 2), short(0x8827, 50),
		ascii(0x9003, "2026:09:21 19:04:37"), ascii(0x9011, "-07:00"), srational(0x9204, -1, 3),
		short(0x9207, 5), short(0x9209, 16), rational(0x920A, 677, 100), short(0xA001, 1),
		short(0xA002, int64(w)), short(0xA003, int64(h)), short(0xA403, 0), short(0xA405, 24),
		ascii(0xA433, "Apple"), ascii(0xA434, "iPhone 15 Pro back triple camera 6.765mm f/1.78"),
	}
	ifd0 := []entry{
		ascii(0x010F, "Apple"), ascii(0x0110, "iPhone 15 Pro"), short(0x0112, 1),
		ascii(0x0131, "17.0"), ascii(0x0132, "2026:09:21 19:04:37"), ascii(0x8298, "© Ada Lovelace"),
		pointer(0x8769, sub...), pointer(0x8825, gps...),
	}
	out := []byte("II*\x00\x08\x00\x00\x00")
	return write(out, ifd0, 8)
}

// photo paints a sunset over the sea with a hill, and saves it as a JPEG
// with Exif data.
func photo() []byte {
	const w, h = 960, 640
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			t := float64(y) / h
			r, g, b := 255-90*t, 120+60*(1-t)-40*t, 80+120*t
			if d := math.Hypot(float64(x-620), float64(y-330)); d < 46 {
				r, g, b = 255, 236, 180
			}
			if y > 360 {
				k := float64(y-360) / (h - 360)
				r, g, b = 40+80*(1-k), 60+40*(1-k), 110+40*(1-k)
				if math.Abs(float64(x-620)) < 30*(1-k)+4 && (y/6)%2 == 0 {
					r, g, b = 255, 200, 140
				}
			}
			if top := 250 + 0.0013*float64((x-150)*(x-150)); top < 362 && float64(y) > top {
				r, g, b = 30, 26, 40
			}
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		log.Fatal(err)
	}
	data := buf.Bytes()
	app1 := append([]byte("Exif\x00\x00"), exifData(w, h)...)
	seg := []byte{0xFF, 0xE1, byte((len(app1) + 2) >> 8), byte(len(app1) + 2)}
	return append(append(append([]byte{}, data[:2]...), append(seg, app1...)...), data[2:]...)
}

type app struct {
	pic  *ui.Bitmap
	meta *exif.Metadata
	err  error
	info exif.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		ui.Box(c).Grow(1).Background(ui.RGB(20, 20, 22)).Padding(t.Space(6)).Children(func() {
			if a.pic != nil {
				ui.Image(c, a.pic).Fill().Fit(ui.Contain)
			}
		})
		ui.Scroll(c).Width(t.Space(96)).Background(t.Surface).Border(0, t.Border).Children(func() {
			ui.Column(c).Padding(t.Space(5)).Children(func() {
				if a.err != nil {
					ui.Text(c, a.err.Error()).TextColor(t.Danger)
					return
				}
				exif.View(c, &a.info, a.meta)
			})
		})
	})
}

func main() {
	data := photo()
	if len(os.Args) > 1 {
		var err error
		if data, err = os.ReadFile(os.Args[1]); err != nil {
			log.Fatal(err)
		}
	}
	a := &app{}
	a.meta, a.err = exif.Parse(data)
	a.pic, _ = ui.DecodeBitmap(data)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Exif",
			Width:   1100,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// Package exif shows a photo's metadata, as the info panels of Photos and
// Lightroom do: the camera and lens, the exposure as four tiles (aperture,
// shutter speed, ISO and focal length), when it was taken, the image's
// size, how it was exposed, where it was taken, and its software and
// copyright, each row only when the photo says. Below them, every tag
// read, by directory, behind a disclosure. A click on the location copies
// its coordinates.
//
//	m, err := exif.Parse(data) // a JPEG or TIFF file's bytes
//	exif.View(c, &app.info, m)
//
// Parse reads the Exif and GPS directories of JPEG files and of TIFF
// files, as cameras' raw files are; maker notes stay unread.
package exif

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a panel keeps: whether every tag shows. Keep it in the
// app's state, one for each panel.
type State struct {
	all      bool
	copiedAt time.Time
}

// View creates the panel of m.
func View(c *ui.Context, s *State, m *Metadata) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(4)).Role(ui.RoleGroup).Label("Photo info")
	if m == nil {
		return root
	}
	root.Children(func() {
		camera, lens := m.camera()
		if camera != "" || lens != "" {
			ui.Column(c).Gap(t.Space(0.5)).Children(func() {
				if camera != "" {
					ui.Text(c, camera).FontWeight(600).FontSize(t.FontSize + 2)
				}
				if lens != "" {
					ui.Text(c, lens).TextColor(t.TextMuted)
				}
			})
		}
		if tiles := m.exposure(); len(tiles) > 0 {
			ui.Row(c).Background(t.Surface).Border(1, t.Border).Radius(t.Space(2)).Dividers(1, t.Border).Clip().Children(func() {
				for i, tl := range tiles {
					ui.Text(c.Key(i), tl).Grow(1).Basis(0).TextAlign(ui.Center).PaddingY(t.Space(2.5)).
						FontWeight(500).FontFeatures("tnum").NoWrap()
				}
			})
		}
		for _, sec := range m.sections() {
			if len(sec.rows) == 0 {
				continue
			}
			ui.Column(c.Key(sec.title)).Gap(t.Space(1.5)).Children(func() {
				ui.Text(c, sec.title).FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.TextMuted)
				for _, r := range sec.rows {
					row := ui.Row(c.Key(r[0])).Gap(t.Space(3)).Children(func() {
						ui.Text(c, r[0]).Width(t.Space(26)).TextColor(t.TextMuted)
						ui.Text(c, r[1]).Grow(1).FontFeatures("tnum").Selectable()
					})
					if sec.title == "Location" && r[0] == "Coordinates" {
						row.Cursor(ui.CursorPointer).Tooltip("Copy coordinates")
						if row.Clicked() {
							lat, lon, _ := m.location()
							c.WriteClipboard(fmt.Sprintf("%.6f, %.6f", lat, lon))
							s.copiedAt = c.Now()
						}
						if left := 1500*time.Millisecond - c.Now().Sub(s.copiedAt); left > 0 {
							ui.Text(c.Key("copied"), "Copied").FontSize(t.FontSize - 2).TextColor(t.Accent).PaddingStart(t.Space(29))
							c.After(left)
						}
					}
				}
			})
		}
		ui.Collapsible(c, fmt.Sprintf("All tags (%d)", len(m.Tags)), &s.all, func() {
			ui.Column(c).Gap(t.Space(1)).PaddingY(t.Space(1)).Children(func() {
				for i, tg := range m.Tags {
					if i == 0 || m.Tags[i-1].Dir != tg.Dir {
						ui.Text(c.Key("dir"+tg.Dir), tg.Dir).FontSize(t.FontSize - 2).FontWeight(600).TextColor(t.TextMuted).
							MarginY(t.Space(1))
					}
					ui.Row(c.Key(i)).Gap(t.Space(2)).Children(func() {
						ui.Text(c, tg.Name).Width(t.Space(36)).FontSize(t.FontSize - 2).Font("monospace").NoWrap().Ellipsis("…")
						ui.Text(c, tg.String()).Grow(1).Basis(0).FontSize(t.FontSize - 2).Font("monospace").Selectable().MaxLines(3)
					})
				}
			})
		})
	})
	return root
}

func (m *Metadata) camera() (camera, lens string) {
	mk, model := m.text("Image", 0x010F), m.text("Image", 0x0110)
	switch {
	case model == "":
		camera = mk
	case mk == "" || strings.HasPrefix(model, mk):
		camera = model
	default:
		camera = mk + " " + model
	}
	lens = m.text("Exif", 0xA434)
	if lens == "" {
		if spec, ok := m.find("Exif", 0xA432); ok && len(spec.Values) == 4 {
			lens = lensSpec(spec.Values)
		}
	}
	return camera, lens
}

// lensSpec writes a lens's focal lengths and widest apertures, as
// "24–70 mm ƒ/2.8".
func lensSpec(v []float64) string {
	focal := trim(v[0])
	if v[1] != v[0] {
		focal += "–" + trim(v[1])
	}
	ap := "ƒ/" + trim(v[2])
	if v[3] != v[2] && v[3] != 0 {
		ap += "–" + trim(v[3])
	}
	return focal + " mm " + ap
}

func (m *Metadata) exposure() []string {
	var out []string
	if f, ok := m.num("Exif", 0x829D); ok && f > 0 {
		out = append(out, "ƒ/"+trim(math.Round(f*10)/10))
	}
	if e, ok := m.num("Exif", 0x829A); ok && e > 0 {
		out = append(out, shutter(e))
	}
	if iso, ok := m.num("Exif", 0x8827); ok {
		out = append(out, "ISO "+trim(iso))
	}
	if f, ok := m.num("Exif", 0x920A); ok && f > 0 {
		out = append(out, trim(math.Round(f*10)/10)+" mm")
	}
	return out
}

// shutter writes an exposure time as cameras show it: 1/250 s, or 2.5 s.
func shutter(sec float64) string {
	if sec < 0.5 {
		return fmt.Sprintf("1/%d s", int(math.Round(1/sec)))
	}
	return trim(math.Round(sec*10)/10) + " s"
}

type section struct {
	title string
	rows  [][2]string
}

func (m *Metadata) sections() []section {
	var taken, image, exposure, location, other section
	taken.title, image.title, exposure.title, location.title, other.title = "Taken", "Image", "Exposure", "Location", "Other"
	add := func(s *section, label, value string) {
		if value != "" {
			s.rows = append(s.rows, [2]string{label, value})
		}
	}
	if when := m.text("Exif", 0x9003); when != "" {
		if tm, err := time.Parse("2006:01:02 15:04:05", when); err == nil {
			when = tm.Format("Monday, 2 January 2006 at 15:04:05")
		}
		if off := m.text("Exif", 0x9011); off != "" {
			when += " (UTC" + off + ")"
		}
		add(&taken, "Date", when)
	}
	w, okW := m.num("Exif", 0xA002)
	h, okH := m.num("Exif", 0xA003)
	if !okW || !okH {
		w, okW = m.num("Image", 0x0100)
		h, okH = m.num("Image", 0x0101)
	}
	if okW && okH {
		add(&image, "Size", fmt.Sprintf("%s × %s (%.1f MP)", trim(w), trim(h), w*h/1e6))
	}
	if cs, ok := m.num("Exif", 0xA001); ok {
		add(&image, "Color space", map[float64]string{1: "sRGB", 65535: "Uncalibrated"}[cs])
	}
	if o, ok := m.num("Image", 0x0112); ok && o > 1 {
		add(&image, "Orientation", orientations[int(o)])
	}
	if p, ok := m.num("Exif", 0x8822); ok {
		add(&exposure, "Program", programs[int(p)])
	}
	if mm, ok := m.num("Exif", 0x9207); ok {
		add(&exposure, "Metering", meterings[int(mm)])
	}
	if b, ok := m.num("Exif", 0x9204); ok {
		add(&exposure, "Compensation", fmt.Sprintf("%+.1f EV", b))
	}
	if f, ok := m.num("Exif", 0x9209); ok {
		fired := "Did not fire"
		if int(f)&1 != 0 {
			fired = "Fired"
		}
		add(&exposure, "Flash", fired)
	}
	if wb, ok := m.num("Exif", 0xA403); ok {
		add(&exposure, "White balance", map[float64]string{0: "Auto", 1: "Manual"}[wb])
	}
	if f, ok := m.num("Exif", 0xA405); ok && f > 0 {
		add(&exposure, "35 mm equiv.", trim(f)+" mm")
	}
	if lat, lon, ok := m.location(); ok {
		add(&location, "Coordinates", fmt.Sprintf("%s, %s", angle(lat, "N", "S"), angle(lon, "E", "W")))
		if alt, ok := m.num("GPS", 0x0006); ok {
			if ref, _ := m.find("GPS", 0x0005); len(ref.Raw) > 0 && ref.Raw[0] == 1 {
				alt = -alt
			}
			add(&location, "Altitude", fmt.Sprintf("%.0f m", alt))
		}
		if d, ok := m.num("GPS", 0x0011); ok {
			add(&location, "Facing", fmt.Sprintf("%.0f° %s", d, compass(d)))
		}
	}
	add(&other, "Software", m.text("Image", 0x0131))
	add(&other, "Artist", m.text("Image", 0x013B))
	add(&other, "Copyright", m.text("Image", 0x8298))
	return []section{taken, image, exposure, location, other}
}

// location returns where the photo was taken in signed degrees.
func (m *Metadata) location() (lat, lon float64, ok bool) {
	la, ok1 := m.find("GPS", 0x0002)
	lo, ok2 := m.find("GPS", 0x0004)
	if !ok1 || !ok2 || len(la.Values) != 3 || len(lo.Values) != 3 {
		return 0, 0, false
	}
	lat = la.Values[0] + la.Values[1]/60 + la.Values[2]/3600
	lon = lo.Values[0] + lo.Values[1]/60 + lo.Values[2]/3600
	if m.text("GPS", 0x0001) == "S" {
		lat = -lat
	}
	if m.text("GPS", 0x0003) == "W" {
		lon = -lon
	}
	return lat, lon, true
}

// angle writes a signed latitude or longitude as degrees, minutes and
// seconds toward pos or neg.
func angle(v float64, pos, neg string) string {
	dir := pos
	if v < 0 {
		v, dir = -v, neg
	}
	d := math.Floor(v)
	mins := (v - d) * 60
	return fmt.Sprintf("%.0f°%02.0f′%04.1f″ %s", d, math.Floor(mins), (mins-math.Floor(mins))*60, dir)
}

func compass(deg float64) string {
	names := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return names[int(math.Round(deg/45))%8]
}

var orientations = map[int]string{
	2: "Mirrored", 3: "Rotated 180°", 4: "Mirrored vertically", 5: "Mirrored, rotated 90° left",
	6: "Rotated 90° right", 7: "Mirrored, rotated 90° right", 8: "Rotated 90° left",
}

var programs = map[int]string{
	1: "Manual", 2: "Program", 3: "Aperture priority", 4: "Shutter priority", 5: "Creative", 6: "Action",
	7: "Portrait", 8: "Landscape",
}

var meterings = map[int]string{
	1: "Average", 2: "Center-weighted", 3: "Spot", 4: "Multi-spot", 5: "Pattern", 6: "Partial",
}

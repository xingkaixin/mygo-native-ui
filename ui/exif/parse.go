package exif

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Tag is a tag of a photo's metadata: the directory it is in ("Image",
// "Exif" or "GPS"), its number and name, and its values.
type Tag struct {
	Dir  string
	ID   uint16
	Name string
	// Text is the value of an ASCII tag.
	Text string
	// Values are the numbers of a numeric tag, rationals divided out.
	Values []float64
	// Raw is the bytes of an undefined or byte tag.
	Raw []byte
}

// String writes the tag's value as it reads.
func (t Tag) String() string {
	switch {
	case t.Text != "":
		return t.Text
	case len(t.Values) > 0:
		parts := make([]string, len(t.Values))
		for i, v := range t.Values {
			parts[i] = trim(v)
		}
		return strings.Join(parts, ", ")
	case len(t.Raw) > 16:
		return fmt.Sprintf("%d bytes", len(t.Raw))
	case isText(t.Raw):
		return string(t.Raw)
	}
	return fmt.Sprintf("% x", t.Raw)
}

func trim(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return fmt.Sprintf("%d", int64(v))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
}

func isText(b []byte) bool {
	for _, c := range b {
		if c < ' ' || c > '~' {
			return false
		}
	}
	return len(b) > 0
}

// Metadata is the tags of a photo, in the order they were read.
type Metadata struct {
	Tags []Tag
}

func (m *Metadata) find(dir string, id uint16) (Tag, bool) {
	for _, t := range m.Tags {
		if t.Dir == dir && t.ID == id {
			return t, true
		}
	}
	return Tag{}, false
}

func (m *Metadata) text(dir string, id uint16) string {
	t, _ := m.find(dir, id)
	return strings.TrimSpace(t.Text)
}

func (m *Metadata) num(dir string, id uint16) (float64, bool) {
	t, ok := m.find(dir, id)
	if !ok || len(t.Values) == 0 {
		return 0, false
	}
	return t.Values[0], true
}

// Parse reads the metadata of a JPEG file, or of a TIFF file such as a
// camera's raw file.
func Parse(data []byte) (*Metadata, error) {
	tiff, err := find(data)
	if err != nil {
		return nil, err
	}
	var bo binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return nil, errors.New("exif: no TIFF header")
	}
	if bo.Uint16(tiff[2:]) != 42 {
		return nil, errors.New("exif: not TIFF")
	}
	r := reader{b: tiff, bo: bo, m: &Metadata{}, seen: map[uint32]bool{}}
	r.dir("Image", bo.Uint32(tiff[4:]))
	if len(r.m.Tags) == 0 {
		return nil, errors.New("exif: no tags")
	}
	return r.m, nil
}

// find returns the TIFF structure in data: all of a TIFF file, or the
// Exif APP1 segment of a JPEG's.
func find(data []byte) ([]byte, error) {
	if len(data) >= 8 && (string(data[:4]) == "II*\x00" || string(data[:4]) == "MM\x00*") {
		return data, nil
	}
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, errors.New("exif: not a JPEG or TIFF file")
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return nil, errors.New("exif: bad JPEG marker")
		}
		marker := data[i+1]
		if marker == 0xD9 || marker == 0xDA {
			break
		}
		n := int(binary.BigEndian.Uint16(data[i+2:]))
		end := i + 2 + n
		if n < 2 || end > len(data) {
			return nil, errors.New("exif: JPEG segment cut short")
		}
		seg := data[i+4 : end]
		if marker == 0xE1 && len(seg) >= 14 && string(seg[:6]) == "Exif\x00\x00" {
			return seg[6:], nil
		}
		i = end
	}
	return nil, errors.New("exif: no Exif data")
}

type reader struct {
	b    []byte
	bo   binary.ByteOrder
	m    *Metadata
	seen map[uint32]bool
}

// sizes are how many bytes a value of each TIFF type takes.
var sizes = map[uint16]int{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 6: 1, 7: 1, 8: 2, 9: 4, 10: 8, 11: 4, 12: 8}

// dir reads the directory at off and the directories it points to, but
// not one twice, which a corrupt file could ask for forever.
func (r *reader) dir(name string, off uint32) {
	if r.seen[off] || int(off)+2 > len(r.b) {
		return
	}
	r.seen[off] = true
	n := int(r.bo.Uint16(r.b[off:]))
	for k := range n {
		e := int(off) + 2 + 12*k
		if e+12 > len(r.b) {
			return
		}
		id, typ, count := r.bo.Uint16(r.b[e:]), r.bo.Uint16(r.b[e+2:]), int(r.bo.Uint32(r.b[e+4:]))
		size, ok := sizes[typ]
		if !ok || count < 0 || count > 1<<20 {
			continue
		}
		// Values of four bytes or fewer sit in the entry itself.
		at := e + 8
		if size*count > 4 {
			at = int(r.bo.Uint32(r.b[e+8:]))
		}
		if at+size*count > len(r.b) {
			continue
		}
		val := r.b[at : at+size*count]
		switch id {
		case 0x8769:
			r.dir("Exif", r.bo.Uint32(val))
			continue
		case 0x8825:
			r.dir("GPS", r.bo.Uint32(val))
			continue
		case 0xA005:
			r.dir("Interop", r.bo.Uint32(val))
			continue
		}
		t := Tag{Dir: name, ID: id, Name: tagName(name, id)}
		switch typ {
		case 2:
			t.Text = strings.TrimRight(string(val), "\x00 ")
		case 1, 7:
			t.Raw = val
		default:
			for i := range count {
				t.Values = append(t.Values, r.number(typ, val[i*size:]))
			}
		}
		r.m.Tags = append(r.m.Tags, t)
	}
	if name == "Image" {
		if e := int(off) + 2 + 12*n; e+4 <= len(r.b) {
			if next := r.bo.Uint32(r.b[e:]); next != 0 {
				r.dir("Thumbnail", next)
			}
		}
	}
}

func (r *reader) number(typ uint16, b []byte) float64 {
	switch typ {
	case 3:
		return float64(r.bo.Uint16(b))
	case 4:
		return float64(r.bo.Uint32(b))
	case 5:
		num, den := r.bo.Uint32(b), r.bo.Uint32(b[4:])
		if den == 0 {
			return 0
		}
		return float64(num) / float64(den)
	case 6:
		return float64(int8(b[0]))
	case 8:
		return float64(int16(r.bo.Uint16(b)))
	case 9:
		return float64(int32(r.bo.Uint32(b)))
	case 10:
		num, den := int32(r.bo.Uint32(b)), int32(r.bo.Uint32(b[4:]))
		if den == 0 {
			return 0
		}
		return float64(num) / float64(den)
	case 11:
		return float64(math.Float32frombits(r.bo.Uint32(b)))
	case 12:
		return math.Float64frombits(r.bo.Uint64(b))
	}
	return float64(b[0])
}

func tagName(dir string, id uint16) string {
	names := tags
	if dir == "GPS" {
		names = gpsTags
	}
	if n, ok := names[id]; ok {
		return n
	}
	return fmt.Sprintf("0x%04X", id)
}

var tags = map[uint16]string{
	0x0100: "ImageWidth", 0x0101: "ImageLength", 0x0102: "BitsPerSample", 0x0103: "Compression",
	0x010E: "ImageDescription", 0x010F: "Make", 0x0110: "Model", 0x0112: "Orientation",
	0x011A: "XResolution", 0x011B: "YResolution", 0x0128: "ResolutionUnit", 0x0131: "Software",
	0x0132: "DateTime", 0x013B: "Artist", 0x0201: "JPEGInterchangeFormat", 0x0202: "JPEGInterchangeFormatLength",
	0x0213: "YCbCrPositioning", 0x8298: "Copyright",
	0x829A: "ExposureTime", 0x829D: "FNumber", 0x8822: "ExposureProgram", 0x8827: "ISOSpeedRatings",
	0x8830: "SensitivityType", 0x9000: "ExifVersion", 0x9003: "DateTimeOriginal", 0x9004: "DateTimeDigitized",
	0x9010: "OffsetTime", 0x9011: "OffsetTimeOriginal", 0x9101: "ComponentsConfiguration",
	0x9201: "ShutterSpeedValue", 0x9202: "ApertureValue", 0x9203: "BrightnessValue", 0x9204: "ExposureBiasValue",
	0x9205: "MaxApertureValue", 0x9207: "MeteringMode", 0x9208: "LightSource", 0x9209: "Flash",
	0x920A: "FocalLength", 0x9214: "SubjectArea", 0x927C: "MakerNote", 0x9286: "UserComment",
	0x9290: "SubSecTime", 0x9291: "SubSecTimeOriginal", 0xA000: "FlashpixVersion", 0xA001: "ColorSpace",
	0xA002: "PixelXDimension", 0xA003: "PixelYDimension", 0xA217: "SensingMethod", 0xA301: "SceneType",
	0xA401: "CustomRendered", 0xA402: "ExposureMode", 0xA403: "WhiteBalance", 0xA404: "DigitalZoomRatio",
	0xA405: "FocalLengthIn35mmFilm", 0xA406: "SceneCaptureType", 0xA420: "ImageUniqueID",
	0xA431: "BodySerialNumber", 0xA432: "LensSpecification", 0xA433: "LensMake", 0xA434: "LensModel",
}

var gpsTags = map[uint16]string{
	0x0000: "GPSVersionID", 0x0001: "GPSLatitudeRef", 0x0002: "GPSLatitude", 0x0003: "GPSLongitudeRef",
	0x0004: "GPSLongitude", 0x0005: "GPSAltitudeRef", 0x0006: "GPSAltitude", 0x0007: "GPSTimeStamp",
	0x000C: "GPSSpeedRef", 0x000D: "GPSSpeed", 0x0010: "GPSImgDirectionRef", 0x0011: "GPSImgDirection",
	0x0017: "GPSDestBearingRef", 0x0018: "GPSDestBearing", 0x001D: "GPSDateStamp", 0x001F: "GPSHPositioningError",
}

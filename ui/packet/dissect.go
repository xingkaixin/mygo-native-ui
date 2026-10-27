package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"
)

// Summary is a packet in a line, as the columns of Wireshark's packet
// list show it.
type Summary struct {
	Source, Destination string
	Protocol            string
	Info                string
}

// Dissect reads an Ethernet II frame into fields: ARP, IPv4 and IPv6 in
// it, TCP, UDP and ICMP in those, and HTTP/1 in TCP, with what follows as
// data. A packet cut short ends with a field saying so.
func Dissect(data []byte) ([]Field, Summary) {
	d := &dissector{data: data}
	d.ethernet()
	return d.fields, d.sum
}

type dissector struct {
	data   []byte
	fields []Field
	sum    Summary
}

func (d *dissector) short(off, n int) bool {
	if off+n <= len(d.data) {
		return false
	}
	d.fields = append(d.fields, Field{Label: "[Packet cut short]", Offset: off, Length: len(d.data) - off})
	return true
}

func (d *dissector) u16(off int) uint16 { return binary.BigEndian.Uint16(d.data[off:]) }
func (d *dissector) u32(off int) uint32 { return binary.BigEndian.Uint32(d.data[off:]) }

func mac(b []byte) string {
	s := make([]string, len(b))
	for i, v := range b {
		s[i] = fmt.Sprintf("%02x", v)
	}
	return strings.Join(s, ":")
}

// bits writes the bits of v, width of them, that mask covers, the others
// as dots, in fours: "0100 .... = Version: 4".
func bits(v uint32, mask uint32, width int, label string) string {
	var b strings.Builder
	for i := width - 1; i >= 0; i-- {
		switch {
		case mask&(1<<i) == 0:
			b.WriteByte('.')
		case v&(1<<i) != 0:
			b.WriteByte('1')
		default:
			b.WriteByte('0')
		}
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
	}
	return b.String() + " = " + label
}

func set(v bool) string {
	if v {
		return "Set"
	}
	return "Not set"
}

func (d *dissector) ethernet() {
	if d.short(0, 14) {
		return
	}
	dst, src := mac(d.data[0:6]), mac(d.data[6:12])
	kind := d.u16(12)
	d.fields = append(d.fields, Field{
		Label: "Ethernet II, Src: " + src + ", Dst: " + dst, Length: 14,
		Fields: []Field{
			{Label: "Destination: " + dst, Offset: 0, Length: 6},
			{Label: "Source: " + src, Offset: 6, Length: 6},
			{Label: fmt.Sprintf("Type: %s (0x%04x)", etherType(kind), kind), Offset: 12, Length: 2},
		},
	})
	d.sum = Summary{Source: src, Destination: dst, Protocol: "Ethernet", Info: fmt.Sprintf("Ethertype 0x%04x", kind)}
	switch kind {
	case 0x0806:
		d.arp(14)
	case 0x0800:
		d.ipv4(14)
	case 0x86dd:
		d.ipv6(14)
	default:
		d.rest(14)
	}
}

func etherType(t uint16) string {
	switch t {
	case 0x0800:
		return "IPv4"
	case 0x0806:
		return "ARP"
	case 0x86dd:
		return "IPv6"
	}
	return "Unknown"
}

func (d *dissector) arp(off int) {
	if d.short(off, 28) {
		return
	}
	op := d.u16(off + 6)
	sha, spa := mac(d.data[off+8:off+14]), ip4(d.data[off+14:off+18])
	tha, tpa := mac(d.data[off+18:off+24]), ip4(d.data[off+24:off+28])
	name, ok := map[uint16]string{1: "request", 2: "reply"}[op]
	if !ok {
		name = "unknown"
	}
	d.fields = append(d.fields, Field{
		Label: "Address Resolution Protocol (" + name + ")", Offset: off, Length: 28,
		Fields: []Field{
			{Label: fmt.Sprintf("Hardware type: Ethernet (%d)", d.u16(off)), Offset: off, Length: 2},
			{Label: fmt.Sprintf("Protocol type: IPv4 (0x%04x)", d.u16(off+2)), Offset: off + 2, Length: 2},
			{Label: fmt.Sprintf("Hardware size: %d", d.data[off+4]), Offset: off + 4, Length: 1},
			{Label: fmt.Sprintf("Protocol size: %d", d.data[off+5]), Offset: off + 5, Length: 1},
			{Label: fmt.Sprintf("Opcode: %s (%d)", name, op), Offset: off + 6, Length: 2},
			{Label: "Sender MAC address: " + sha, Offset: off + 8, Length: 6},
			{Label: "Sender IP address: " + spa, Offset: off + 14, Length: 4},
			{Label: "Target MAC address: " + tha, Offset: off + 18, Length: 6},
			{Label: "Target IP address: " + tpa, Offset: off + 24, Length: 4},
		},
	})
	d.sum.Protocol = "ARP"
	if op == 2 {
		d.sum.Info = spa + " is at " + sha
	} else {
		d.sum.Info = "Who has " + tpa + "? Tell " + spa
	}
}

func ip4(b []byte) string { return netip.AddrFrom4([4]byte(b)).String() }

func (d *dissector) ipv4(off int) {
	if d.short(off, 20) {
		return
	}
	b := d.data[off:]
	ihl := int(b[0]&0x0f) * 4
	if ihl < 20 || d.short(off, ihl) {
		return
	}
	total := int(d.u16(off + 2))
	flags := d.u16(off + 6)
	proto := b[9]
	src, dst := ip4(b[12:16]), ip4(b[16:20])
	f := Field{
		Label: "Internet Protocol Version 4, Src: " + src + ", Dst: " + dst, Offset: off, Length: ihl,
		Fields: []Field{
			{Label: bits(uint32(b[0]), 0xf0, 8, fmt.Sprintf("Version: %d", b[0]>>4)), Offset: off, Length: 1},
			{Label: bits(uint32(b[0]), 0x0f, 8, fmt.Sprintf("Header Length: %d bytes (%d)", ihl, ihl/4)), Offset: off, Length: 1},
			{Label: fmt.Sprintf("Differentiated Services Field: 0x%02x", b[1]), Offset: off + 1, Length: 1},
			{Label: fmt.Sprintf("Total Length: %d", total), Offset: off + 2, Length: 2},
			{Label: fmt.Sprintf("Identification: 0x%04x (%d)", d.u16(off+4), d.u16(off+4)), Offset: off + 4, Length: 2},
			{Label: fmt.Sprintf("Flags: 0x%x", flags>>13), Offset: off + 6, Length: 1, Fields: []Field{
				{Label: bits(uint32(b[6]), 0x80, 8, "Reserved bit: "+set(flags&0x8000 != 0)), Offset: off + 6, Length: 1},
				{Label: bits(uint32(b[6]), 0x40, 8, "Don't fragment: "+set(flags&0x4000 != 0)), Offset: off + 6, Length: 1},
				{Label: bits(uint32(b[6]), 0x20, 8, "More fragments: "+set(flags&0x2000 != 0)), Offset: off + 6, Length: 1},
			}},
			{Label: bits(uint32(flags), 0x1fff, 16, fmt.Sprintf("Fragment Offset: %d", flags&0x1fff)), Offset: off + 6, Length: 2},
			{Label: fmt.Sprintf("Time to Live: %d", b[8]), Offset: off + 8, Length: 1},
			{Label: fmt.Sprintf("Protocol: %s (%d)", ipProto(proto), proto), Offset: off + 9, Length: 1},
			{Label: fmt.Sprintf("Header Checksum: 0x%04x", d.u16(off+10)), Offset: off + 10, Length: 2},
			{Label: "Source Address: " + src, Offset: off + 12, Length: 4},
			{Label: "Destination Address: " + dst, Offset: off + 16, Length: 4},
		},
	}
	if ihl > 20 {
		f.Fields = append(f.Fields, Field{Label: fmt.Sprintf("Options: (%d bytes)", ihl-20), Offset: off + 20, Length: ihl - 20})
	}
	d.fields = append(d.fields, f)
	d.sum.Source, d.sum.Destination, d.sum.Protocol = src, dst, "IPv4"
	end := min(len(d.data), max(off+ihl, off+total))
	d.transport(proto, off+ihl, end)
}

func (d *dissector) ipv6(off int) {
	if d.short(off, 40) {
		return
	}
	b := d.data[off:]
	head := d.u32(off)
	next := b[6]
	src := netip.AddrFrom16([16]byte(b[8:24])).String()
	dst := netip.AddrFrom16([16]byte(b[24:40])).String()
	payload := int(d.u16(off + 4))
	d.fields = append(d.fields, Field{
		Label: "Internet Protocol Version 6, Src: " + src + ", Dst: " + dst, Offset: off, Length: 40,
		Fields: []Field{
			{Label: bits(uint32(b[0]), 0xf0, 8, fmt.Sprintf("Version: %d", b[0]>>4)), Offset: off, Length: 1},
			{Label: fmt.Sprintf("Traffic Class: 0x%02x", head>>20&0xff), Offset: off, Length: 2},
			{Label: fmt.Sprintf("Flow Label: 0x%05x", head&0xfffff), Offset: off + 1, Length: 3},
			{Label: fmt.Sprintf("Payload Length: %d", payload), Offset: off + 4, Length: 2},
			{Label: fmt.Sprintf("Next Header: %s (%d)", ipProto(next), next), Offset: off + 6, Length: 1},
			{Label: fmt.Sprintf("Hop Limit: %d", b[7]), Offset: off + 7, Length: 1},
			{Label: "Source Address: " + src, Offset: off + 8, Length: 16},
			{Label: "Destination Address: " + dst, Offset: off + 24, Length: 16},
		},
	})
	d.sum.Source, d.sum.Destination, d.sum.Protocol = src, dst, "IPv6"
	d.transport(next, off+40, min(len(d.data), off+40+payload))
}

func ipProto(p byte) string {
	switch p {
	case 1:
		return "ICMP"
	case 6:
		return "TCP"
	case 17:
		return "UDP"
	case 58:
		return "ICMPv6"
	}
	return "Unknown"
}

// transport reads the protocol carried from off to end.
func (d *dissector) transport(proto byte, off, end int) {
	switch proto {
	case 6:
		d.tcp(off, end)
	case 17:
		d.udp(off, end)
	case 1:
		d.icmp(off, end)
	default:
		d.rest(off)
	}
}

func (d *dissector) tcp(off, end int) {
	if d.short(off, 20) {
		return
	}
	b := d.data[off:]
	sp, dp := d.u16(off), d.u16(off+2)
	seq, ack := d.u32(off+4), d.u32(off+8)
	hl := int(b[12]>>4) * 4
	if hl < 20 || d.short(off, hl) {
		return
	}
	flags := uint32(d.u16(off+12)) & 0x0fff
	win := d.u16(off + 14)
	n := max(0, end-off-hl)
	names := []struct {
		bit  uint32
		name string
	}{{0x100, "Accurate ECN"}, {0x080, "Congestion Window Reduced"}, {0x040, "ECN-Echo"}, {0x020, "Urgent"},
		{0x010, "Acknowledgment"}, {0x008, "Push"}, {0x004, "Reset"}, {0x002, "Syn"}, {0x001, "Fin"}}
	brief := []struct {
		bit  uint32
		name string
	}{{0x002, "SYN"}, {0x001, "FIN"}, {0x004, "RST"}, {0x008, "PSH"}, {0x010, "ACK"}, {0x020, "URG"}}
	var on []string
	for _, s := range brief {
		if flags&s.bit != 0 {
			on = append(on, s.name)
		}
	}
	flagFields := make([]Field, len(names))
	for i, f := range names {
		flagFields[i] = Field{Label: bits(flags, f.bit, 12, f.name+": "+set(flags&f.bit != 0)), Offset: off + 12, Length: 2}
	}
	f := Field{
		Label:  fmt.Sprintf("Transmission Control Protocol, Src Port: %d, Dst Port: %d, Seq: %d, Ack: %d, Len: %d", sp, dp, seq, ack, n),
		Offset: off, Length: hl,
		Fields: []Field{
			{Label: fmt.Sprintf("Source Port: %d", sp), Offset: off, Length: 2},
			{Label: fmt.Sprintf("Destination Port: %d", dp), Offset: off + 2, Length: 2},
			{Label: fmt.Sprintf("Sequence Number: %d", seq), Offset: off + 4, Length: 4},
			{Label: fmt.Sprintf("Acknowledgment Number: %d", ack), Offset: off + 8, Length: 4},
			{Label: bits(uint32(b[12]), 0xf0, 8, fmt.Sprintf("Header Length: %d bytes (%d)", hl, hl/4)), Offset: off + 12, Length: 1},
			{Label: fmt.Sprintf("Flags: 0x%03x (%s)", flags, strings.Join(on, ", ")), Offset: off + 12, Length: 2, Fields: flagFields},
			{Label: fmt.Sprintf("Window: %d", win), Offset: off + 14, Length: 2},
			{Label: fmt.Sprintf("Checksum: 0x%04x", d.u16(off+16)), Offset: off + 16, Length: 2},
			{Label: fmt.Sprintf("Urgent Pointer: %d", d.u16(off+18)), Offset: off + 18, Length: 2},
		},
	}
	if hl > 20 {
		f.Fields = append(f.Fields, Field{Label: fmt.Sprintf("Options: (%d bytes)", hl-20), Offset: off + 20, Length: hl - 20})
	}
	d.fields = append(d.fields, f)
	d.sum.Protocol = "TCP"
	d.sum.Info = fmt.Sprintf("%d → %d [%s] Seq=%d Win=%d Len=%d", sp, dp, strings.Join(on, ", "), seq, win, n)
	if flags&0x010 != 0 {
		d.sum.Info = fmt.Sprintf("%d → %d [%s] Seq=%d Ack=%d Win=%d Len=%d", sp, dp, strings.Join(on, ", "), seq, ack, win, n)
	}
	if n > 0 && !d.http(off+hl, off+hl+n) {
		d.rest(off + hl)
	}
}

// http reads the head of an HTTP/1 request or response from off to end,
// and reports whether there was one.
func (d *dissector) http(off, end int) bool {
	p := d.data[off:end]
	methods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH ", "HTTP/1."}
	if !hasAny(p, methods) {
		return false
	}
	head := len(p)
	if i := bytes.Index(p, []byte("\r\n\r\n")); i >= 0 {
		head = i + 4
	}
	f := Field{Label: "Hypertext Transfer Protocol", Offset: off, Length: head}
	at := off
	for line := range strings.SplitSeq(string(p[:head]), "\r\n") {
		if line != "" {
			f.Fields = append(f.Fields, Field{Label: line, Offset: at, Length: len(line) + 2})
		}
		at += len(line) + 2
	}
	d.fields = append(d.fields, f)
	d.sum.Protocol = "HTTP"
	if len(f.Fields) > 0 {
		d.sum.Info = f.Fields[0].Label
	}
	if off+head < end {
		d.rest(off + head)
	}
	return true
}

func hasAny(p []byte, prefixes []string) bool {
	for _, s := range prefixes {
		if bytes.HasPrefix(p, []byte(s)) {
			return true
		}
	}
	return false
}

func (d *dissector) udp(off, end int) {
	if d.short(off, 8) {
		return
	}
	sp, dp, n := d.u16(off), d.u16(off+2), d.u16(off+4)
	d.fields = append(d.fields, Field{
		Label: fmt.Sprintf("User Datagram Protocol, Src Port: %d, Dst Port: %d", sp, dp), Offset: off, Length: 8,
		Fields: []Field{
			{Label: fmt.Sprintf("Source Port: %d", sp), Offset: off, Length: 2},
			{Label: fmt.Sprintf("Destination Port: %d", dp), Offset: off + 2, Length: 2},
			{Label: fmt.Sprintf("Length: %d", n), Offset: off + 4, Length: 2},
			{Label: fmt.Sprintf("Checksum: 0x%04x", d.u16(off+6)), Offset: off + 6, Length: 2},
		},
	})
	d.sum.Protocol = "UDP"
	d.sum.Info = fmt.Sprintf("%d → %d Len=%d", sp, dp, max(0, int(n)-8))
	if off+8 < end {
		d.rest(off + 8)
	}
}

func (d *dissector) icmp(off, end int) {
	if d.short(off, 8) {
		return
	}
	kind, code := d.data[off], d.data[off+1]
	id, seq := d.u16(off+4), d.u16(off+6)
	name, ok := map[byte]string{0: "Echo (ping) reply", 3: "Destination unreachable", 8: "Echo (ping) request", 11: "Time-to-live exceeded"}[kind]
	if !ok {
		name = "Unknown"
	}
	f := Field{
		Label: "Internet Control Message Protocol", Offset: off, Length: 8,
		Fields: []Field{
			{Label: fmt.Sprintf("Type: %d (%s)", kind, name), Offset: off, Length: 1},
			{Label: fmt.Sprintf("Code: %d", code), Offset: off + 1, Length: 1},
			{Label: fmt.Sprintf("Checksum: 0x%04x", d.u16(off+2)), Offset: off + 2, Length: 2},
		},
	}
	d.sum.Protocol, d.sum.Info = "ICMP", name
	if kind == 0 || kind == 8 {
		f.Fields = append(f.Fields,
			Field{Label: fmt.Sprintf("Identifier: %d (0x%04x)", id, id), Offset: off + 4, Length: 2},
			Field{Label: fmt.Sprintf("Sequence Number: %d (0x%04x)", seq, seq), Offset: off + 6, Length: 2})
		d.sum.Info = fmt.Sprintf("%s id=0x%04x, seq=%d", name, id, seq)
	}
	d.fields = append(d.fields, f)
	if off+8 < end {
		d.rest(off + 8)
	}
}

// rest adds the bytes from off to the end as data.
func (d *dissector) rest(off int) {
	if n := len(d.data) - off; n > 0 {
		d.fields = append(d.fields, Field{Label: fmt.Sprintf("Data (%d bytes)", n), Offset: off, Length: n})
	}
}

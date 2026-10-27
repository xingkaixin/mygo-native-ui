// Packet looks through a short capture of a laptop fetching a web page:
// choose a packet in the list, then a field to light its bytes, or click
// a byte to find its field.
//
//	go run ./examples/packet
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/netip"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/packet"
)

var (
	laptop    = []byte{0x3c, 0x22, 0xfb, 0x8a, 0x14, 0x07}
	router    = []byte{0xa4, 0x91, 0xb1, 0x02, 0x5e, 0xc9}
	broadcast = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	here      = netip.MustParseAddr("192.168.1.24")
	gateway   = netip.MustParseAddr("192.168.1.1")
	server    = netip.MustParseAddr("93.184.215.14")
	here6     = netip.MustParseAddr("fe80::3e22:fbff:fe8a:1407")
	mdns6     = netip.MustParseAddr("ff02::fb")
)

func be16(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }
func be32(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

func cat(parts ...[]byte) []byte {
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

func eth(dst, src []byte, kind uint16, payload []byte) []byte {
	return cat(dst, src, be16(kind), payload)
}

func ipv4(src, dst netip.Addr, proto byte, id uint16, payload []byte) []byte {
	head := cat([]byte{0x45, 0}, be16(uint16(20+len(payload))), be16(id), be16(0x4000), []byte{64, proto}, be16(0), src.AsSlice(), dst.AsSlice())
	var sum uint32
	for i := 0; i < 20; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(head[i:]))
	}
	for sum > 0xffff {
		sum = sum&0xffff + sum>>16
	}
	binary.BigEndian.PutUint16(head[10:], ^uint16(sum))
	return cat(head, payload)
}

func tcp(sp, dp uint16, seq, ack uint32, flags uint16, payload []byte) []byte {
	return cat(be16(sp), be16(dp), be32(seq), be32(ack), be16(5<<12|flags), be16(64240), be16(0x9b3e), be16(0), payload)
}

func udp(sp, dp uint16, payload []byte) []byte {
	return cat(be16(sp), be16(dp), be16(uint16(8+len(payload))), be16(0x51c2), payload)
}

const (
	fin = 1 << iota
	syn
	rst
	psh
	ack
)

var request = "GET / HTTP/1.1\r\nHost: example.com\r\nUser-Agent: curl/8.7.1\r\nAccept: */*\r\n\r\n"

var response = "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Length: 38\r\n\r\n<!doctype html><title>Example</title>"

var capture = [][]byte{
	eth(broadcast, laptop, 0x0806, cat(be16(1), be16(0x0800), []byte{6, 4}, be16(1), laptop, here.AsSlice(), make([]byte, 6), gateway.AsSlice())),
	eth(laptop, router, 0x0806, cat(be16(1), be16(0x0800), []byte{6, 4}, be16(2), router, gateway.AsSlice(), laptop, here.AsSlice())),
	eth(router, laptop, 0x0800, ipv4(here, gateway, 17, 0x1c46, udp(52144, 53, []byte("\x8a\x21\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x07example\x03com\x00\x00\x01\x00\x01")))),
	eth(laptop, router, 0x0800, ipv4(gateway, here, 17, 0x0b12, udp(53, 52144, []byte("\x8a\x21\x81\x80\x00\x01\x00\x01\x00\x00\x00\x00\x07example\x03com\x00\x00\x01\x00\x01\xc0\x0c\x00\x01\x00\x01\x00\x00\x0e\x10\x00\x04\x5d\xb8\xd7\x0e")))),
	eth(router, laptop, 0x0800, ipv4(here, server, 6, 0x5a01, tcp(51234, 80, 0, 0, syn, nil))),
	eth(laptop, router, 0x0800, ipv4(server, here, 6, 0x0000, tcp(80, 51234, 0, 1, syn|ack, nil))),
	eth(router, laptop, 0x0800, ipv4(here, server, 6, 0x5a02, tcp(51234, 80, 1, 1, ack, nil))),
	eth(router, laptop, 0x0800, ipv4(here, server, 6, 0x5a03, tcp(51234, 80, 1, 1, psh|ack, []byte(request)))),
	eth(laptop, router, 0x0800, ipv4(server, here, 6, 0x7e41, tcp(80, 51234, 1, 1+uint32(len(request)), psh|ack, []byte(response)))),
	eth(router, laptop, 0x0800, ipv4(here, server, 6, 0x5a04, tcp(51234, 80, 1+uint32(len(request)), 1+uint32(len(response)), fin|ack, nil))),
	eth(router, laptop, 0x0800, ipv4(here, gateway, 1, 0x3f20, cat([]byte{8, 0}, be16(0xf7fd), be16(1), be16(1), []byte("abcdefghijklmnop")))),
	eth(laptop, router, 0x0800, ipv4(gateway, here, 1, 0xa0c3, cat([]byte{0, 0}, be16(0xfffd), be16(1), be16(1), []byte("abcdefghijklmnop")))),
	eth([]byte{0x33, 0x33, 0, 0, 0, 0xfb}, laptop, 0x86dd, cat(be32(0x60000000), be16(8+12), []byte{17, 255}, here6.AsSlice(), mdns6.AsSlice(), udp(5353, 5353, make([]byte, 12)))),
}

type app struct {
	list    ui.ListState
	chosen  int
	details packet.State
	split   float32
}

var columns = []ui.TableColumn{
	{Title: "No.", Width: 44, Align: ui.End},
	{Title: "Source", Width: 170},
	{Title: "Destination", Width: 170},
	{Title: "Protocol", Width: 72},
	{Title: "Length", Width: 60, Align: ui.End},
	{Title: "Info"},
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	sums := make([]packet.Summary, len(capture))
	for i, p := range capture {
		_, sums[i] = packet.Dissect(p)
	}
	a.list.Selected = &a.chosen
	ui.SplitVerticalPercent(c, &a.split, func() {
		ui.Table(c, &a.list, columns, len(capture), func(row, col int) {
			s := sums[row]
			text := []string{fmt.Sprint(row + 1), s.Source, s.Destination, s.Protocol, fmt.Sprint(len(capture[row])), s.Info}[col]
			ui.Text(c, text).SingleLine().Ellipsis("…")
		}).Grow(1)
	}, func() {
		fields, _ := packet.Dissect(capture[a.chosen])
		packet.View(c, &a.details, fields, capture[a.chosen]).Grow(1)
	}).Fill().Background(t.Background)
}

func main() {
	a := &app{split: 34}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Packet",
			Width:   1000,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

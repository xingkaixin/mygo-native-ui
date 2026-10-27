// Package cidr plans the subnets of a network, as the Visual Subnet
// Calculator does: a network divided into subnets, each of which Divide
// splits in halves and Join puts back together with its other half. A bar
// shows the subnets in the network's addresses as wide as their share,
// and the bits of the address of the subnet chosen show which the network
// fixes, which the subnet fixes, and which are left for its hosts. A
// subnet overlapping a range taken elsewhere, as a peered network's, is
// marked.
//
//	plan := &cidr.Plan{Network: netip.MustParsePrefix("10.0.0.0/16")}
//	cidr.View(c, &app.cidr, plan).Grow(1)
//
// IPv4 and IPv6 networks are both planned.
package cidr

import (
	"fmt"
	"math"
	"math/big"
	"net/netip"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Plan is a network and its subnets: the subnets in address order, which
// cover the network, and ranges taken elsewhere that they must not
// overlap. A plan whose subnets don't cover its network starts again
// from the network whole.
type Plan struct {
	Network netip.Prefix
	Subnets []netip.Prefix
	Taken   []netip.Prefix
}

// State is what a view keeps: the subnet chosen and the rows' place.
// Keep it in the app's state, one for each view.
type State struct {
	chosen int
	list   ui.ListState
}

// View creates the planner of p, whose Subnets it changes. Changed
// reports a change. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, p *Plan) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(4))
	if !p.Network.IsValid() {
		return root
	}
	p.Network = p.Network.Masked()
	if !covers(p.Network, p.Subnets) {
		p.Subnets = []netip.Prefix{p.Network}
	}
	s.chosen = max(0, min(len(p.Subnets)-1, s.chosen))
	root.Children(func() {
		s.bits(c, p)
		s.bar(c, p)
		s.table(c, root, p)
	})
	return root
}

// covers reports whether subnets, in order, cover network with no gaps.
func covers(network netip.Prefix, subnets []netip.Prefix) bool {
	if len(subnets) == 0 {
		return false
	}
	next := network.Addr()
	for i, sub := range subnets {
		if !sub.IsValid() || sub.Addr() != next || sub.Bits() < network.Bits() || !network.Contains(sub.Addr()) {
			return false
		}
		last := lastAddr(sub)
		if i == len(subnets)-1 {
			return last == lastAddr(network)
		}
		next = last.Next()
	}
	return false
}

// bits builds the bits of the chosen subnet's address, in bytes: the
// network's, the subnet's and the hosts'.
func (s *State) bits(c *ui.Context, p *Plan) {
	t := c.Theme()
	sub := p.Subnets[s.chosen]
	raw := sub.Addr().AsSlice()
	cell := t.Space(4)
	ui.Row(c).Wrap().Gap(t.Space(2)).Children(func() {
		for i, b := range raw {
			ui.Row(c.Key(i)).Gap(1).Children(func() {
				for k := range 8 {
					bit := i*8 + k
					on := b&(0x80>>k) != 0
					fill, text := t.Surface, t.TextMuted
					switch {
					case bit < p.Network.Bits():
						fill, text = t.Text.Alpha(0.14), t.Text
					case bit < sub.Bits():
						fill, text = t.Accent, t.AccentText
					}
					label := "0"
					if on {
						label = "1"
					}
					ui.Text(c.Key(k), label).Size(cell, cell+t.Space(1)).TextAlign(ui.Center).Radius(2).
						Background(fill).TextColor(text).FontSize(t.FontSize - 1).Font("monospace")
				}
			})
		}
	})
	ui.Row(c).Gap(t.Space(4)).Children(func() {
		key := func(fill ui.Color, label string) {
			ui.Row(c.Key(label)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(3), t.Space(3)).Radius(2).Background(fill).Border(1, t.Border)
				ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			})
		}
		key(t.Text.Alpha(0.14), fmt.Sprintf("Network, %d bits", p.Network.Bits()))
		key(t.Accent, fmt.Sprintf("Subnet, %d bits", sub.Bits()-p.Network.Bits()))
		key(t.Surface, fmt.Sprintf("Hosts, %d bits", sub.Addr().BitLen()-sub.Bits()))
	})
}

// bar builds the network's addresses as a bar, a subnet as wide as its
// share, and the ranges taken over it.
func (s *State) bar(c *ui.Context, p *Plan) {
	t := c.Theme()
	bar := ui.Box(c).Height(t.Space(9)).Radius(t.Radius).Clip().Cursor(ui.CursorPointer).Label("Subnets")
	share := func(q netip.Prefix) float64 { return math.Exp2(float64(p.Network.Bits() - q.Bits())) }
	at := func(a netip.Addr) float64 { return offset(p.Network, a) }
	if bar.Clicked() {
		if x, _, ok := bar.PointerPosition(); ok {
			f, start := float64(x/bar.Bounds().W), 0.0
			for i, sub := range p.Subnets {
				if start += share(sub); f < start {
					s.chosen = i
					s.list.ScrollIntoView(i)
					break
				}
			}
		}
	}
	chosen := s.chosen
	bar.Draw(func(pt *ui.Painter, r ui.Rect) {
		start := 0.0
		for i, sub := range p.Subnets {
			w := share(sub)
			x := r.X + r.W*float32(start)
			fill := t.Accent.Alpha(0.18 + 0.1*float32(i%2))
			if i == chosen {
				fill = t.Accent
			}
			pt.Fill(ui.Rect{X: x, Y: r.Y, W: r.W * float32(w), H: r.H}, fill, 0)
			if i > 0 {
				pt.Fill(ui.Rect{X: x, Y: r.Y, W: 1, H: r.H}, t.Background, 0)
			}
			start += w
		}
		for _, taken := range p.Taken {
			if !taken.Overlaps(p.Network) {
				continue
			}
			unit := math.Exp2(float64(p.Network.Bits() - p.Network.Addr().BitLen()))
			from := max(0, at(taken.Masked().Addr()))
			to := min(1, at(lastAddr(taken))+unit)
			band := ui.Rect{X: r.X + r.W*float32(from), Y: r.Y + r.H - 4, W: max(2, r.W*float32(to-from)), H: 4}
			pt.Fill(band, t.Danger, 0)
		}
	})
	ui.Row(c).Children(func() {
		ui.Text(c, p.Network.Addr().String()).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Font("monospace")
		ui.Spacer(c)
		ui.Text(c, lastAddr(p.Network).String()).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Font("monospace")
	})
}

// offset returns how far into network a is, from 0 at its first address
// to 1 past its last.
func offset(network netip.Prefix, a netip.Addr) float64 {
	n := new(big.Int).SetBytes(a.AsSlice())
	n.Sub(n, new(big.Int).SetBytes(network.Addr().AsSlice()))
	f, _ := new(big.Float).SetInt(n).Float64()
	return f / math.Exp2(float64(a.BitLen()-network.Bits()))
}

func (s *State) table(c *ui.Context, root ui.Element, p *Plan) {
	t := c.Theme()
	v4 := p.Network.Addr().Is4()
	cols := []ui.TableColumn{{Title: "Subnet", Width: 230}, {Title: "Range"}, {Title: "Hosts", Width: 140, Align: ui.End}, {Title: "", Width: 150, Fixed: true}}
	if v4 {
		cols = slices.Insert(cols, 1, ui.TableColumn{Title: "Mask", Width: 130})
	}
	s.list.Selected = &s.chosen
	subnets := p.Subnets
	// The rows' buttons change the subnets once the table is built, which
	// reads them till then.
	var change func()
	ui.Table(c, &s.list, cols, len(subnets), func(row, col int) {
		sub := subnets[row]
		if !v4 && col >= 1 {
			col++
		}
		switch col {
		case 0:
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, sub.String()).Font("monospace").SingleLine()
				if overlaps(sub, p.Taken) {
					ui.Text(c, "Overlaps").FontSize(t.FontSize - 2).TextColor(t.Danger).Tooltip("Overlaps a range taken elsewhere")
				}
			})
		case 1:
			ui.Text(c, mask(sub)).Font("monospace").SingleLine()
		case 2:
			ui.Text(c, sub.Masked().Addr().String()+" – "+lastAddr(sub).String()).Font("monospace").SingleLine().Ellipsis("…")
		case 3:
			ui.Text(c, hosts(sub)).FontFeatures("tnum").SingleLine()
		case 4:
			ui.Row(c).Gap(t.Space(1.5)).Children(func() {
				if ui.Button(c, "Divide").Disabled(sub.Bits() >= sub.Addr().BitLen()).Clicked() {
					change = func() {
						a, b := halves(sub)
						p.Subnets = slices.Replace(p.Subnets, row, row+1, a, b)
						if s.chosen > row {
							s.chosen++
						}
					}
				}
				joinable, sibling := false, 0
				if b := sub.Bits() - 1; b >= p.Network.Bits() {
					sibling = row + 1
					if sub.Addr().AsSlice()[b/8]&(0x80>>(b%8)) != 0 {
						sibling = row - 1
					}
					joinable = sibling >= 0 && sibling < len(subnets) && subnets[sibling].Bits() == sub.Bits()
				}
				if ui.Button(c, "Join").Disabled(!joinable).Clicked() {
					change = func() {
						lo := min(row, sibling)
						p.Subnets = slices.Replace(p.Subnets, lo, lo+2, parent(sub))
						if s.chosen > lo {
							s.chosen--
						}
					}
				}
			})
		}
	}).Grow(1).Label("Subnets")
	if change != nil {
		change()
		root.SetChanged()
	}
}

func overlaps(sub netip.Prefix, taken []netip.Prefix) bool {
	for _, q := range taken {
		if q.Overlaps(sub) {
			return true
		}
	}
	return false
}

// halves returns the two halves of p.
func halves(p netip.Prefix) (netip.Prefix, netip.Prefix) {
	a := netip.PrefixFrom(p.Masked().Addr(), p.Bits()+1)
	raw := a.Addr().AsSlice()
	raw[p.Bits()/8] |= 0x80 >> (p.Bits() % 8)
	b, _ := netip.AddrFromSlice(raw)
	return a, netip.PrefixFrom(b, p.Bits()+1)
}

func parent(p netip.Prefix) netip.Prefix {
	q, _ := p.Addr().Prefix(p.Bits() - 1)
	return q
}

// lastAddr returns the last address of p: all its host bits set.
func lastAddr(p netip.Prefix) netip.Addr {
	raw := p.Masked().Addr().AsSlice()
	for bit := p.Bits(); bit < len(raw)*8; bit++ {
		raw[bit/8] |= 0x80 >> (bit % 8)
	}
	a, _ := netip.AddrFromSlice(raw)
	return a
}

// mask writes the netmask of an IPv4 subnet.
func mask(p netip.Prefix) string {
	raw := make([]byte, 4)
	for bit := range p.Bits() {
		raw[bit/8] |= 0x80 >> (bit % 8)
	}
	a, _ := netip.AddrFromSlice(raw)
	return a.String()
}

// hosts writes how many hosts p holds: its addresses less the network's
// and the broadcast address in IPv4, all of them in IPv6, as a power of
// two past what fits.
func hosts(p netip.Prefix) string {
	free := p.Addr().BitLen() - p.Bits()
	if free > 48 {
		return "2^" + strconv.Itoa(free) + " addresses"
	}
	n := int64(1) << free
	if p.Addr().Is4() && free >= 2 {
		n -= 2
	}
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Package certview shows a chain of X.509 certificates, as browsers and
// Keychain Access show a site's: whether the chain verifies against the
// system's roots, and why not when it does not; the certificates from the
// leaf to the root, each its subject's name and its issuer's, the root the
// system found added at the end; and the details of the one chosen with a
// click or Up and Down: its subject's and issuer's names, the time it is
// valid, as a bar of the time gone and the days left, in the warning color
// for its last 30 days and the danger color once expired or before it is
// valid, its other names, its public key, signature and usages, and its
// SHA-256 and SHA-1 fingerprints, which a button copies.
//
//	conn, err := tls.Dial("tcp", "github.com:443", &tls.Config{InsecureSkipVerify: true})
//	...
//	app.chain = conn.ConnectionState().PeerCertificates
//	...
//	certview.View(c, &app.cert, app.chain).Grow(1)
package certview

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// nearExpiry is how long before it expires a certificate shows a warning.
const nearExpiry = 30 * 24 * time.Hour

// State is what a view keeps: the certificate chosen, and how the chain
// verified. Keep it in the app's state, one for each view.
type State struct {
	chosen int
	list   ui.ListState
	chain  []*x509.Certificate
	result chan verdict
	// verdict is nil while the chain is being verified.
	verdict *verdict
}

// verdict is how a chain verified: the error, or the root the system
// found when the chain does not end with it.
type verdict struct {
	root *x509.Certificate
	err  error
}

// View creates the view of chain, the leaf first, as a server sends it or
// a PEM file holds it: the verdict across the top, and below it the chain
// on the left and the details of the certificate chosen on the right,
// which scroll. Give it a size, or grow it in its parent. It verifies
// chain in the background against the system's roots for any usage,
// without a host name, as a new chain comes.
func View(c *ui.Context, s *State, chain []*x509.Certificate) ui.Element {
	t := c.Theme()
	if !slices.Equal(s.chain, chain) {
		*s = State{chain: slices.Clone(chain), result: make(chan verdict, 1)}
		if len(chain) > 0 {
			go verify(c.Services(), chain, s.result)
		}
	}
	select {
	case v := <-s.result:
		s.verdict = &v
	default:
	}
	certs := chain
	if s.verdict != nil && s.verdict.root != nil {
		certs = append(slices.Clip(chain), s.verdict.root)
	}
	s.chosen = max(0, min(s.chosen, len(certs)-1))
	s.list.Selected = &s.chosen
	s.list.Label = func(i int) string { return subject(certs[i]) }
	now := c.Now()
	view := ui.Column(c).MinHeight(0)
	view.Children(func() {
		if len(certs) == 0 {
			ui.Text(c, "No certificates").TextColor(t.TextMuted).Padding(t.Space(4))
			return
		}
		banner(c, s.verdict)
		ui.Divider(c)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			ui.List(c, &s.list, len(certs), func(i int) {
				entry(c, certs[i], i, len(certs), i == s.chosen, now, s.verdict != nil && certs[i] == s.verdict.root)
			}).Width(t.Space(64)).Shrink(0).Padding(t.Space(2))
			ui.Divider(c)
			ui.Scroll(c).Grow(1).Children(func() {
				details(c, certs[s.chosen], now)
			})
		})
	})
	return view
}

// verify verifies chain and sends how it did on result, then asks for a
// frame.
func verify(services ui.Services, chain []*x509.Certificate, result chan<- verdict) {
	pool := x509.NewCertPool()
	for _, cert := range chain[1:] {
		pool.AddCert(cert)
	}
	chains, err := chain[0].Verify(x509.VerifyOptions{
		Intermediates: pool,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	var v verdict
	switch {
	case err != nil:
		v.err = err
	case len(chains) > 0:
		if root := chains[0][len(chains[0])-1]; !slices.ContainsFunc(chain, root.Equal) {
			v.root = root
		}
	}
	result <- v
	services.Invalidate()
}

// banner shows how the chain verified, v, or that it is being verified
// while v is nil.
func banner(c *ui.Context, v *verdict) {
	t := c.Theme()
	row := ui.Row(c).MinHeight(t.Space(10)).Gap(t.Space(2.5)).AlignItems(ui.Center).Padding(t.Space(2.5), t.Space(4))
	row.Children(func() {
		if v == nil {
			ui.Spinner(c).Label("Verifying")
			ui.Text(c, "Verifying against the system roots…").TextColor(t.TextMuted)
			return
		}
		color, title, detail := t.Success, "Trusted", "Verifies against the system roots"
		if v.err != nil {
			color, title, detail = t.Danger, "Not trusted", reason(v.err)
		}
		row.Background(color.Alpha(0.08))
		ui.Box(c).Size(t.Space(4.5), t.Space(4.5)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, color, r.W/2)
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
			if v.err != nil {
				p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.75, t.Background)
				p.StrokePolyline([][2]float32{{cx - k, cy + k}, {cx + k, cy - k}}, 1.75, t.Background)
				return
			}
			p.StrokePolyline([][2]float32{{cx - k*1.1, cy}, {cx - k*0.3, cy + k*0.8}, {cx + k*1.2, cy - k*0.8}}, 1.75, t.Background)
		})
		ui.Text(c, title).Bold().TextColor(color).NoWrap()
		ui.Text(c, detail).TextColor(t.TextMuted).Grow(1).Shrink(1)
	})
}

// reason returns err as a sentence, without its package's prefix.
func reason(err error) string {
	s := strings.TrimPrefix(err.Error(), "x509: ")
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// entry builds the row of cert, the i-th of n: a dot on the line joining
// the chain, in the color of its validity, its subject's name, and its
// issuer's, or that it is self-signed, or a root the system added.
func entry(c *ui.Context, cert *x509.Certificate, i, n int, chosen bool, now time.Time, system bool) {
	t := c.Theme()
	text, muted := t.Text, t.TextMuted
	dot, _ := status(t, cert, now)
	if chosen {
		text, muted, dot = t.AccentText, t.AccentText.Alpha(0.8), t.AccentText
	}
	issuer := issuedBy(cert)
	if system {
		issuer += " · system root"
	}
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Stretch).Padding(0, t.Space(2)).Children(func() {
		ui.Box(c).Width(t.Space(3)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, d := r.X+r.W/2, r.Y+t.Space(4.25), t.Space(2.5)
			line := muted.Alpha(0.4)
			if i > 0 {
				p.Line(cx, r.Y, cx, cy-d/2-t.Space(0.75), 1, line)
			}
			if i < n-1 {
				p.Line(cx, cy+d/2+t.Space(0.75), cx, r.Y+r.H, 1, line)
			}
			p.Fill(ui.Rect{X: cx - d/2, Y: cy - d/2, W: d, H: d}, dot, d/2)
		})
		ui.Column(c).Gap(t.Space(0.5)).Grow(1).MinWidth(0).Padding(t.Space(2), 0).Children(func() {
			ui.Text(c, subject(cert)).TextColor(text).FontWeight(600).SingleLine()
			ui.Text(c, issuer).FontSize(t.FontSize - 1).TextColor(muted).SingleLine()
		})
	})
}

// details builds the sections of cert.
func details(c *ui.Context, cert *x509.Certificate, now time.Time) {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	date := func(d time.Time) string {
		return i18n.FormatDate(locale, d.Local(), i18n.DateOptions{Date: i18n.MediumStyle, Time: i18n.ShortStyle, HourCycle: c.Region().HourCycle})
	}
	ui.Column(c).Padding(t.Space(5), t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, subject(cert)).FontSize(t.FontSize + 5).Bold().Selectable()
			ui.Text(c, issuedBy(cert)).TextColor(t.TextMuted)
		})
		section(c, "Validity", func() {
			row(c, "Not before", date(cert.NotBefore))
			row(c, "Not after", date(cert.NotAfter))
			color, words := status(t, cert, now)
			span := cert.NotAfter.Sub(cert.NotBefore)
			gone := float32(0)
			if span > 0 {
				gone = float32(max(0, min(1, now.Sub(cert.NotBefore).Seconds()/span.Seconds())))
			}
			field(c, "", func() {
				ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Height(t.Space(1.5)).Grow(1).MaxWidth(t.Space(60)).Draw(func(p *ui.Painter, r ui.Rect) {
						p.Fill(r, t.Text.Alpha(0.1), r.H/2)
						if gone > 0 {
							p.Fill(ui.Rect{X: r.X, Y: r.Y, W: max(r.H, r.W*gone), H: r.H}, color, r.H/2)
						}
					})
					ui.Text(c, words).FontSize(t.FontSize - 1).FontWeight(600).TextColor(color).NoWrap()
				})
			})
		})
		section(c, "Subject Name", func() { names(c, cert.Subject) })
		section(c, "Issuer Name", func() {
			names(c, cert.Issuer)
			mono(c, "Serial number", spaced(cert.SerialNumber.Bytes()), false)
		})
		var ips, uris []string
		for _, ip := range cert.IPAddresses {
			ips = append(ips, ip.String())
		}
		for _, u := range cert.URIs {
			uris = append(uris, u.String())
		}
		if len(cert.DNSNames)+len(ips)+len(cert.EmailAddresses)+len(uris) > 0 {
			section(c, "Subject Alternative Names", func() {
				row(c, "DNS names", cert.DNSNames...)
				row(c, "IP addresses", ips...)
				row(c, "Email addresses", cert.EmailAddresses...)
				row(c, "URIs", uris...)
			})
		}
		section(c, "Public Key", func() {
			algorithm, size := key(cert)
			row(c, "Algorithm", algorithm)
			row(c, "Size", size)
		})
		section(c, "Signature", func() {
			row(c, "Algorithm", cert.SignatureAlgorithm.String())
		})
		usages, extended := keyUsages(cert)
		if len(usages)+len(extended) > 0 {
			section(c, "Usage", func() {
				row(c, "Key usage", strings.Join(usages, ", "))
				row(c, "Extended key usage", strings.Join(extended, ", "))
			})
		}
		section(c, "Fingerprints", func() {
			sum256, sum1 := sha256.Sum256(cert.Raw), sha1.Sum(cert.Raw)
			mono(c, "SHA-256", spaced(sum256[:]), true)
			mono(c, "SHA-1", spaced(sum1[:]), true)
		})
	})
}

// section builds a section titled title of the rows build builds.
func section(c *ui.Context, title string, build func()) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Text(c, title).FontSize(t.FontSize-1).Bold().Padding(0, 0, t.Space(1), 0)
		build()
	})
}

// field builds a row of the label label, muted, in a column as wide in
// every row, and of what build builds beside it.
func field(c *ui.Context, label string, build func()) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
		ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Width(t.Space(32)).Shrink(0)
		ui.Column(c).Gap(t.Space(1)).Grow(1).Basis(0).MinWidth(0).Children(build)
	})
}

// row builds a row of label and values, one under the other, which can be
// selected, or none when there are none.
func row(c *ui.Context, label string, values ...string) {
	values = slices.DeleteFunc(values, func(v string) bool { return v == "" })
	if len(values) == 0 {
		return
	}
	field(c, label, func() {
		for _, v := range values {
			ui.Text(c, v).FontSize(c.Theme().FontSize - 1).Selectable()
		}
	})
}

// mono builds a row of label and value in a monospaced font, with a
// button copying value if copyable.
func mono(c *ui.Context, label, value string, copyable bool) {
	t := c.Theme()
	field(c, label, func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
			ui.Text(c, value).Font("monospace").FontSize(t.FontSize - 1).Selectable().Grow(1).Basis(0).MinWidth(0)
			if copyable {
				copyButton(c, label, strings.ReplaceAll(value, "\n", " "))
			}
		})
	})
}

// copied is how long a copy button says it copied.
const copied = 1500 * time.Millisecond

// copyButton creates a button copying text, which says Copied for a
// while, named Copy and what of.
func copyButton(c *ui.Context, of, text string) {
	t := c.Theme()
	b := ui.ButtonBase(c).Width(t.Space(15)).Padding(t.Space(0.5), 0).Radius(t.Space(1)).Border(1, t.Border).
		Shrink(0).Label("Copy " + of)
	at := ui.Local(b, "certview.copied", func() time.Time { return time.Time{} })
	if b.Clicked() {
		c.WriteClipboard(text)
		*at = c.Now()
	}
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	label, color := "Copy", t.Text
	if left := copied - c.Now().Sub(*at); left > 0 {
		c.After(left)
		label, color = "Copied", t.Success
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(t.FontSize - 2).TextColor(color).NoWrap()
	})
}

// status returns the color of where now is in cert's validity, and its
// words, as "12 days left".
func status(t *ui.Theme, cert *x509.Certificate, now time.Time) (ui.Color, string) {
	days := func(d time.Duration) string {
		switch n := int(d.Hours() / 24); n {
		case 0:
			return "less than a day"
		case 1:
			return "1 day"
		default:
			return strconv.Itoa(n) + " days"
		}
	}
	switch left := cert.NotAfter.Sub(now); {
	case now.Before(cert.NotBefore):
		return t.Danger, "Not yet valid, starts in " + days(cert.NotBefore.Sub(now))
	case left < 0:
		return t.Danger, "Expired " + days(-left) + " ago"
	case left < nearExpiry:
		return t.Warning, days(left) + " left"
	default:
		return t.Success, days(left) + " left"
	}
}

// attributes are the names of the attributes of names, by their OIDs.
var attributes = map[string]string{
	"2.5.4.3":                  "Common name",
	"2.5.4.5":                  "Serial number",
	"2.5.4.6":                  "Country",
	"2.5.4.7":                  "Locality",
	"2.5.4.8":                  "State or province",
	"2.5.4.9":                  "Street address",
	"2.5.4.10":                 "Organization",
	"2.5.4.11":                 "Organizational unit",
	"2.5.4.15":                 "Business category",
	"2.5.4.17":                 "Postal code",
	"1.2.840.113549.1.9.1":     "Email address",
	"1.3.6.1.4.1.311.60.2.1.1": "Jurisdiction locality",
	"1.3.6.1.4.1.311.60.2.1.2": "Jurisdiction state",
	"1.3.6.1.4.1.311.60.2.1.3": "Jurisdiction country",
}

// names builds a row for each attribute of n, in the order of the
// certificate.
func names(c *ui.Context, n pkix.Name) {
	for _, a := range n.Names {
		label, ok := attributes[a.Type.String()]
		if !ok {
			label = a.Type.String()
		}
		row(c, label, fmt.Sprint(a.Value))
	}
}

// name returns what n is called: its common name, else its first
// organization, else all of it.
func name(n pkix.Name) string {
	switch {
	case n.CommonName != "":
		return n.CommonName
	case len(n.Organization) > 0:
		return n.Organization[0]
	}
	return n.String()
}

// subject returns what cert's subject is called, or its first DNS name
// when the subject is empty.
func subject(cert *x509.Certificate) string {
	if s := name(cert.Subject); s != "" {
		return s
	}
	if len(cert.DNSNames) > 0 {
		return cert.DNSNames[0]
	}
	return "Unnamed certificate"
}

// issuedBy returns who issued cert, "Issued by E6", or that it is
// self-signed.
func issuedBy(cert *x509.Certificate) string {
	if string(cert.RawIssuer) == string(cert.RawSubject) {
		return "Self-signed"
	}
	return "Issued by " + name(cert.Issuer)
}

// key returns the algorithm of cert's public key and its size.
func key(cert *x509.Certificate) (algorithm, size string) {
	switch k := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return "RSA", strconv.Itoa(k.N.BitLen()) + " bits"
	case *ecdsa.PublicKey:
		params := k.Curve.Params()
		return "ECDSA", strconv.Itoa(params.BitSize) + " bits, curve " + params.Name
	case ed25519.PublicKey:
		return "Ed25519", "256 bits"
	}
	return cert.PublicKeyAlgorithm.String(), ""
}

var (
	usageNames = []string{
		"Digital signature", "Content commitment", "Key encipherment", "Data encipherment",
		"Key agreement", "Certificate signing", "CRL signing", "Encipher only", "Decipher only",
	}
	extendedNames = map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageAny:                            "Any",
		x509.ExtKeyUsageServerAuth:                     "TLS server authentication",
		x509.ExtKeyUsageClientAuth:                     "TLS client authentication",
		x509.ExtKeyUsageCodeSigning:                    "Code signing",
		x509.ExtKeyUsageEmailProtection:                "Email protection",
		x509.ExtKeyUsageIPSECEndSystem:                 "IPsec end system",
		x509.ExtKeyUsageIPSECTunnel:                    "IPsec tunnel",
		x509.ExtKeyUsageIPSECUser:                      "IPsec user",
		x509.ExtKeyUsageTimeStamping:                   "Time stamping",
		x509.ExtKeyUsageOCSPSigning:                    "OCSP signing",
		x509.ExtKeyUsageMicrosoftServerGatedCrypto:     "Microsoft server gated crypto",
		x509.ExtKeyUsageNetscapeServerGatedCrypto:      "Netscape server gated crypto",
		x509.ExtKeyUsageMicrosoftCommercialCodeSigning: "Microsoft commercial code signing",
		x509.ExtKeyUsageMicrosoftKernelCodeSigning:     "Microsoft kernel code signing",
	}
)

// keyUsages returns the names of cert's key usages and of its extended
// key usages, those Go does not know by their OIDs.
func keyUsages(cert *x509.Certificate) (usages, extended []string) {
	for i, n := range usageNames {
		if cert.KeyUsage&(1<<i) != 0 {
			usages = append(usages, n)
		}
	}
	for _, u := range cert.ExtKeyUsage {
		extended = append(extended, extendedNames[u])
	}
	for _, oid := range cert.UnknownExtKeyUsage {
		extended = append(extended, oid.String())
	}
	return usages, extended
}

// spaced returns b in hex, a pair of digits a byte, apart, and in two
// even lines when longer than 16 bytes.
func spaced(b []byte) string {
	if len(b) == 0 {
		return "00"
	}
	line := len(b)
	if line > 16 {
		line = (line + 1) / 2
	}
	var s strings.Builder
	for i, x := range b {
		switch {
		case i == 0:
		case i%line == 0:
			s.WriteByte('\n')
		default:
			s.WriteByte(' ')
		}
		fmt.Fprintf(&s, "%02X", x)
	}
	return s.String()
}

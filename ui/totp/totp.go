// Package totp lists one-time codes, as authenticator apps such as
// 1Password and Google Authenticator do: a row for each account with its
// issuer and name, the current code in two groups, and a ring running
// down until the next. In the last five seconds the ring turns red and the
// next code shows beneath. A click on a row, or Return with it focused,
// copies its code, and the row says so for a moment. A field above
// filters the accounts by issuer or name.
//
//	acct, err := totp.ParseURI("otpauth://totp/GitHub:ada?secret=JBSWY3DPEHPK3PXP&issuer=GitHub")
//	totp.List(c, &app.codes, app.accounts)
//
// Codes are RFC 6238's, from the time on the clock.
package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Account is a secret shared with a site, and how codes are made from it.
type Account struct {
	Issuer, Name string
	Secret       []byte
	// Digits is how long a code is, 6 when zero.
	Digits int
	// Period is how long a code lasts, 30 seconds when zero.
	Period time.Duration
	// Algorithm is the HMAC's hash: "SHA1" when empty, "SHA256" or
	// "SHA512".
	Algorithm string
}

// ParseURI returns the account of an otpauth:// URI, as a QR code to set
// up two-factor sign-in holds.
func ParseURI(uri string) (Account, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return Account{}, err
	}
	if u.Scheme != "otpauth" || u.Host != "totp" {
		return Account{}, fmt.Errorf("totp: %s://%s is not otpauth://totp", u.Scheme, u.Host)
	}
	var a Account
	label := strings.TrimPrefix(u.Path, "/")
	if issuer, name, ok := strings.Cut(label, ":"); ok {
		a.Issuer, a.Name = issuer, strings.TrimSpace(name)
	} else {
		a.Name = label
	}
	q := u.Query()
	if s := q.Get("issuer"); s != "" {
		a.Issuer = s
	}
	if a.Secret, err = DecodeSecret(q.Get("secret")); err != nil {
		return Account{}, err
	}
	if s := q.Get("digits"); s != "" {
		if a.Digits, err = strconv.Atoi(s); err != nil || a.Digits < 6 || a.Digits > 10 {
			return Account{}, fmt.Errorf("totp: digits %q", s)
		}
	}
	if s := q.Get("period"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return Account{}, fmt.Errorf("totp: period %q", s)
		}
		a.Period = time.Duration(n) * time.Second
	}
	a.Algorithm = strings.ToUpper(q.Get("algorithm"))
	if newHash(a.Algorithm) == nil {
		return Account{}, fmt.Errorf("totp: algorithm %q", a.Algorithm)
	}
	return a, nil
}

// DecodeSecret decodes a secret written in base32, as sites show it to
// type in: case, spaces and padding don't matter.
func DecodeSecret(s string) ([]byte, error) {
	s = strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "=", "").Replace(s))
	if s == "" {
		return nil, errors.New("totp: no secret")
	}
	b, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("totp: secret: %w", err)
	}
	return b, nil
}

func newHash(alg string) func() hash.Hash {
	switch alg {
	case "", "SHA1":
		return sha1.New
	case "SHA256":
		return sha256.New
	case "SHA512":
		return sha512.New
	}
	return nil
}

func (a Account) period() time.Duration {
	if a.Period <= 0 {
		return 30 * time.Second
	}
	return a.Period
}

// Code returns a's code at time at.
func (a Account) Code(at time.Time) string {
	digits := a.Digits
	if digits <= 0 {
		digits = 6
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(at.Unix()/int64(a.period()/time.Second)))
	h := newHash(a.Algorithm)
	if h == nil {
		h = sha1.New
	}
	mac := hmac.New(h, a.Secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	// RFC 4226's dynamic truncation: the last nibble picks where four
	// bytes are read.
	off := sum[len(sum)-1] & 0x0f
	n := uint64(binary.BigEndian.Uint32(sum[off:]) & 0x7fffffff)
	mod := uint64(1)
	for range digits {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, n%mod)
}

// Remaining returns how long a's code at time at lasts.
func (a Account) Remaining(at time.Time) time.Duration {
	p := a.period()
	return p - time.Duration(at.UnixNano()%int64(p))
}

// group splits a code in two, the longer half first.
func group(code string) string {
	h := (len(code) + 1) / 2
	return code[:h] + " " + code[h:]
}

// State is what a list keeps: its filter, and which code was copied when.
// Keep it in the app's state, one for each list.
type State struct {
	query    string
	copied   int
	copiedAt time.Time
}

const (
	soon      = 5 * time.Second
	copyShows = 1500 * time.Millisecond
)

// List creates the list of accounts' codes, which it copies to the
// clipboard when asked.
func List(c *ui.Context, s *State, accounts []Account) ui.Element {
	t := c.Theme()
	now := c.Now()
	q := strings.ToLower(strings.TrimSpace(s.query))
	list := ui.Column(c).Gap(t.Space(3)).Role(ui.RoleGroup).Label("One-time codes")
	list.Children(func() {
		ui.SearchField(c, &s.query).Placeholder("Search accounts")
		rows := ui.Column(c).Background(t.Surface).Border(1, t.Border).Radius(t.Space(2.5)).Dividers(1, t.Border).Clip()
		rows.Children(func() {
			shown := 0
			for i, a := range accounts {
				if q != "" && !strings.Contains(strings.ToLower(a.Issuer+" "+a.Name), q) {
					continue
				}
				shown++
				row(c.Key(i), s, i, a, now)
			}
			if shown == 0 {
				ui.Text(c, "No accounts match").TextColor(t.TextMuted).Padding(t.Space(5)).TextAlign(ui.Center)
			}
		})
	})
	next := time.Second - time.Duration(now.Nanosecond())
	if left := copyShows - now.Sub(s.copiedAt); left > 0 {
		next = min(next, left)
	}
	c.After(next)
	return list
}

func row(c *ui.Context, s *State, i int, a Account, now time.Time) {
	t := c.Theme()
	code := a.Code(now)
	left := a.Remaining(now)
	label := a.Issuer
	if label == "" {
		label = a.Name
	}
	e := ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(3), t.Space(4)).Focusable().
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label + " " + a.Name).Value(code)
	if e.Clicked() || e.Shortcut(0, ui.KeyEnter) || e.Shortcut(ui.Cmd, ui.KeyC) {
		c.WriteClipboard(code)
		s.copied, s.copiedAt = i, now
	}
	if e.Hovered() {
		e.Background(t.Text.Alpha(0.04))
	}
	copied := s.copied == i && now.Sub(s.copiedAt) < copyShows
	e.Children(func() {
		ui.Avatar(c, label, nil).Size(t.Space(9), t.Space(9))
		ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, label).FontWeight(600).SingleLine()
			if a.Issuer != "" && a.Name != "" {
				ui.Text(c, a.Name).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
			}
		})
		ui.Column(c).AlignItems(ui.End).Gap(t.Space(0.5)).Children(func() {
			col := t.Text
			if left <= soon {
				col = t.Danger
			}
			ui.Text(c, group(code)).Font("monospace").FontSize(t.FontSize + 8).FontWeight(500).
				FontFeatures("tnum").LetterSpacing(1).TextColor(col)
			switch {
			case copied:
				ui.Text(c, "Copied").FontSize(t.FontSize - 2).TextColor(t.Accent).FontWeight(600)
			case left <= soon:
				ui.Text(c, "Next "+group(a.Code(now.Add(left)))).FontSize(t.FontSize - 2).TextColor(t.TextMuted).
					Font("monospace").FontFeatures("tnum")
			default:
				// Holds the line, so that rows keep their height as it fills.
				ui.Text(c, " ").FontSize(t.FontSize - 2)
			}
		})
		ring(c, left, a.period())
	})
}

// ring creates the countdown of a code lasting period, left of it to go.
func ring(c *ui.Context, left, period time.Duration) {
	t := c.Theme()
	size := t.Space(7)
	secs := int((left + time.Second - 1) / time.Second)
	ui.Box(c).Size(size, size).Label(strconv.Itoa(secs) + " seconds left").Draw(func(p *ui.Painter, r ui.Rect) {
		frac := float32(left) / float32(period)
		col := t.Accent
		if left <= soon {
			col = t.Danger
		}
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2-2
		var track ui.Path
		track.Circle(cx, cy, rad)
		p.StrokePath(&track, 3, t.Border)
		// The arc starts at twelve o'clock and runs clockwise, shrinking
		// toward it.
		var arc ui.Path
		arc.MoveTo(cx, cy-rad)
		for k := 1; k <= 48; k++ {
			a := float64(k) / 48 * 2 * math.Pi
			arc.LineTo(cx+rad*float32(math.Sin(a)), cy-rad*float32(math.Cos(a)))
		}
		p.StrokePath(arc.Trim(0, frac).Cap(ui.RoundCap), 3, col)
		span := ui.Span{Text: strconv.Itoa(secs), Size: t.FontSize - 3, Color: t.TextMuted, Features: "tnum"}
		w, h := p.MeasureText(0, span)
		p.RichText(cx-w/2, cy-h/2, 0, span)
	})
}

// Certview shows the certificates a server sends, or a PEM file holds:
// whether they verify against the system's roots, the chain from the leaf
// up, and the details of the one chosen. It connects to github.com:443
// without an argument.
//
//	go run ./examples/certview [host[:port] | file.pem]
package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"net"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/certview"
)

// app is the state the window shows: the chain, and the view's state.
type app struct {
	chain []*x509.Certificate
	cert  certview.State
}

func (a *app) view(c *ui.Context) {
	ui.Column(c).Fill().Background(c.Theme().Background).Children(func() {
		certview.View(c, &a.cert, a.chain).Grow(1)
	})
}

// dial returns the certificates the server at addr sends, verified or
// not, so that the view can say why they do not verify.
func dial(addr string) ([]*x509.Certificate, error) {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "443")
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.ConnectionState().PeerCertificates, nil
}

// read returns the certificates of the PEM file at path.
func read(path string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var chain []*x509.Certificate
	for {
		var block *pem.Block
		if block, data = pem.Decode(data); block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		chain = append(chain, cert)
	}
	if len(chain) == 0 {
		return nil, errors.New(path + ": no certificates")
	}
	return chain, nil
}

func main() {
	source := "github.com:443"
	if len(os.Args) > 1 {
		source = os.Args[1]
	}
	load := dial
	if _, err := os.Stat(source); err == nil {
		load = read
	}
	chain, err := load(source)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{chain: chain}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   source,
			Width:   920,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

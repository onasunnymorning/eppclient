package main

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"

	epp "github.com/onasunnymorning/eppclient"
	"github.com/wsxiaoys/terminal/color"
)

var (
	cfg  *Config   // set by session()
	conn *epp.Conn // open connection, if any
)

// session loads the profile and opens a logged-in connection on first use.
// Commands call it only once their arguments are valid, so --help, version
// and usage errors never touch the network.
func session() *epp.Conn {
	if conn != nil {
		return conn
	}
	c, err := loadConfig(opts.profile)
	if err != nil {
		fatalf("loading credentials for profile %q: %v", opts.profile, err)
	}
	cfg = c
	conn = connect(cfg)
	return conn
}

func closeSession() {
	if conn == nil {
		return
	}
	logger := epp.DebugLogger
	epp.DebugLogger = nil
	conn.Close()
	epp.DebugLogger = logger
	conn = nil
}

func connect(cfg *Config) *epp.Conn {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}

	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		host = cfg.Addr
	}
	tlsCfg.ServerName = host

	if cfg.CACert != "" {
		ca, err := os.ReadFile(cfg.CACert)
		fatalif(err)
		tlsCfg.RootCAs = x509.NewCertPool()
		tlsCfg.RootCAs.AppendCertsFromPEM(ca)
	}

	if cfg.Cert != "" && cfg.Key != "" {
		crt, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		fatalif(err)
		tlsCfg.Certificates = append(tlsCfg.Certificates, crt)
	}

	if !cfg.TLS {
		tlsCfg = nil
	}

	color.Fprintf(os.Stderr, "Connecting to %s\n", cfg.Addr)
	nc, err := net.Dial("tcp", cfg.Addr)
	fatalif(err)

	if tlsCfg != nil {
		color.Fprintf(os.Stderr, "Establishing TLS connection\n")
		tc := tls.Client(nc, tlsCfg)
		fatalif(tc.Handshake())
		nc = tc
	}

	// Keep the greeting and login (with the password) out of the XML log.
	logger := epp.DebugLogger
	epp.DebugLogger = nil
	defer func() { epp.DebugLogger = logger }()

	color.Fprintf(os.Stderr, "Performing EPP handshake\n")
	c, err := epp.NewConn(nc)
	fatalif(err)

	color.Fprintf(os.Stderr, "Logging in as %s...\n", cfg.User)
	_, err = c.Login(cfg.User, cfg.Password, "")
	fatalif(err)

	return c
}

// usesContacts reports whether the registry works with contact objects.
// A greeting without any objURIs is treated as "yes" to stay on the safe side.
func usesContacts(c *epp.Conn) bool {
	return len(c.Greeting.Objects) == 0 || c.Greeting.SupportsObject(epp.ObjContact)
}

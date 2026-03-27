package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	epp "github.com/onasunnymorning/eppclient"
)

// Config holds the EPP connection configuration.
type Config struct {
	Addr     string
	User     string
	Password string
	TLS      bool
	Cert     string
	Key      string
	CACert   string
}

// connectEPP establishes a connection to the EPP server using the provided configuration.
func connectEPP(cfg *Config) (*epp.Conn, error) {
	// TODO: Connection pooling could be implemented here or managed globally to avoid
	// establishing a new connection per request. For now, we connect on demand.
	
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}

	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		host = cfg.Addr
	}
	tlsCfg.ServerName = host

	if cfg.CACert != "" {
		ca, err := ioutil.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		tlsCfg.RootCAs = x509.NewCertPool()
		tlsCfg.RootCAs.AppendCertsFromPEM(ca)
	}

	if cfg.Cert != "" && cfg.Key != "" {
		crt, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert/key: %w", err)
		}
		tlsCfg.Certificates = append(tlsCfg.Certificates, crt)
	}

	if !cfg.TLS {
		tlsCfg = nil
	}

	conn, err := net.Dial("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	if tlsCfg != nil {
		tc := tls.Client(conn, tlsCfg)
		err = tc.Handshake()
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("tls handshake failed: %w", err)
		}
		conn = tc
	}

	// Disable debug logging for bot to avoid cluttering stdout unless needed
	logger := epp.DebugLogger
	epp.DebugLogger = nil
	
	c, err := epp.NewConn(conn)
	epp.DebugLogger = logger
	
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("epp handshake failed: %w", err)
	}

	logger = epp.DebugLogger
	epp.DebugLogger = nil
	_, err = c.Login(cfg.User, cfg.Password, "")
	epp.DebugLogger = logger
	
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("epp login failed: %w", err)
	}

	return c, nil
}

// checkDomain runs the EPP check command against a given domain list.
func checkDomain(cfg *Config, domains []string) (*epp.DomainCheckResponse, error) {
	c, err := connectEPP(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	// By requesting a fee check for 1 period, the server will return currency and pricing info.
	extData := map[string]string{
		"fee:period": "1",
	}

	dc, err := c.CheckDomainExtensions(domains, extData)
	if err != nil {
		return nil, fmt.Errorf("check domain failed: %w", err)
	}

	return dc, nil
}

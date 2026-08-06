package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	sslCheckLogTimeFormat  = "20060102-150405"
	sslCheckTimeout        = 20 * time.Second
	publicIPLookupTimeout  = 5 * time.Second
	publicIPResponseLimit  = 1 << 20
	publicIPLookupEndpoint = "https://stat.ripe.net/data/whats-my-ip/data.json"
)

type dialContextFunc func(context.Context, string, string) (net.Conn, error)
type publicIPLookupFunc func(context.Context) (string, error)

type sslCheckResult struct {
	Profile                string
	Address                string
	PublicIPAddress        string
	PublicIPLookupErr      error
	ServerName             string
	ClientCertificate      string
	ClientKey              string
	CACertificate          string
	CredentialsTLSEnabled  bool
	StartedAt              time.Time
	EndedAt                time.Time
	TCPAttempted           bool
	TCPConnected           bool
	TLSHandshakeAttempted  bool
	TLSHandshakeSuccessful bool
	ConnectionState        *tls.ConnectionState
	FailureStage           string
	Err                    error
}

func isSSLCheck(cmd string, args []string) bool {
	return cmd == "check" && len(args) > 0 && args[0] == "ssl"
}

func sslCheckLogFilename(timestamp time.Time) string {
	return "test-ssl-" + timestamp.Format(sslCheckLogTimeFormat) + ".log"
}

func runSSLCheck(profile, directory string, startedAt time.Time) (string, error) {
	cfg, err := loadConfig(profile, false)
	if err != nil {
		result := sslCheckResult{
			Profile:      profile,
			StartedAt:    startedAt,
			EndedAt:      time.Now(),
			FailureStage: "credentials",
			Err:          err,
		}
		path, logErr := saveSSLCheckLog(directory, result)
		return path, errors.Join(err, logErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sslCheckTimeout)
	defer cancel()

	dialer := &net.Dialer{}
	httpClient := &http.Client{Timeout: publicIPLookupTimeout}
	lookupPublicIP := func(ctx context.Context) (string, error) {
		return lookupPublicIPAddress(ctx, httpClient, publicIPLookupEndpoint)
	}
	result := performSSLCheck(ctx, profile, cfg, startedAt, dialer.DialContext, lookupPublicIP)
	path, logErr := saveSSLCheckLog(directory, result)
	return path, errors.Join(result.Err, logErr)
}

func performSSLCheck(ctx context.Context, profile string, cfg *Config, startedAt time.Time, dial dialContextFunc, lookupPublicIP publicIPLookupFunc) (result sslCheckResult) {
	result = sslCheckResult{
		Profile:               profile,
		Address:               cfg.Addr,
		ClientCertificate:     cfg.Cert,
		ClientKey:             cfg.Key,
		CACertificate:         cfg.CACert,
		CredentialsTLSEnabled: cfg.TLS,
		StartedAt:             startedAt,
	}
	defer func() {
		result.EndedAt = time.Now()
	}()

	lookupCtx, cancelLookup := context.WithTimeout(ctx, publicIPLookupTimeout)
	result.PublicIPAddress, result.PublicIPLookupErr = lookupPublicIP(lookupCtx)
	cancelLookup()

	result.TCPAttempted = true
	conn, err := dial(ctx, "tcp", cfg.Addr)
	if err != nil {
		result.FailureStage = "TCP connection"
		result.Err = err
		return result
	}
	result.TCPConnected = true
	defer conn.Close()

	tlsConfig, serverName, err := sslCheckTLSConfig(cfg)
	result.ServerName = serverName
	if err != nil {
		result.FailureStage = "TLS configuration"
		result.Err = err
		return result
	}

	tlsConn := tls.Client(conn, tlsConfig)
	result.TLSHandshakeAttempted = true
	err = tlsConn.HandshakeContext(ctx)
	state := tlsConn.ConnectionState()
	result.ConnectionState = &state
	if err != nil {
		result.FailureStage = "TLS handshake"
		result.Err = err
		return result
	}

	result.TLSHandshakeSuccessful = true
	return result
}

func lookupPublicIPAddress(ctx context.Context, client *http.Client, endpoint string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create RIPEstat public IP request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("request public IP from RIPEstat: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("RIPEstat public IP request returned HTTP %s", response.Status)
	}

	var payload struct {
		Data struct {
			IP string `json:"ip"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, publicIPResponseLimit))
	if err := decoder.Decode(&payload); err != nil {
		return "", fmt.Errorf("decode RIPEstat public IP response: %w", err)
	}

	ipAddress := strings.TrimSpace(payload.Data.IP)
	if net.ParseIP(ipAddress) == nil {
		return "", fmt.Errorf("RIPEstat returned an invalid public IP address")
	}
	return ipAddress, nil
}

func sslCheckTLSConfig(cfg *Config) (*tls.Config, string, error) {
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return nil, "", fmt.Errorf("invalid addr %q: expected host:port: %w", cfg.Addr, err)
	}
	if cfg.Cert == "" {
		return nil, host, errors.New("cert not found in credentials profile")
	}
	if cfg.Key == "" {
		return nil, host, errors.New("key not found in credentials profile")
	}

	clientCertificate, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
	if err != nil {
		return nil, host, fmt.Errorf("load client certificate and key: %w", err)
	}

	var roots *x509.CertPool
	if cfg.CACert != "" {
		caPEM, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, host, fmt.Errorf("read CA certificate: %w", err)
		}
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEM) {
			return nil, host, fmt.Errorf("CA certificate %q does not contain a valid PEM certificate", cfg.CACert)
		}
	} else {
		roots, err = x509.SystemCertPool()
		if err != nil {
			return nil, host, fmt.Errorf("load system CA certificates: %w", err)
		}
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCertificate},
		RootCAs:      roots,
		ServerName:   host,
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS12,
	}, host, nil
}

func saveSSLCheckLog(directory string, result sslCheckResult) (string, error) {
	filename := sslCheckLogFilename(result.StartedAt)
	path := filepath.Join(directory, filename)
	for suffix := 2; ; suffix++ {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			extension := filepath.Ext(filename)
			base := filename[:len(filename)-len(extension)]
			path = filepath.Join(directory, fmt.Sprintf("%s-%d%s", base, suffix, extension))
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create SSL test log: %w", err)
		}

		if _, err := file.Write(formatSSLCheckLog(result)); err != nil {
			_ = file.Close()
			return "", fmt.Errorf("write SSL test log: %w", err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close SSL test log: %w", err)
		}
		return path, nil
	}
}

func formatSSLCheckLog(result sslCheckResult) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "EPP-ID TCP Connection Test")
	fmt.Fprintln(&output, "================================")
	fmt.Fprintf(&output, "Profile: %s\n", result.Profile)
	fmt.Fprintf(&output, "Address: %s\n", valueOrUnavailable(result.Address))
	fmt.Fprintf(&output, "My IP Address: %s\n", valueOrUnavailable(result.PublicIPAddress))
	if result.PublicIPLookupErr != nil {
		fmt.Fprintf(&output, "Public IP Lookup Error: %v\n", result.PublicIPLookupErr)
	}
	fmt.Fprintln(&output)
	if result.TCPConnected {
		fmt.Fprintln(&output, "Result: SUCCESS")
	} else {
		fmt.Fprintln(&output, "Result: FAILED")
		writeSSLCheckFailure(&output, result)
		writeProcessTiming(&output, result)
		return output.Bytes()
	}

	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "EPP-ID SSL/TLS Connection Test")
	fmt.Fprintln(&output, "================================")
	fmt.Fprintf(&output, "Profile: %s\n", result.Profile)
	fmt.Fprintf(&output, "Address: %s\n", valueOrUnavailable(result.Address))
	fmt.Fprintf(&output, "Server Name: %s\n", valueOrUnavailable(result.ServerName))
	fmt.Fprintln(&output, "Requested TLS Version: TLS 1.2")
	fmt.Fprintf(&output, "Credentials TLS Enabled: %t\n", result.CredentialsTLSEnabled)
	fmt.Fprintf(&output, "Client Certificate: %s\n", valueOrUnavailable(result.ClientCertificate))
	fmt.Fprintf(&output, "Client Key: %s\n", valueOrUnavailable(result.ClientKey))
	if result.CACertificate == "" {
		fmt.Fprintln(&output, "CA Certificate: system certificate pool")
	} else {
		fmt.Fprintf(&output, "CA Certificate: %s\n", result.CACertificate)
	}
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "TLS Handshake: %s\n", connectionStatus(result.TLSHandshakeAttempted, result.TLSHandshakeSuccessful))

	if state := result.ConnectionState; state != nil {
		if state.Version != 0 {
			fmt.Fprintf(&output, "Negotiated TLS Version: %s\n", tlsVersionName(state.Version))
		}
		if state.CipherSuite != 0 {
			fmt.Fprintf(&output, "Cipher Suite: %s\n", tls.CipherSuiteName(state.CipherSuite))
		}
		fmt.Fprintf(&output, "Verified Chains: %d\n", len(state.VerifiedChains))
		fmt.Fprintf(&output, "Server Certificates: %d\n", len(state.PeerCertificates))
		for index, certificate := range state.PeerCertificates {
			fingerprint := sha256.Sum256(certificate.Raw)
			fmt.Fprintf(&output, "\nServer Certificate %d\n", index+1)
			fmt.Fprintf(&output, "  Subject: %s\n", certificate.Subject.String())
			fmt.Fprintf(&output, "  Issuer: %s\n", certificate.Issuer.String())
			fmt.Fprintf(&output, "  Serial Number: %s\n", certificate.SerialNumber.String())
			fmt.Fprintf(&output, "  Valid From: %s\n", certificate.NotBefore.Format(processTimeFormat))
			fmt.Fprintf(&output, "  Valid Until: %s\n", certificate.NotAfter.Format(processTimeFormat))
			fmt.Fprintf(&output, "  DNS Names: %s\n", strings.Join(certificate.DNSNames, ", "))
			fmt.Fprintf(&output, "  SHA-256 Fingerprint: %X\n", fingerprint)
		}
	}

	fmt.Fprintln(&output)
	if result.TLSHandshakeSuccessful {
		fmt.Fprintln(&output, "Result: SUCCESS")
	} else {
		fmt.Fprintln(&output, "Result: FAILED")
		writeSSLCheckFailure(&output, result)
	}
	writeProcessTiming(&output, result)
	return output.Bytes()
}

func writeSSLCheckFailure(output *bytes.Buffer, result sslCheckResult) {
	if result.FailureStage != "" {
		fmt.Fprintf(output, "Failure Stage: %s\n", result.FailureStage)
	}
	if result.Err != nil {
		fmt.Fprintf(output, "Error: %v\n", result.Err)
	}
}

func writeProcessTiming(output *bytes.Buffer, result sslCheckResult) {
	fmt.Fprintln(output)
	fmt.Fprintf(output, "Start Time: %s\n", result.StartedAt.Format(processTimeFormat))
	fmt.Fprintf(output, "End Time: %s\n", result.EndedAt.Format(processTimeFormat))
	fmt.Fprintf(output, "Duration Process: %s\n", result.EndedAt.Sub(result.StartedAt).Round(time.Millisecond))
}

func connectionStatus(attempted, successful bool) string {
	if !attempted {
		return "NOT ATTEMPTED"
	}
	if successful {
		return "SUCCESS"
	}
	return "FAILED"
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", version)
	}
}

func valueOrUnavailable(value string) string {
	if value == "" {
		return "not available"
	}
	return value
}

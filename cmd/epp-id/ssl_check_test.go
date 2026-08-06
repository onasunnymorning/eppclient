package main

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSSLCheckLogFilename(t *testing.T) {
	timestamp := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.FixedZone("WIB", 7*60*60))

	if got, want := sslCheckLogFilename(timestamp), "test-ssl-20260806-170405.log"; got != want {
		t.Fatalf("sslCheckLogFilename() = %q, want %q", got, want)
	}
}

func TestPerformSSLCheckWithTLS12AndClientCertificate(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{
		ClientAuth: tls.RequireAnyClientCert,
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
	}
	server.StartTLS()
	defer server.Close()

	caPath := filepath.Join(t.TempDir(), "server-ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(caPath, caPEM, 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Addr:   server.Listener.Addr().String(),
		TLS:    true,
		Cert:   "../../config/.epp/ssl/certificate.pem",
		Key:    "../../config/.epp/ssl/certificate.key",
		CACert: caPath,
	}
	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialer := &net.Dialer{}

	result := performSSLCheck(ctx, "default", cfg, startedAt, dialer.DialContext)

	if result.Err != nil {
		t.Fatalf("performSSLCheck() error = %v", result.Err)
	}
	if !result.TCPConnected || !result.TLSHandshakeSuccessful {
		t.Fatalf("performSSLCheck() result = %+v", result)
	}
	if result.ConnectionState == nil {
		t.Fatal("performSSLCheck() did not record TLS connection state")
	}
	if got, want := result.ConnectionState.Version, uint16(tls.VersionTLS12); got != want {
		t.Fatalf("TLS version = %#x, want %#x", got, want)
	}
	if len(result.ConnectionState.VerifiedChains) == 0 {
		t.Fatal("performSSLCheck() did not verify the server certificate")
	}
	if !result.EndedAt.After(result.StartedAt) {
		t.Fatalf("end time %v is not after start time %v", result.EndedAt, result.StartedAt)
	}

	logContent := string(formatSSLCheckLog(result))
	for _, expected := range []string{
		"TCP Connection: SUCCESS",
		"TLS Handshake: SUCCESS",
		"Negotiated TLS Version: TLS 1.2",
		"Server Certificates: 1",
		"Result: SUCCESS",
		"Duration Process:",
	} {
		if !strings.Contains(logContent, expected) {
			t.Fatalf("SSL log does not contain %q:\n%s", expected, logContent)
		}
	}
}

func TestPerformSSLCheckRecordsConfigurationFailureWithoutDialing(t *testing.T) {
	cfg := &Config{Addr: "epp.registry.example.id:700", TLS: true}
	dialCalled := false
	dial := func(context.Context, string, string) (net.Conn, error) {
		dialCalled = true
		return nil, nil
	}

	result := performSSLCheck(context.Background(), "default", cfg, time.Now(), dial)

	if result.Err == nil || !strings.Contains(result.Err.Error(), "cert not found") {
		t.Fatalf("performSSLCheck() error = %v", result.Err)
	}
	if result.FailureStage != "TLS configuration" {
		t.Fatalf("failure stage = %q, want TLS configuration", result.FailureStage)
	}
	if result.TCPAttempted || dialCalled {
		t.Fatal("performSSLCheck() dialed despite invalid TLS configuration")
	}
	if got := string(formatSSLCheckLog(result)); !strings.Contains(got, "TCP Connection: NOT ATTEMPTED") || !strings.Contains(got, "Result: FAILED") {
		t.Fatalf("failure log = %q", got)
	}
}

func TestSaveSSLCheckLogDoesNotOverwriteSameSecond(t *testing.T) {
	directory := t.TempDir()
	startedAt := time.Date(2026, time.August, 6, 17, 4, 5, 0, time.UTC)
	result := sslCheckResult{
		Profile:                "default",
		Address:                "epp.registry.example.id:700",
		StartedAt:              startedAt,
		EndedAt:                startedAt.Add(time.Second),
		TCPAttempted:           true,
		TCPConnected:           true,
		TLSHandshakeAttempted:  true,
		TLSHandshakeSuccessful: true,
	}

	firstPath, err := saveSSLCheckLog(directory, result)
	if err != nil {
		t.Fatal(err)
	}
	secondPath, err := saveSSLCheckLog(directory, result)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := filepath.Base(firstPath), "test-ssl-20260806-170405.log"; got != want {
		t.Fatalf("first filename = %q, want %q", got, want)
	}
	if got, want := filepath.Base(secondPath), "test-ssl-20260806-170405-2.log"; got != want {
		t.Fatalf("second filename = %q, want %q", got, want)
	}
	info, err := os.Stat(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0600); got != want {
		t.Fatalf("log permissions = %o, want %o", got, want)
	}
	data, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "Duration Process: 1s\n") {
		t.Fatalf("log footer = %q", data)
	}
}

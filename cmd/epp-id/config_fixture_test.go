package main

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"
)

func TestSampleCertificateChainAndPrivateKey(t *testing.T) {
	pair, err := tls.LoadX509KeyPair(
		"../../config/.epp-id/ssl/certificate.pem",
		"../../config/.epp-id/ssl/certificate.key",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pair.Certificate), 3; got != want {
		t.Fatalf("certificate chain contains %d certificates, want %d", got, want)
	}

	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	intermediate, err := x509.ParseCertificate(pair.Certificate[1])
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(pair.Certificate[2])
	if err != nil {
		t.Fatal(err)
	}

	intermediates := x509.NewCertPool()
	intermediates.AddCert(intermediate)
	roots := x509.NewCertPool()
	roots.AddCert(root)

	_, err = leaf.Verify(x509.VerifyOptions{
		Intermediates: intermediates,
		Roots:         roots,
		CurrentTime:   time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		t.Fatalf("verify sample certificate chain: %v", err)
	}
}

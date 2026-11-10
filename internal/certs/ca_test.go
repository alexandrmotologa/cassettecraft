package certs_test

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/cassettecraft/internal/certs"
)

func TestGenerateEphemeralCA(t *testing.T) {
	ca, err := certs.GenerateEphemeralCA()
	if err != nil {
		t.Fatalf("failed to generate ephemeral CA: %v", err)
	}

	if ca.RootCert == nil {
		t.Fatal("RootCert is nil")
	}

	if ca.RootCert.Subject.CommonName != "CassetteCraft Root CA" {
		t.Fatalf("expected CN 'CassetteCraft Root CA', got %q", ca.RootCert.Subject.CommonName)
	}

	// Issue leaf certificate for a domain
	leaf, err := ca.GetCertificate("api.stripe.com")
	if err != nil {
		t.Fatalf("failed to get leaf certificate: %v", err)
	}

	if len(leaf.Certificate) < 2 {
		t.Fatalf("expected cert chain to include leaf and root, got %d", len(leaf.Certificate))
	}

	parsedLeaf, err := x509.ParseCertificate(leaf.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse leaf certificate: %v", err)
	}

	if len(parsedLeaf.DNSNames) == 0 || parsedLeaf.DNSNames[0] != "api.stripe.com" {
		t.Fatalf("expected DNSName api.stripe.com, got %v", parsedLeaf.DNSNames)
	}

	// Verify caching works
	cachedLeaf, err := ca.GetCertificate("api.stripe.com")
	if err != nil {
		t.Fatalf("failed to get cached leaf: %v", err)
	}
	if leaf != cachedLeaf {
		t.Fatal("expected cached certificate pointer to match")
	}

	// Test IP address certificate
	ipLeaf, err := ca.GetCertificate("127.0.0.1")
	if err != nil {
		t.Fatalf("failed to get leaf for IP: %v", err)
	}
	parsedIPLeaf, err := x509.ParseCertificate(ipLeaf.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse IP leaf: %v", err)
	}
	if len(parsedIPLeaf.IPAddresses) == 0 || parsedIPLeaf.IPAddresses[0].String() != "127.0.0.1" {
		t.Fatalf("expected IP 127.0.0.1, got %v", parsedIPLeaf.IPAddresses)
	}
}

func TestPersistentCA(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cassettecraft-ca-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	ca1, err := certs.NewCA(tempDir)
	if err != nil {
		t.Fatalf("failed to create persistent CA: %v", err)
	}

	certFile := filepath.Join(tempDir, "ca.crt")
	keyFile := filepath.Join(tempDir, "ca.key")
	if _, err := os.Stat(certFile); err != nil {
		t.Fatalf("ca.crt not created: %v", err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		t.Fatalf("ca.key not created: %v", err)
	}

	// Load again from disk
	ca2, err := certs.NewCA(tempDir)
	if err != nil {
		t.Fatalf("failed to reload persistent CA: %v", err)
	}

	if ca1.FingerprintSHA256() != ca2.FingerprintSHA256() {
		t.Fatalf("expected fingerprints to match on reload, got %s != %s", ca1.FingerprintSHA256(), ca2.FingerprintSHA256())
	}
}

func TestTLSConfigHandshake(t *testing.T) {
	ca, err := certs.GenerateEphemeralCA()
	if err != nil {
		t.Fatal(err)
	}

	tlsCfg := ca.GetTLSConfig()
	cert, err := tlsCfg.GetCertificate(&tls.ClientHelloInfo{ServerName: "api.github.com"})
	if err != nil {
		t.Fatalf("failed to get cert from tls.Config: %v", err)
	}
	if cert == nil {
		t.Fatal("returned nil certificate")
	}
}

package certs

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CA manages the root certificate authority and dynamic leaf certificate issuance.
type CA struct {
	RootCert   *x509.Certificate
	RootKey    crypto.Signer
	CertPEM    []byte
	KeyPEM     []byte
	cacheMu    sync.RWMutex
	certCache  map[string]*tls.Certificate
	leafKey    *rsa.PrivateKey
}

// NewCA loads an existing CA from certDir, or generates a new one if not found.
func NewCA(certDir string) (*CA, error) {
	if certDir == "" {
		home, _ := os.UserHomeDir()
		certDir = filepath.Join(home, ".cassettecraft", "ca")
	}

	certPath := filepath.Join(certDir, "ca.crt")
	keyPath := filepath.Join(certDir, "ca.key")

	if fileExists(certPath) && fileExists(keyPath) {
		ca, err := loadCA(certPath, keyPath)
		if err == nil {
			return ca, nil
		}
		// If loading failed, regenerate
	}

	return GenerateCA(certDir)
}

// GenerateEphemeralCA creates an in-memory CA without saving to disk.
func GenerateEphemeralCA() (*CA, error) {
	return generateCAInternal("", false)
}

// GenerateCA creates a fresh root CA and saves it to certDir.
func GenerateCA(certDir string) (*CA, error) {
	return generateCAInternal(certDir, true)
}

func generateCAInternal(certDir string, persist bool) (*CA, error) {
	// Generate 2048-bit RSA key for the root CA
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate root key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now().Add(-24 * time.Hour)
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour) // 10 years validity

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"CassetteCraft Local Authority"},
			CommonName:    "CassetteCraft Root CA",
			Country:       []string{"US"},
			Province:      []string{"Testing"},
			Locality:      []string{"Local"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create root certificate: %w", err)
	}

	rootCert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated root certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rootKey)})

	if persist && certDir != "" {
		if err := os.MkdirAll(certDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create CA directory %q: %w", certDir, err)
		}
		if err := os.WriteFile(filepath.Join(certDir, "ca.crt"), certPEM, 0644); err != nil {
			return nil, fmt.Errorf("failed to write ca.crt: %w", err)
		}
		if err := os.WriteFile(filepath.Join(certDir, "ca.key"), keyPEM, 0600); err != nil {
			return nil, fmt.Errorf("failed to write ca.key: %w", err)
		}
	}

	// Pre-generate a shared leaf private key for all dynamic certificates to optimize handshake speed
	sharedLeafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate shared leaf key: %w", err)
	}

	return &CA{
		RootCert:  rootCert,
		RootKey:   rootKey,
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		certCache: make(map[string]*tls.Certificate),
		leafKey:   sharedLeafKey,
	}, nil
}

func loadCA(certPath, keyPath string) (*CA, error) {
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file %q: %w", certPath, err)
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %q: %w", keyPath, err)
	}

	certBlock, _ := pem.Decode(certBytes)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode cert PEM in %q", certPath)
	}
	rootCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode key PEM in %q", keyPath)
	}

	var rootKey crypto.Signer
	if key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err == nil {
		rootKey = key
	} else if key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes); err == nil {
		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("key in %q is not a crypto.Signer", keyPath)
		}
		rootKey = signer
	} else {
		return nil, fmt.Errorf("unsupported private key format in %q", keyPath)
	}

	sharedLeafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate shared leaf key: %w", err)
	}

	return &CA{
		RootCert:  rootCert,
		RootKey:   rootKey,
		CertPEM:   certBytes,
		KeyPEM:    keyBytes,
		certCache: make(map[string]*tls.Certificate),
		leafKey:   sharedLeafKey,
	}, nil
}

// GetCertificate dynamically signs and returns a TLS certificate for the given hostname or IP.
func (ca *CA) GetCertificate(host string) (*tls.Certificate, error) {
	// Strip port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	ca.cacheMu.RLock()
	cached, ok := ca.certCache[host]
	ca.cacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	ca.cacheMu.Lock()
	defer ca.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if cached, ok := ca.certCache[host]; ok {
		return cached, nil
	}

	// Generate leaf certificate
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate leaf serial: %w", err)
	}

	notBefore := time.Now().Add(-1 * time.Hour)
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1 year validity

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"CassetteCraft Intercepted Endpoint"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, ca.RootCert, &ca.leafKey.PublicKey, ca.RootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create leaf cert for host %q: %w", host, err)
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{derBytes, ca.RootCert.Raw},
		PrivateKey:  ca.leafKey,
	}

	ca.certCache[host] = cert
	return cert, nil
}

// GetTLSConfig returns a tls.Config configured for MITM interception using GetCertificate.
func (ca *CA) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			serverName := info.ServerName
			if serverName == "" {
				serverName = "localhost"
			}
			return ca.GetCertificate(serverName)
		},
		NextProtos: []string{"http/1.1"},
	}
}

// ExportCertToFile writes the root certificate in PEM format to a file.
func (ca *CA) ExportCertToFile(destPath string) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(destPath, ca.CertPEM, 0644)
}

// FingerprintSHA256 returns the SHA-256 fingerprint of the Root CA cert.
func (ca *CA) FingerprintSHA256() string {
	hash := sha256.Sum256(ca.RootCert.Raw)
	return fmt.Sprintf("%X", hash)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

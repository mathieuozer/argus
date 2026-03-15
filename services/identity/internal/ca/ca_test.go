package ca

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

func TestNewDevCA(t *testing.T) {
	t.Run("creates valid CA", func(t *testing.T) {
		devCA, err := NewDevCA()
		if err != nil {
			t.Fatalf("NewDevCA() returned error: %v", err)
		}

		if devCA == nil {
			t.Fatal("NewDevCA() returned nil")
		}

		// Parse and validate the CA cert PEM
		caPEM := devCA.CACertPEM()
		if len(caPEM) == 0 {
			t.Fatal("CACertPEM() returned empty bytes")
		}

		block, _ := pem.Decode(caPEM)
		if block == nil {
			t.Fatal("failed to decode CA cert PEM")
		}
		if block.Type != "CERTIFICATE" {
			t.Errorf("expected PEM type %q, got %q", "CERTIFICATE", block.Type)
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("failed to parse CA certificate: %v", err)
		}

		if !cert.IsCA {
			t.Error("expected CA cert to have IsCA=true")
		}
		if cert.Subject.CommonName != "Argus Dev CA" {
			t.Errorf("expected CN %q, got %q", "Argus Dev CA", cert.Subject.CommonName)
		}
		if len(cert.Subject.Organization) == 0 || cert.Subject.Organization[0] != "Argus" {
			t.Error("expected organization to be Argus")
		}
		if cert.BasicConstraintsValid != true {
			t.Error("expected BasicConstraintsValid to be true")
		}
		if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
			t.Error("expected KeyUsageCertSign to be set")
		}
		if cert.KeyUsage&x509.KeyUsageCRLSign == 0 {
			t.Error("expected KeyUsageCRLSign to be set")
		}

		// Verify the cert is not expired
		now := time.Now()
		if now.Before(cert.NotBefore) {
			t.Error("CA cert NotBefore is in the future")
		}
		if now.After(cert.NotAfter) {
			t.Error("CA cert is expired")
		}
	})
}

func TestIssueCert(t *testing.T) {
	devCA, err := NewDevCA()
	if err != nil {
		t.Fatalf("failed to create dev CA: %v", err)
	}

	t.Run("issued cert is valid", func(t *testing.T) {
		certPEM, keyPEM, err := devCA.IssueCert("test-agent.argus.local", 1*time.Hour)
		if err != nil {
			t.Fatalf("IssueCert() returned error: %v", err)
		}

		if len(certPEM) == 0 {
			t.Fatal("certPEM is empty")
		}
		if len(keyPEM) == 0 {
			t.Fatal("keyPEM is empty")
		}

		// Parse the issued certificate
		certBlock, _ := pem.Decode(certPEM)
		if certBlock == nil {
			t.Fatal("failed to decode cert PEM")
		}
		if certBlock.Type != "CERTIFICATE" {
			t.Errorf("expected cert PEM type %q, got %q", "CERTIFICATE", certBlock.Type)
		}

		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			t.Fatalf("failed to parse issued certificate: %v", err)
		}

		if cert.Subject.CommonName != "test-agent.argus.local" {
			t.Errorf("expected CN %q, got %q", "test-agent.argus.local", cert.Subject.CommonName)
		}
		if cert.IsCA {
			t.Error("issued cert should not be a CA")
		}

		// Verify TTL is approximately correct (within a few seconds)
		expectedExpiry := time.Now().Add(1 * time.Hour)
		if cert.NotAfter.Before(expectedExpiry.Add(-10 * time.Second)) {
			t.Error("cert expires too early")
		}
		if cert.NotAfter.After(expectedExpiry.Add(10 * time.Second)) {
			t.Error("cert expires too late")
		}

		// Verify extended key usage
		hasServerAuth := false
		hasClientAuth := false
		for _, usage := range cert.ExtKeyUsage {
			if usage == x509.ExtKeyUsageServerAuth {
				hasServerAuth = true
			}
			if usage == x509.ExtKeyUsageClientAuth {
				hasClientAuth = true
			}
		}
		if !hasServerAuth {
			t.Error("expected ExtKeyUsageServerAuth")
		}
		if !hasClientAuth {
			t.Error("expected ExtKeyUsageClientAuth")
		}

		// Parse the private key
		keyBlock, _ := pem.Decode(keyPEM)
		if keyBlock == nil {
			t.Fatal("failed to decode key PEM")
		}
		if keyBlock.Type != "EC PRIVATE KEY" {
			t.Errorf("expected key PEM type %q, got %q", "EC PRIVATE KEY", keyBlock.Type)
		}
	})

	t.Run("issued cert is signed by CA", func(t *testing.T) {
		certPEM, _, err := devCA.IssueCert("signed-agent.argus.local", 1*time.Hour)
		if err != nil {
			t.Fatalf("IssueCert() returned error: %v", err)
		}

		// Parse the issued cert
		certBlock, _ := pem.Decode(certPEM)
		if certBlock == nil {
			t.Fatal("failed to decode cert PEM")
		}
		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			t.Fatalf("failed to parse issued certificate: %v", err)
		}

		// Parse the CA cert
		caBlock, _ := pem.Decode(devCA.CACertPEM())
		if caBlock == nil {
			t.Fatal("failed to decode CA cert PEM")
		}
		caCert, err := x509.ParseCertificate(caBlock.Bytes)
		if err != nil {
			t.Fatalf("failed to parse CA certificate: %v", err)
		}

		// Verify the cert is signed by the CA
		roots := x509.NewCertPool()
		roots.AddCert(caCert)

		opts := x509.VerifyOptions{
			Roots:     roots,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		_, err = cert.Verify(opts)
		if err != nil {
			t.Errorf("cert verification against CA failed: %v", err)
		}
	})

	t.Run("each issued cert has unique serial number", func(t *testing.T) {
		certPEM1, _, err := devCA.IssueCert("agent-1.argus.local", 1*time.Hour)
		if err != nil {
			t.Fatalf("first IssueCert() error: %v", err)
		}

		certPEM2, _, err := devCA.IssueCert("agent-2.argus.local", 1*time.Hour)
		if err != nil {
			t.Fatalf("second IssueCert() error: %v", err)
		}

		block1, _ := pem.Decode(certPEM1)
		cert1, _ := x509.ParseCertificate(block1.Bytes)

		block2, _ := pem.Decode(certPEM2)
		cert2, _ := x509.ParseCertificate(block2.Bytes)

		if cert1.SerialNumber.Cmp(cert2.SerialNumber) == 0 {
			t.Error("issued certs should have different serial numbers")
		}
	})
}

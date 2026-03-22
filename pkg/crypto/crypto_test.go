package crypto

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	tests := []struct {
		name       string
		commonName string
		ttl        time.Duration
	}{
		{
			name:       "standard cert with 1 hour TTL",
			commonName: "argus-test.local",
			ttl:        1 * time.Hour,
		},
		{
			name:       "cert with long TTL",
			commonName: "argus-long.local",
			ttl:        365 * 24 * time.Hour,
		},
		{
			name:       "cert with short TTL",
			commonName: "argus-short.local",
			ttl:        5 * time.Minute,
		},
		{
			name:       "cert with SPIFFE-style CN",
			commonName: "spiffe://argus.example.com/tenant/t1/agent/a1",
			ttl:        1 * time.Hour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			certPEM, keyPEM, err := GenerateSelfSignedCert(tc.commonName, tc.ttl)
			if err != nil {
				t.Fatalf("GenerateSelfSignedCert() error = %v", err)
			}

			// Verify cert PEM is non-empty and decodable
			if len(certPEM) == 0 {
				t.Fatal("certPEM is empty")
			}
			if len(keyPEM) == 0 {
				t.Fatal("keyPEM is empty")
			}

			// Decode cert PEM
			certBlock, rest := pem.Decode(certPEM)
			if certBlock == nil {
				t.Fatal("failed to decode cert PEM block")
			}
			if certBlock.Type != "CERTIFICATE" {
				t.Errorf("cert PEM type = %q; want %q", certBlock.Type, "CERTIFICATE")
			}
			if len(rest) != 0 {
				t.Errorf("unexpected trailing data after cert PEM: %d bytes", len(rest))
			}

			// Decode key PEM
			keyBlock, rest := pem.Decode(keyPEM)
			if keyBlock == nil {
				t.Fatal("failed to decode key PEM block")
			}
			if keyBlock.Type != "EC PRIVATE KEY" {
				t.Errorf("key PEM type = %q; want %q", keyBlock.Type, "EC PRIVATE KEY")
			}
			if len(rest) != 0 {
				t.Errorf("unexpected trailing data after key PEM: %d bytes", len(rest))
			}

			// Parse the X.509 certificate
			cert, err := x509.ParseCertificate(certBlock.Bytes)
			if err != nil {
				t.Fatalf("x509.ParseCertificate() error = %v", err)
			}

			// Verify CN
			if cert.Subject.CommonName != tc.commonName {
				t.Errorf("CommonName = %q; want %q", cert.Subject.CommonName, tc.commonName)
			}

			// Verify expiry is approximately correct
			now := time.Now()
			if cert.NotBefore.After(now) {
				t.Errorf("NotBefore (%v) is after now (%v)", cert.NotBefore, now)
			}

			expectedExpiry := now.Add(tc.ttl)
			expiryDelta := cert.NotAfter.Sub(expectedExpiry)
			if expiryDelta < 0 {
				expiryDelta = -expiryDelta
			}
			// Allow up to 5 seconds of clock skew for test execution time
			if expiryDelta > 5*time.Second {
				t.Errorf("NotAfter = %v; expected approximately %v (delta: %v)", cert.NotAfter, expectedExpiry, expiryDelta)
			}

			// Verify key usage
			if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
				t.Error("cert missing KeyUsageDigitalSignature")
			}
			if cert.KeyUsage&x509.KeyUsageKeyEncipherment == 0 {
				t.Error("cert missing KeyUsageKeyEncipherment")
			}

			// Verify extended key usage
			foundServerAuth := false
			foundClientAuth := false
			for _, usage := range cert.ExtKeyUsage {
				if usage == x509.ExtKeyUsageServerAuth {
					foundServerAuth = true
				}
				if usage == x509.ExtKeyUsageClientAuth {
					foundClientAuth = true
				}
			}
			if !foundServerAuth {
				t.Error("cert missing ExtKeyUsageServerAuth")
			}
			if !foundClientAuth {
				t.Error("cert missing ExtKeyUsageClientAuth")
			}

			// Verify the cert is self-signed (Issuer == Subject)
			if cert.Issuer.CommonName != cert.Subject.CommonName {
				t.Errorf("Issuer.CN = %q; want %q (self-signed)", cert.Issuer.CommonName, cert.Subject.CommonName)
			}
		})
	}
}

func TestGenerateSelfSignedCertUniqueness(t *testing.T) {
	// Two certs generated with the same params should have different serial numbers
	cert1PEM, _, err := GenerateSelfSignedCert("test.local", 1*time.Hour)
	if err != nil {
		t.Fatalf("first GenerateSelfSignedCert() error = %v", err)
	}

	cert2PEM, _, err := GenerateSelfSignedCert("test.local", 1*time.Hour)
	if err != nil {
		t.Fatalf("second GenerateSelfSignedCert() error = %v", err)
	}

	block1, _ := pem.Decode(cert1PEM)
	block2, _ := pem.Decode(cert2PEM)

	c1, _ := x509.ParseCertificate(block1.Bytes)
	c2, _ := x509.ParseCertificate(block2.Bytes)

	if c1.SerialNumber.Cmp(c2.SerialNumber) == 0 {
		t.Error("two generated certs have the same serial number; expected unique serials")
	}
}

func TestHMACSignAndVerify(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  []byte
	}{
		{
			name: "simple message",
			data: []byte("hello argus"),
			key:  []byte("secret-key-123"),
		},
		{
			name: "empty data",
			data: []byte(""),
			key:  []byte("secret-key"),
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
			key:  []byte("binary-key"),
		},
		{
			name: "long message",
			data: bytes.Repeat([]byte("a"), 10000),
			key:  []byte("key-for-long-msg"),
		},
		{
			name: "unicode data",
			data: []byte("merhaba dunya"),
			key:  []byte("anahtar"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig := HMACSign(tc.data, tc.key)

			// Signature should not be empty
			if len(sig) == 0 {
				t.Fatal("HMACSign returned empty signature")
			}

			// SHA256 HMAC should always be 32 bytes
			if len(sig) != 32 {
				t.Errorf("signature length = %d; want 32", len(sig))
			}

			// Verify should succeed with same data and key
			if !HMACVerify(tc.data, sig, tc.key) {
				t.Error("HMACVerify returned false for valid signature")
			}
		})
	}
}

func TestHMACVerifyFailsWithWrongData(t *testing.T) {
	tests := []struct {
		name     string
		original []byte
		tampered []byte
		key      []byte
	}{
		{
			name:     "different message",
			original: []byte("correct data"),
			tampered: []byte("tampered data"),
			key:      []byte("shared-key"),
		},
		{
			name:     "extra byte",
			original: []byte("hello"),
			tampered: []byte("hello!"),
			key:      []byte("key"),
		},
		{
			name:     "empty vs non-empty",
			original: []byte("data"),
			tampered: []byte(""),
			key:      []byte("key"),
		},
		{
			name:     "single bit difference",
			original: []byte{0x00},
			tampered: []byte{0x01},
			key:      []byte("key"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig := HMACSign(tc.original, tc.key)
			if HMACVerify(tc.tampered, sig, tc.key) {
				t.Error("HMACVerify returned true for tampered data; want false")
			}
		})
	}
}

func TestHMACVerifyFailsWithWrongKey(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		signKey   []byte
		verifyKey []byte
	}{
		{
			name:      "different key",
			data:      []byte("some data"),
			signKey:   []byte("correct-key"),
			verifyKey: []byte("wrong-key"),
		},
		{
			name:      "empty verify key",
			data:      []byte("some data"),
			signKey:   []byte("correct-key"),
			verifyKey: []byte(""),
		},
		{
			name:      "key with extra byte",
			data:      []byte("some data"),
			signKey:   []byte("key"),
			verifyKey: []byte("key!"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig := HMACSign(tc.data, tc.signKey)
			if HMACVerify(tc.data, sig, tc.verifyKey) {
				t.Error("HMACVerify returned true for wrong key; want false")
			}
		})
	}
}

func TestHMACSignDeterministic(t *testing.T) {
	data := []byte("deterministic test")
	key := []byte("test-key")

	sig1 := HMACSign(data, key)
	sig2 := HMACSign(data, key)

	if !bytes.Equal(sig1, sig2) {
		t.Error("HMACSign is not deterministic; two calls with same input produced different results")
	}
}

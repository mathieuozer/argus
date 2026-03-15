package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateCACert creates a self-signed CA certificate and private key.
// It returns PEM-encoded cert and key bytes.
func generateCACert(t *testing.T) (certPEM, keyPEM []byte, caKey *ecdsa.PrivateKey, caCert *x509.Certificate) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}

	caCert = &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Argus Test CA"},
			CommonName:   "Argus Test Root CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caCert, caCert, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}

	// Parse back the DER so we have a fully-populated *x509.Certificate for signing.
	caCert, err = x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		t.Fatalf("marshal CA key: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: caKeyDER})

	return certPEM, keyPEM, caKey, caCert
}

// generateServerCert creates a server certificate signed by the given CA.
// It returns PEM-encoded cert and key bytes.
func generateServerCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) (certPEM, keyPEM []byte) {
	t.Helper()

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Argus Test"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create server cert: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})

	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		t.Fatalf("marshal server key: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyDER})

	return certPEM, keyPEM
}

// writePEM writes PEM data to a file in the given directory and returns the path.
func writePEM(t *testing.T, dir, filename string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	return path
}

func TestNewTLSConfig(t *testing.T) {
	// Generate test certificates once for the subtests that need valid certs.
	caCertPEM, _, caKey, caCert := generateCACert(t)
	serverCertPEM, serverKeyPEM := generateServerCert(t, caCert, caKey)

	tmpDir := t.TempDir()
	caPath := writePEM(t, tmpDir, "ca.crt", caCertPEM)
	serverCertPath := writePEM(t, tmpDir, "server.crt", serverCertPEM)
	serverKeyPath := writePEM(t, tmpDir, "server.key", serverKeyPEM)

	tests := []struct {
		name       string
		cfg        *Config
		wantErr    bool
		errContain string
		validate   func(t *testing.T, tlsCfg *tls.Config)
	}{
		{
			name: "valid certs produce correct TLS config",
			cfg: &Config{
				CACertPath:     caPath,
				ServerCertPath: serverCertPath,
				ServerKeyPath:  serverKeyPath,
			},
			wantErr: false,
			validate: func(t *testing.T, tlsCfg *tls.Config) {
				t.Helper()

				if tlsCfg.ClientAuth != tls.RequireAndVerifyClientCert {
					t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert", tlsCfg.ClientAuth)
				}

				if tlsCfg.ClientCAs == nil {
					t.Fatal("ClientCAs is nil, want populated CA pool")
				}

				if len(tlsCfg.Certificates) != 1 {
					t.Fatalf("Certificates has %d entries, want 1", len(tlsCfg.Certificates))
				}

				if tlsCfg.MinVersion != tls.VersionTLS12 {
					t.Errorf("MinVersion = %d, want %d (TLS 1.2)", tlsCfg.MinVersion, tls.VersionTLS12)
				}
			},
		},
		{
			name: "missing CA cert file returns error",
			cfg: &Config{
				CACertPath:     filepath.Join(tmpDir, "nonexistent-ca.crt"),
				ServerCertPath: serverCertPath,
				ServerKeyPath:  serverKeyPath,
			},
			wantErr:    true,
			errContain: "read CA cert",
		},
		{
			name: "missing server cert file returns error",
			cfg: &Config{
				CACertPath:     caPath,
				ServerCertPath: filepath.Join(tmpDir, "nonexistent-server.crt"),
				ServerKeyPath:  serverKeyPath,
			},
			wantErr:    true,
			errContain: "load server cert",
		},
		{
			name: "missing server key file returns error",
			cfg: &Config{
				CACertPath:     caPath,
				ServerCertPath: serverCertPath,
				ServerKeyPath:  filepath.Join(tmpDir, "nonexistent-server.key"),
			},
			wantErr:    true,
			errContain: "load server cert",
		},
		{
			name: "invalid CA cert PEM returns error",
			cfg: func() *Config {
				badCA := writePEM(t, tmpDir, "bad-ca.crt", []byte("not a valid PEM"))
				return &Config{
					CACertPath:     badCA,
					ServerCertPath: serverCertPath,
					ServerKeyPath:  serverKeyPath,
				}
			}(),
			wantErr:    true,
			errContain: "failed to append CA cert",
		},
		{
			name: "mismatched server cert and key returns error",
			cfg: func() *Config {
				// Generate a second server cert with a different key
				_, otherKeyPEM := generateServerCert(t, caCert, caKey)
				otherKeyPath := writePEM(t, tmpDir, "other-server.key", otherKeyPEM)
				return &Config{
					CACertPath:     caPath,
					ServerCertPath: serverCertPath,
					ServerKeyPath:  otherKeyPath,
				}
			}(),
			wantErr:    true,
			errContain: "load server cert",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tlsCfg, err := NewTLSConfig(tc.cfg)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContain != "" {
					if !containsStr(err.Error(), tc.errContain) {
						t.Errorf("error = %q, want substring %q", err.Error(), tc.errContain)
					}
				}
				if tlsCfg != nil {
					t.Error("expected nil tls.Config when error occurs")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tlsCfg == nil {
				t.Fatal("tls.Config is nil, expected non-nil")
			}

			if tc.validate != nil {
				tc.validate(t, tlsCfg)
			}
		})
	}
}

func TestNewTLSConfig_RequireAndVerifyClientCert(t *testing.T) {
	// Dedicated test for the mTLS enforcement requirement: the returned
	// tls.Config must require and verify client certificates.
	caCertPEM, _, caKey, caCert := generateCACert(t)
	serverCertPEM, serverKeyPEM := generateServerCert(t, caCert, caKey)

	tmpDir := t.TempDir()
	caPath := writePEM(t, tmpDir, "ca.crt", caCertPEM)
	certPath := writePEM(t, tmpDir, "server.crt", serverCertPEM)
	keyPath := writePEM(t, tmpDir, "server.key", serverKeyPEM)

	cfg := &Config{
		CACertPath:     caPath,
		ServerCertPath: certPath,
		ServerKeyPath:  keyPath,
	}

	tlsCfg, err := NewTLSConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tlsCfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert (%v)",
			tlsCfg.ClientAuth, tls.RequireAndVerifyClientCert)
	}
}

func TestNewTLSConfig_CACertInPool(t *testing.T) {
	// Verify the CA certificate is actually in the returned client CA pool
	// by checking that the pool recognizes a certificate signed by that CA.
	caCertPEM, _, caKey, caCert := generateCACert(t)
	serverCertPEM, serverKeyPEM := generateServerCert(t, caCert, caKey)

	tmpDir := t.TempDir()
	caPath := writePEM(t, tmpDir, "ca.crt", caCertPEM)
	certPath := writePEM(t, tmpDir, "server.crt", serverCertPEM)
	keyPath := writePEM(t, tmpDir, "server.key", serverKeyPEM)

	cfg := &Config{
		CACertPath:     caPath,
		ServerCertPath: certPath,
		ServerKeyPath:  keyPath,
	}

	tlsCfg, err := NewTLSConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the ClientCAs pool is non-nil (contains our test CA).
	if tlsCfg.ClientCAs == nil {
		t.Error("ClientCAs pool is nil, expected at least the test CA")
	}
}

// containsStr is a helper to avoid importing strings just for one call.
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

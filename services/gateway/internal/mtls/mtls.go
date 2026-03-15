package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Config holds mTLS configuration.
type Config struct {
	CACertPath     string
	ServerCertPath string
	ServerKeyPath  string
}

// NewTLSConfig creates a TLS configuration for mTLS termination.
func NewTLSConfig(cfg *Config) (*tls.Config, error) {
	caCert, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	serverCert, err := tls.LoadX509KeyPair(cfg.ServerCertPath, cfg.ServerKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load server cert: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// Package tls provides shared TLS configuration for consumoor sinks and
// sources.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// ErrDisabled is returned by Build when TLS is not enabled.
var ErrDisabled = errors.New("tls: not enabled")

// Config configures TLS for a connection. When Enabled is true with no
// other fields, it behaves the same as the previous bare tls: true setting.
type Config struct {
	// Enabled enables TLS for the connection.
	Enabled bool `yaml:"enabled"`
	// CAFile is the path to a PEM-encoded CA certificate file used to
	// verify the server's certificate.
	CAFile string `yaml:"caFile"`
	// CertFile is the path to a PEM-encoded client certificate file for
	// mutual TLS authentication.
	CertFile string `yaml:"certFile"`
	// KeyFile is the path to a PEM-encoded client private key file for
	// mutual TLS authentication.
	KeyFile string `yaml:"keyFile"`
	// InsecureSkipVerify disables server certificate verification.
	InsecureSkipVerify bool `yaml:"insecureSkipVerify"`
}

// Validate checks the TLS configuration for errors.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if (c.CertFile == "") != (c.KeyFile == "") {
		return errors.New("tls: certFile and keyFile must both be set or both be empty")
	}

	if c.CAFile != "" {
		if _, err := os.Stat(c.CAFile); err != nil {
			return fmt.Errorf("tls: caFile %q: %w", c.CAFile, err)
		}
	}

	if c.CertFile != "" {
		if _, err := os.Stat(c.CertFile); err != nil {
			return fmt.Errorf("tls: certFile %q: %w", c.CertFile, err)
		}
	}

	if c.KeyFile != "" {
		if _, err := os.Stat(c.KeyFile); err != nil {
			return fmt.Errorf("tls: keyFile %q: %w", c.KeyFile, err)
		}
	}

	return nil
}

// Build constructs a *tls.Config from the settings. Returns ErrDisabled
// when TLS is not enabled. The returned config always sets MinVersion to
// TLS 1.2.
func (c *Config) Build() (*tls.Config, error) {
	if !c.Enabled {
		return nil, ErrDisabled
	}

	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.InsecureSkipVerify, //nolint:gosec // user-controlled knob
	}

	if c.CAFile != "" {
		caPEM, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("tls: reading CA file: %w", err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf(
				"tls: CA file %q contains no valid certificates",
				c.CAFile,
			)
		}

		cfg.RootCAs = pool
	}

	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("tls: loading client certificate: %w", err)
		}

		cfg.Certificates = []tls.Certificate{cert}
	}

	return cfg, nil
}

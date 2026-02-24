package tls_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	xtls "github.com/ethpandaops/xatu/pkg/consumoor/tls"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     xtls.Config
		setup   func(t *testing.T) xtls.Config
		wantErr string
	}{
		{
			name: "disabled config is always valid",
			cfg:  xtls.Config{Enabled: false, CertFile: "missing.pem"},
		},
		{
			name:    "certFile without keyFile",
			cfg:     xtls.Config{Enabled: true, CertFile: "cert.pem"},
			wantErr: "certFile and keyFile must both be set",
		},
		{
			name:    "keyFile without certFile",
			cfg:     xtls.Config{Enabled: true, KeyFile: "key.pem"},
			wantErr: "certFile and keyFile must both be set",
		},
		{
			name:    "caFile does not exist",
			cfg:     xtls.Config{Enabled: true, CAFile: "/nonexistent/ca.pem"},
			wantErr: "caFile",
		},
		{
			name: "caFile exists",
			setup: func(t *testing.T) xtls.Config {
				t.Helper()
				ca, _, _ := generateTestCA(t)
				caPath := filepath.Join(t.TempDir(), "ca.pem")
				require.NoError(t, os.WriteFile(caPath, ca, 0o600))

				return xtls.Config{Enabled: true, CAFile: caPath}
			},
		},
		{
			name: "enabled with no extra fields is valid",
			cfg:  xtls.Config{Enabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			if tt.setup != nil {
				cfg = tt.setup(t)
			}

			err := cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_Build(t *testing.T) {
	t.Run("disabled returns ErrDisabled", func(t *testing.T) {
		cfg := xtls.Config{Enabled: false}
		tlsCfg, err := cfg.Build()
		require.ErrorIs(t, err, xtls.ErrDisabled)
		assert.Nil(t, tlsCfg)
	})

	t.Run("enabled with no extras returns minimal config", func(t *testing.T) {
		cfg := xtls.Config{Enabled: true}
		tlsCfg, err := cfg.Build()
		require.NoError(t, err)
		require.NotNil(t, tlsCfg)

		assert.Equal(t, uint16(0x0303), tlsCfg.MinVersion) // TLS 1.2
		assert.False(t, tlsCfg.InsecureSkipVerify)
		assert.Nil(t, tlsCfg.RootCAs)
		assert.Empty(t, tlsCfg.Certificates)
	})

	t.Run("insecure skip verify", func(t *testing.T) {
		cfg := xtls.Config{Enabled: true, InsecureSkipVerify: true}
		tlsCfg, err := cfg.Build()
		require.NoError(t, err)
		assert.True(t, tlsCfg.InsecureSkipVerify)
	})

	t.Run("custom CA file", func(t *testing.T) {
		caPEM, _, _ := generateTestCA(t)
		caPath := filepath.Join(t.TempDir(), "ca.pem")
		require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))

		cfg := xtls.Config{Enabled: true, CAFile: caPath}
		tlsCfg, err := cfg.Build()
		require.NoError(t, err)
		require.NotNil(t, tlsCfg.RootCAs)
	})

	t.Run("invalid CA file content", func(t *testing.T) {
		caPath := filepath.Join(t.TempDir(), "bad-ca.pem")
		require.NoError(t, os.WriteFile(caPath, []byte("not a cert"), 0o600))

		cfg := xtls.Config{Enabled: true, CAFile: caPath}
		_, err := cfg.Build()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid certificates")
	})

	t.Run("missing CA file", func(t *testing.T) {
		cfg := xtls.Config{Enabled: true, CAFile: "/nonexistent/ca.pem"}
		_, err := cfg.Build()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading CA file")
	})

	t.Run("client certificate", func(t *testing.T) {
		caPEM, caKey, caCert := generateTestCA(t)
		certPEM, keyPEM := generateTestCert(t, caCert, caKey)

		dir := t.TempDir()
		caPath := filepath.Join(dir, "ca.pem")
		certPath := filepath.Join(dir, "cert.pem")
		keyPath := filepath.Join(dir, "key.pem")

		require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))
		require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))
		require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))

		cfg := xtls.Config{
			Enabled:  true,
			CAFile:   caPath,
			CertFile: certPath,
			KeyFile:  keyPath,
		}
		tlsCfg, err := cfg.Build()
		require.NoError(t, err)
		require.NotNil(t, tlsCfg.RootCAs)
		assert.Len(t, tlsCfg.Certificates, 1)
	})

	t.Run("invalid client certificate", func(t *testing.T) {
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.pem")
		keyPath := filepath.Join(dir, "key.pem")

		require.NoError(t, os.WriteFile(certPath, []byte("bad"), 0o600))
		require.NoError(t, os.WriteFile(keyPath, []byte("bad"), 0o600))

		cfg := xtls.Config{
			Enabled:  true,
			CertFile: certPath,
			KeyFile:  keyPath,
		}
		_, err := cfg.Build()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading client certificate")
	})
}

// generateTestCA creates a self-signed CA certificate and returns the PEM
// bytes, private key, and parsed certificate.
func generateTestCA(t *testing.T) ([]byte, *ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-ca"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	parsed, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return certPEM, key, parsed
}

// generateTestCert creates a client certificate signed by the given CA.
func generateTestCert(
	t *testing.T,
	caCert *x509.Certificate,
	caKey *ecdsa.PrivateKey,
) (certPEM, keyPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

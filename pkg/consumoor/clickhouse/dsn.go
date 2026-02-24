package clickhouse

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go"

	xtls "github.com/ethpandaops/xatu/pkg/consumoor/tls"
)

const schemeClickHouseSecure = "clickhouses"

func parseChGoOptions(
	dsn string,
	dialTimeout, readTimeout time.Duration,
	tlsCfg *xtls.Config,
) (ch.Options, error) {
	if !strings.Contains(dsn, "://") {
		dsn = "clickhouse://" + dsn
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return ch.Options{}, err
	}

	switch u.Scheme {
	case "http", "https":
		return ch.Options{},
			fmt.Errorf("ch-go backend supports native tcp only, got DSN scheme %q", u.Scheme)
	case "clickhouse", "tcp", schemeClickHouseSecure:
	default:
		return ch.Options{},
			fmt.Errorf("unsupported DSN scheme %q for ch-go backend", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return ch.Options{}, fmt.Errorf("missing host in DSN")
	}

	if strings.Contains(host, ",") {
		return ch.Options{}, fmt.Errorf("multiple hosts are not supported by ch-go backend")
	}

	port := u.Port()
	if port == "" {
		if u.Scheme == schemeClickHouseSecure || isTrue(u.Query().Get("secure")) || isTrue(u.Query().Get("tls")) {
			port = "9440"
		} else {
			port = "9000"
		}
	}

	hostPort := net.JoinHostPort(host, port)

	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		database = "default"
	}

	username := "default"
	password := ""

	if u.User != nil {
		if user := u.User.Username(); user != "" {
			username = user
		}

		if pass, ok := u.User.Password(); ok {
			password = pass
		}
	}

	if user := u.Query().Get("username"); user != "" {
		username = user
	}

	if pass := u.Query().Get("password"); pass != "" {
		password = pass
	}

	if db := u.Query().Get("database"); db != "" {
		database = db
	}

	q := u.Query()

	tlsConfig, err := resolveTLSConfig(u, q, tlsCfg)
	if err != nil {
		return ch.Options{}, fmt.Errorf("resolving TLS config: %w", err)
	}

	return ch.Options{
		Address:     hostPort,
		Database:    database,
		User:        username,
		Password:    password,
		DialTimeout: dialTimeout,
		ReadTimeout: readTimeout,
		TLS:         tlsConfig,
	}, nil
}

// resolveTLSConfig determines the TLS configuration for the connection.
// TLS is enabled if the DSN scheme is "clickhouses", query params contain
// secure=true or tls=true, or the TLS config has Enabled set. When TLS is
// triggered by the DSN but no explicit TLS config is provided, a minimal
// TLS 1.2 config is used for backward compatibility.
func resolveTLSConfig(
	u *url.URL,
	q url.Values,
	tlsCfg *xtls.Config,
) (*tls.Config, error) {
	dsnWantsTLS := isTrue(q.Get("secure")) ||
		isTrue(q.Get("tls")) ||
		u.Scheme == schemeClickHouseSecure

	// If the explicit config has TLS enabled, use it — it takes priority.
	if tlsCfg != nil && tlsCfg.Enabled {
		cfg, err := tlsCfg.Build()
		if err != nil {
			return nil, err
		}

		return cfg, nil
	}

	// DSN-triggered TLS with no explicit config: bare TLS 1.2 (backward
	// compatible).
	if dsnWantsTLS {
		return &tls.Config{
			MinVersion: tls.VersionTLS12,
		}, nil
	}

	return nil, nil //nolint:nilnil // nil TLS config means plaintext
}

func isTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

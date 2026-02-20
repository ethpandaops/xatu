package clickhouse

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go"
)

func parseChGoOptions(dsn string) (ch.Options, error) {
	if !strings.Contains(dsn, "://") {
		dsn = "clickhouse://" + dsn
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return ch.Options{}, err
	}

	switch u.Scheme {
	case "http", "https":
		return ch.Options{}, fmt.Errorf("ch-go backend supports native tcp only, got DSN scheme %q", u.Scheme)
	case "clickhouse", "tcp", "clickhouses":
	default:
		return ch.Options{}, fmt.Errorf("unsupported DSN scheme %q for ch-go backend", u.Scheme)
	}

	hostPort := u.Host
	if hostPort == "" {
		return ch.Options{}, fmt.Errorf("missing host in DSN")
	}

	if strings.Contains(hostPort, ",") {
		return ch.Options{}, fmt.Errorf("multiple hosts are not supported by ch-go backend")
	}

	if _, _, splitErr := net.SplitHostPort(hostPort); splitErr != nil {
		hostPort = net.JoinHostPort(hostPort, "9000")
	}

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

	q := u.Query()
	if user := q.Get("username"); user != "" {
		username = user
	}

	if pass := q.Get("password"); pass != "" {
		password = pass
	}

	if db := q.Get("database"); db != "" {
		database = db
	}

	var tlsConfig *tls.Config
	if isTrue(q.Get("secure")) || isTrue(q.Get("tls")) || u.Scheme == "clickhouses" {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return ch.Options{
		Address:     hostPort,
		Database:    database,
		User:        username,
		Password:    password,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 30 * time.Second,
		TLS:         tlsConfig,
	}, nil
}

func isTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

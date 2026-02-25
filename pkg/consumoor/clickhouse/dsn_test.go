package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChGoOptions(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		wantAddr string
		wantDB   string
		wantUser string
		wantPass string
		wantTLS  bool
		wantErr  string
	}{
		// --- Schemes ---
		{
			name:     "clickhouse scheme",
			dsn:      "clickhouse://localhost:9000/mydb",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  false,
		},
		{
			name:     "tcp scheme",
			dsn:      "tcp://localhost:9000/mydb",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  false,
		},
		{
			name:     "clickhouses scheme enables TLS",
			dsn:      "clickhouses://localhost:9440/mydb",
			wantAddr: "localhost:9440",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  true,
		},
		{
			name:     "no scheme defaults to clickhouse",
			dsn:      "localhost:9000/mydb",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  false,
		},
		{
			name:    "http scheme rejected",
			dsn:     "http://localhost:8123/mydb",
			wantErr: "native tcp only",
		},
		{
			name:    "https scheme rejected",
			dsn:     "https://localhost:8443/mydb",
			wantErr: "native tcp only",
		},
		{
			name:    "unsupported scheme rejected",
			dsn:     "ftp://localhost/mydb",
			wantErr: "unsupported DSN scheme",
		},

		// --- Default values ---
		{
			name:     "default port 9000 for clickhouse scheme",
			dsn:      "clickhouse://localhost/mydb",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  false,
		},
		{
			name:     "default database when path is empty",
			dsn:      "clickhouse://localhost:9000",
			wantAddr: "localhost:9000",
			wantDB:   "default",
			wantUser: "default",
			wantTLS:  false,
		},

		// --- TLS trigger combinations ---
		{
			name:     "secure=true enables TLS",
			dsn:      "clickhouse://localhost:9440/mydb?secure=true",
			wantAddr: "localhost:9440",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  true,
		},
		{
			name:     "tls=true enables TLS",
			dsn:      "clickhouse://localhost:9440/mydb?tls=true",
			wantAddr: "localhost:9440",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  true,
		},
		{
			name:     "secure=1 enables TLS",
			dsn:      "clickhouse://localhost:9440/mydb?secure=1",
			wantAddr: "localhost:9440",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  true,
		},
		{
			name:     "tls=yes enables TLS",
			dsn:      "clickhouse://localhost:9440/mydb?tls=yes",
			wantAddr: "localhost:9440",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  true,
		},
		{
			name:     "secure=false no TLS",
			dsn:      "clickhouse://localhost:9000/mydb?secure=false",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "default",
			wantTLS:  false,
		},

		// --- Credential extraction ---
		{
			name:     "userinfo credentials",
			dsn:      "clickhouse://admin:secret@localhost:9000/mydb",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "admin",
			wantPass: "secret",
			wantTLS:  false,
		},
		{
			name:     "query param credentials",
			dsn:      "clickhouse://localhost:9000/mydb?username=quser&password=qpass",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "quser",
			wantPass: "qpass",
			wantTLS:  false,
		},
		{
			name:     "query params override userinfo",
			dsn:      "clickhouse://admin:secret@localhost:9000/mydb?username=override&password=overpass",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "override",
			wantPass: "overpass",
			wantTLS:  false,
		},
		{
			name:     "userinfo username only with empty password",
			dsn:      "clickhouse://admin@localhost:9000/mydb",
			wantAddr: "localhost:9000",
			wantDB:   "mydb",
			wantUser: "admin",
			wantPass: "",
			wantTLS:  false,
		},

		// --- Database extraction ---
		{
			name:     "database from path",
			dsn:      "clickhouse://localhost:9000/production",
			wantAddr: "localhost:9000",
			wantDB:   "production",
			wantUser: "default",
			wantTLS:  false,
		},
		{
			name:     "database query param overrides path",
			dsn:      "clickhouse://localhost:9000/pathdb?database=querydb",
			wantAddr: "localhost:9000",
			wantDB:   "querydb",
			wantUser: "default",
			wantTLS:  false,
		},

		// --- Error cases ---
		{
			name:    "missing host",
			dsn:     "clickhouse:///mydb",
			wantErr: "missing host",
		},
		{
			name:    "multiple hosts rejected",
			dsn:     "clickhouse://host1:9000,host2:9000/mydb",
			wantErr: "multiple hosts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseChGoOptions(tt.dsn, 5*time.Second, 30*time.Second, nil)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAddr, opts.Address)
			assert.Equal(t, tt.wantDB, opts.Database)
			assert.Equal(t, tt.wantUser, opts.User)
			assert.Equal(t, tt.wantPass, opts.Password)

			if tt.wantTLS {
				assert.NotNil(t, opts.TLS, "expected TLS config")
			} else {
				assert.Nil(t, opts.TLS, "expected no TLS config")
			}
		})
	}
}

func TestIsTrue(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{" true ", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isTrue(tt.input))
		})
	}
}

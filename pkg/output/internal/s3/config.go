// Package s3 wraps an S3-compatible object store client (AWS S3,
// Cloudflare R2, MinIO, etc.) with the minimal surface needed by xatu
// output sinks: connect, PutObject. Higher-level sinks layer their own
// event-shape semantics on top.
package s3

import (
	"errors"
)

// Config configures an S3-compatible client. It is YAML-inlineable so
// higher-level sinks can embed it directly under their own `config:`
// block.
type Config struct {
	// Endpoint is the S3 endpoint host. For Cloudflare R2 use the
	// account-scoped endpoint (e.g. "<account>.r2.cloudflarestorage.com").
	// For AWS use the regional endpoint (e.g. "s3.us-east-1.amazonaws.com").
	// Must NOT include a scheme — set Insecure to toggle HTTPS/HTTP.
	Endpoint string `yaml:"endpoint"`

	// Bucket is the destination bucket name. Must already exist.
	Bucket string `yaml:"bucket"`

	// Region is the bucket region. Required by AWS, ignored by R2/MinIO
	// but harmless. Defaults to "auto" which works for R2.
	Region string `yaml:"region" default:"auto"`

	// AccessKeyID and SecretAccessKey authenticate the client. Required.
	AccessKeyID     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`

	// Insecure disables HTTPS (plain HTTP). Defaults to false — i.e. SSL
	// is used. Modeled as opt-in plain-HTTP rather than opt-in SSL because
	// `bool` zero-value defaulting traps the opposite shape: an explicit
	// `useSsl: false` would be indistinguishable from "unset" and get
	// overridden back to true by the defaults loader.
	Insecure bool `yaml:"insecure"`
}

// Validate checks for the minimum required fields.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}

	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	if c.Bucket == "" {
		return errors.New("bucket is required")
	}

	if c.AccessKeyID == "" {
		return errors.New("accessKeyId is required")
	}

	if c.SecretAccessKey == "" {
		return errors.New("secretAccessKey is required")
	}

	return nil
}

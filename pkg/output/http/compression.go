package http

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"net/http"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
)

type CompressionStrategy string

var (
	CompressionStrategyNone   CompressionStrategy = "none"
	CompressionStrategyGzip   CompressionStrategy = "gzip"
	CompressionStrategyZstd   CompressionStrategy = "zstd"
	CompressionStrategyZlib   CompressionStrategy = "zlib"
	CompressionStrategySnappy CompressionStrategy = "snappy"
)

type Compressor struct {
	Strategy CompressionStrategy
}

func (c *Compressor) Compress(in *bytes.Buffer) (*bytes.Buffer, error) {
	switch c.Strategy {
	case CompressionStrategyGzip:
		return c.gzipCompress(in)
	case CompressionStrategyZstd:
		return c.zstdCompress(in)
	case CompressionStrategyZlib:
		return c.zlibCompress(in)
	case CompressionStrategySnappy:
		return c.snappyCompress(in)
	default:
		return in, nil
	}
}

func (c *Compressor) gzipCompress(in *bytes.Buffer) (*bytes.Buffer, error) {
	out := &bytes.Buffer{}
	g := gzip.NewWriter(out)

	_, err := g.Write(in.Bytes())
	if err != nil {
		return out, err
	}

	if err := g.Close(); err != nil {
		return out, err
	}

	return out, nil
}

func (c *Compressor) zstdCompress(in *bytes.Buffer) (*bytes.Buffer, error) {
	out := &bytes.Buffer{}

	z, err := zstd.NewWriter(out)
	if err != nil {
		return out, err
	}

	_, err = z.Write(in.Bytes())
	if err != nil {
		return out, err
	}

	if err := z.Close(); err != nil {
		return out, err
	}

	return out, nil
}

func (c *Compressor) zlibCompress(in *bytes.Buffer) (*bytes.Buffer, error) {
	out := &bytes.Buffer{}
	z := zlib.NewWriter(out)

	_, err := z.Write(in.Bytes())
	if err != nil {
		return out, err
	}

	if err := z.Close(); err != nil {
		return out, err
	}

	return out, nil
}

func (c *Compressor) snappyCompress(in *bytes.Buffer) (*bytes.Buffer, error) {
	compressed := snappy.Encode(nil, in.Bytes())

	return bytes.NewBuffer(compressed), nil
}

func (c *Compressor) AddHeaders(req *http.Request) {
	switch c.Strategy {
	case CompressionStrategyGzip:
		req.Header.Set("Content-Encoding", "gzip")
	case CompressionStrategyZstd:
		req.Header.Set("Content-Encoding", "zstd")
	case CompressionStrategyZlib:
		req.Header.Set("Content-Encoding", "deflate")
	case CompressionStrategySnappy:
		req.Header.Set("Content-Encoding", "snappy")
	}
}

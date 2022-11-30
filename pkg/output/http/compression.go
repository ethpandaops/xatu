package http

type CompressionStrategy string

var (
	CompressionStrategyNone CompressionStrategy = "none"
	CompressionStrategyGzip CompressionStrategy = "gzip"
)

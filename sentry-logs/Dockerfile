FROM timberio/vector:0.44.0-distroless-libc

ARG VERSION=dev
ENV XATU_VERSION=${VERSION}

# Copy our configs to a clean directory (avoids loading base image's vector.yaml)
COPY configs/vector.yaml /etc/xatu/vector.yaml
COPY configs/sources.d/ /etc/xatu/sources.d/
COPY proto/xatu.desc /etc/xatu/proto/xatu.desc

# Required environment variables (crashes on startup if missing):
#   XATU_CLIENT_NAME - Name of this client instance
#   XATU_NETWORK_NAME - Ethereum network name e.g. mainnet, holesky
#   XATU_SERVER_URL - Xatu server HTTP endpoint
#   XATU_AUTH - Base64-encoded username:password for authentication
#
# Optional environment variables:
#   XATU_VERSION - Version string (default: set at build time)
#   XATU_NETWORK_ID - Ethereum network ID as integer (default: 0)
#   XATU_COMPRESSION - Compression algorithm: gzip, none (default: gzip)
#   XATU_BATCH_MAX_EVENTS - Max events per batch (default: 5000)
#   XATU_BATCH_TIMEOUT_SECS - Batch timeout in seconds (default: 5)

ENTRYPOINT ["vector", "--config-dir", "/etc/xatu/", "--config-dir", "/etc/xatu/sources.d/"]

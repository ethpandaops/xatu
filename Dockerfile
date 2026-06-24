FROM golang:1.26.2 AS builder
WORKDIR /src
COPY go.sum go.mod ./
RUN go mod download
COPY . .
# Version metadata is injected at release time (see .github/workflows/goreleaser.yaml);
# local/CI builds fall back to dev defaults. GOOS/GOARCH come from BuildKit's
# predefined platform args so the multi-arch release build stamps the right
# values under QEMU emulation.
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG GIT_COMMIT=unknown
RUN go build \
  -ldflags="-s -w \
  -X github.com/ethpandaops/xatu/pkg/proto/xatu.Release=${VERSION} \
  -X github.com/ethpandaops/xatu/pkg/proto/xatu.GitCommit=${GIT_COMMIT} \
  -X github.com/ethpandaops/xatu/pkg/proto/xatu.GOOS=${TARGETOS} \
  -X github.com/ethpandaops/xatu/pkg/proto/xatu.GOARCH=${TARGETARCH}" \
  -o /bin/app .

# cryo is the execution-layer extractor used by EL cannon. We build from git
# (matching ethpandaops/base-images clickhouse-tools, the source of the legacy
# pipeline's data): master's `blocks` dataset exposes gas_limit, which the
# 0.3.2 crates.io release omits. cryo's column output is the schema contract for
# the canonical_execution_* routes, and the state-read datasets' semantics vary
# across cryo versions, so we PIN to a specific commit for reproducible builds
# and stable data. This ARG line is the single place the cryo ref is pinned:
# every build path (docker build, docker compose, `make docker`, and the
# release cryo image) builds this Dockerfile, so bump it here to upgrade cryo
# everywhere. Override with --build-arg CRYO_GIT_REF=<ref> for ad-hoc testing.
FROM rust:1-bookworm AS cryo-builder
ARG CRYO_GIT_REF=559b65455d7ef6b03e8e9e96a0e50fd4fe8a9c86
RUN cargo install --git https://github.com/paradigmxyz/cryo --rev "${CRYO_GIT_REF}" --locked cryo_cli

FROM ubuntu:latest
RUN apt-get update && apt-get -y upgrade && apt-get install -y --no-install-recommends \
  libssl-dev \
  ca-certificates \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/app /xatu
COPY --from=cryo-builder /usr/local/cargo/bin/cryo /usr/local/bin/cryo
EXPOSE 5555
ENTRYPOINT ["/xatu"]

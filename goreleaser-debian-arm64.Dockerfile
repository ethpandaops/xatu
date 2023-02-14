FROM debian:latest
RUN dpkg --add-architecture arm64
RUN apt-get update && apt-get -y upgrade && apt-get install -y --no-install-recommends \
  libssl-dev:arm64 \
  ca-certificates:arm64 \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
COPY xatu* /xatu
ENTRYPOINT ["/xatu"]

FROM golang:1.25 AS builder
WORKDIR /src

# Copy sibling dependencies that are referenced in go.mod replace directives
COPY tysm/ /tysm/
COPY hermes/ /hermes/

# Copy xatu project files
COPY xatu/go.sum xatu/go.mod ./
RUN go mod download
COPY xatu/ .
RUN go build -o /bin/app .

FROM ubuntu:latest
RUN apt-get update && apt-get -y upgrade && apt-get install -y --no-install-recommends \
  libssl-dev \
  ca-certificates \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/app /xatu
EXPOSE 5555
ENTRYPOINT ["/xatu"]

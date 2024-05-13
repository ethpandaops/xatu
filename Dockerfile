FROM golang:1.22 AS builder
WORKDIR /src
COPY go.sum go.mod ./
ARG GOPROXY
ENV GOPROXY=${GOPROXY}
RUN go mod download -x
COPY . .
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
